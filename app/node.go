package app

import (
	"fmt"
	"time"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type V2 = rl.Vector2

func SV2(s *Serializer, v *V2) {
	SFloat(s, &v.X)
	SFloat(s, &v.Y)
}

type Node struct {
	ID     int
	Pos    V2
	Name   string
	Pinned bool

	InputPorts  []NodePort
	OutputPorts []NodePort

	Action NodeAction
	Valid  bool

	Running bool
	done    chan struct{}

	ResultAvailable bool
	Result          NodeActionResult

	InputPortPositions  []V2
	OutputPortPositions []V2
	DragRect            rl.Rectangle

	// Map from output port name to output state. Names are used because they are
	// more stable than port number in the face of polymorhpic nodes.
	//
	// (It is an open design question whether we want any of this state to reset
	// when re-running the node. For now I say no resets.)
	outputState map[string]*NodeOutputState
}

var _ Serializable = &Node{}

func (n *Node) Serialize(s *Serializer) bool {
	SInt(s, &n.ID)
	SV2(s, &n.Pos)
	SStr(s, &n.Name)
	SBool(s, &n.Pinned)

	SSlice(s, &n.InputPorts)
	SSlice(s, &n.OutputPorts)

	if s.Encode {
		s.WriteStr(n.Action.Tag())
		n.Action.Serialize(s)
	} else {
		tag, ok := s.ReadStr()
		if !ok {
			return false
		}
		meta := GetNodeActionMeta(tag)
		n.Action = meta.Alloc()
		n.Action.Serialize(s)
	}

	// The remainder of the fields are dynamic and need not be serialized.

	return s.Ok()
}

func (n *Node) String() string {
	return fmt.Sprintf("Node#%d(%s)", n.ID, n.Name)
}

func (n *Node) ClayID() clay.ElementID {
	return clay.IDI("Node", n.ID)
}

func (n *Node) DragHandleClayID() clay.ElementID {
	return clay.IDI("NodeDragHandle", n.ID)
}

func (n *Node) DragKey() string {
	return fmt.Sprintf("Node#%d", n.ID)
}

func (n *Node) GetOutputState(portName string) *NodeOutputState {
	if existing, ok := n.outputState[portName]; ok {
		return existing
	}
	newState := &NodeOutputState{}
	if n.outputState == nil {
		n.outputState = make(map[string]*NodeOutputState)
	}
	n.outputState[portName] = newState
	return newState
}

type NodePort struct {
	Name string
	Type FlowType
}

var _ Serializable = &NodePort{}

func (np *NodePort) Serialize(s *Serializer) bool {
	SStr(s, &np.Name)
	SThing(s, &np.Type)
	return s.Ok()
}

type NodeOutputState struct {
	Collapsed bool
}

type Wire struct {
	StartNode, EndNode *Node
	StartPort, EndPort int
}

func (w *Wire) Type() FlowType {
	return w.StartNode.OutputPorts[w.StartPort].Type
}

func (n *Node) Run(rerunInputs bool) <-chan struct{} {
	if n.Running {
		fmt.Printf("Node %s is already running; starting another done-er\n", n)

		done := n.done
		if done == nil {
			// This is possible since we race with n.Running. If this happens, just
			// send a new channel that is already complete.
			//
			// TODO: I've not yet seen this happen.
			fmt.Printf("Node %s lost the race with n.Running...we're just done\n", n)
			go func() {
				done <- struct{}{}
			}()
			return done
		} else {
			go func() {
				// oh the hacks
				for {
					if n.Running {
						time.Sleep(1 * time.Millisecond)
					} else {
						break
					}
				}
				done <- struct{}{}
				fmt.Printf("Node %s is no longer running, extra done-er is done\n", n)
			}()
		}
		return done
	}

	fmt.Printf("Running node %s\n", n)
	n.Running = true
	n.ResultAvailable = false
	n.done = make(chan struct{})

	go func() {
		// Wait on input ports
		var inputRuns []<-chan struct{}
		for _, inputNode := range NodeInputs(n) {
			rerunThisNode := rerunInputs && !inputNode.Pinned
			if rerunThisNode || !inputNode.ResultAvailable {
				fmt.Printf("Node %s wants node %s to run\n", n, inputNode)
				inputRuns = append(inputRuns, inputNode.Run(rerunInputs))
			}
		}
		for _, inputRun := range inputRuns {
			<-inputRun
		}

		// If any inputs have errors, stop.
		for _, inputNode := range NodeInputs(n) {
			if !inputNode.ResultAvailable || inputNode.Result.Err != nil {
				n.Running = false
				n.done <- struct{}{}
				n.done = nil
				return
			}
		}

		fmt.Printf("Node %s: all inputs are done\n", n)

		// Run action
		res := <-n.Action.Run(n)
		if res.Err == nil && len(res.Outputs) != len(n.OutputPorts) {
			panic(fmt.Errorf("bad num outputs for %s: got %d, expected %d", n, len(n.OutputPorts), len(res.Outputs)))
		}
		for i, output := range res.Outputs {
			if err := Typecheck(*output.Type, n.OutputPorts[i].Type); err != nil {
				panic(fmt.Errorf("bad value type for %s output port %d: %v", n, i, err))
			}
		}
		n.Result = res
		n.Running = false
		n.ResultAvailable = true

		n.done <- struct{}{}
		n.done = nil
	}()

	return n.done
}

func (n *Node) ClearResult() {
	n.ResultAvailable = false
	n.Result = NodeActionResult{}
}

func (n *Node) GetInputWire(port int) (*Wire, bool) {
	for _, wire := range wires {
		if wire.EndNode == n && wire.EndPort == port {
			return wire, true
		}
	}
	return nil, false
}

func (n *Node) InputIsWired(port int) bool {
	_, res := n.GetInputWire(port)
	return res
}

func (n *Node) GetInputValue(port int) (FlowValue, bool, error) {
	if port >= len(n.InputPorts) {
		panic(fmt.Errorf("node %s has no port %d", n, port))
	}

	for _, wire := range wires {
		if wire.EndNode == n && wire.EndPort == port {
			wireValue, ok := wire.StartNode.GetOutputValue(wire.StartPort)
			if !ok {
				return FlowValue{}, false, nil
			}
			if err := Typecheck(*wireValue.Type, n.InputPorts[port].Type); err != nil {
				return wireValue, true, fmt.Errorf("on input port %d: %v", port, err)
			}
			return wireValue, true, nil
		}
	}
	return FlowValue{}, false, nil
}

func (n *Node) GetOutputValue(port int) (FlowValue, bool) {
	if port >= len(n.OutputPorts) {
		panic(fmt.Errorf("node %s has no port %d", n, port))
	}

	if !n.ResultAvailable {
		return FlowValue{}, false
	}
	if len(n.OutputPorts) != len(n.Result.Outputs) {
		panic(fmt.Errorf("incorrect number of output values for %s: got %d, expected %d", n, len(n.Result.Outputs), len(n.OutputPorts)))
	}
	return n.Result.Outputs[port], true
}

// Update cached positions and rectangles and so on based on layout
func (n *Node) UpdateLayoutInfo() {
	n.InputPortPositions = make([]V2, len(n.InputPorts))
	n.OutputPortPositions = make([]V2, len(n.OutputPorts))

	nodeData, ok := clay.GetElementData(n.ClayID())
	if !ok {
		// This node has not been rendered yet. That's fine. Maybe it was just added.
		return
	}

	bboxNode := nodeData.BoundingBox
	for i := range n.InputPorts {
		if portData, ok := clay.GetElementData(PortAnchorID(n, false, i)); ok {
			bboxPort := portData.BoundingBox
			n.InputPortPositions[i] = V2{bboxNode.X, bboxPort.Y}
		}
	}
	for i := range n.OutputPorts {
		if portData, ok := clay.GetElementData(PortAnchorID(n, true, i)); ok {
			bboxPort := portData.BoundingBox
			n.OutputPortPositions[i] = V2{bboxNode.X + bboxNode.Width, bboxPort.Y}
		}
	}

	n.DragRect = rl.Rectangle(util.Must1B(clay.GetElementData(n.DragHandleClayID())).BoundingBox)
}

// All implementations of NodeAction should be marked with `GEN:NodeAction` in
// a comment, in order to be picked up by go:generate.
type NodeAction interface {
	UpdateAndValidate(n *Node)
	UI(n *Node)
	Run(n *Node) <-chan NodeActionResult
	// TODO: Cancellation!
	Tag() string // This is implemented automatically by go:generate.
	Serializable
}

type NodeActionResult struct {
	Outputs []FlowValue
	Err     error
}

type NodeActionMeta struct {
	Tag   string
	Alloc func() NodeAction
}

func GetNodeActionMeta(tag string) NodeActionMeta {
	for _, meta := range allNodeActions {
		if tag == meta.Tag {
			return meta
		}
	}
	panic("unknown node action type; make sure to run go:generate")
}

// See node_actions_gen.go for the definition of allNodeActions.

var nodeID = 0

func NewNodeID() int {
	nodeID++
	return nodeID
}
