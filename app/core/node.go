package core

import (
	"context"
	"fmt"
	"runtime/debug"
	"slices"
	"sync"

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

	Action              NodeAction
	Valid               bool
	HandleSkippedInputs bool

	// Runtime state
	mu      sync.Mutex
	Running bool
	done    chan struct{}

	resultAvailable bool
	result          NodeActionResult

	InputPortPositions  []V2
	OutputPortPositions []V2
	DragRect            rl.Rectangle

	Graph *Graph

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

func (n *Node) Clone() *Node {
	newNode := &Node{
		ID:                  n.ID,
		Pos:                 n.Pos,
		Name:                n.Name,
		Pinned:              n.Pinned,
		Action:              n.Action,
		Valid:               n.Valid,
		HandleSkippedInputs: n.HandleSkippedInputs,
		// Runtime state (mu, Running, done, resultAvailable, result) is strictly ignored/zeroed
		InputPortPositions:  nil, // Will be recalculated
		OutputPortPositions: nil, // Will be recalculated
		DragRect:            n.DragRect,
		Graph:               n.Graph, // Will be updated by caller if needed
		outputState:         nil,     // UI state reset
	}

	if n.InputPorts != nil {
		newNode.InputPorts = make([]NodePort, len(n.InputPorts))
		copy(newNode.InputPorts, n.InputPorts)
	}
	if n.OutputPorts != nil {
		newNode.OutputPorts = make([]NodePort, len(n.OutputPorts))
		copy(newNode.OutputPorts, n.OutputPorts)
	}

	return newNode
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

func (n *Node) Run(ctx context.Context, rerunInputs bool) <-chan struct{} {
	n.mu.Lock()
	if n.Running {
		fmt.Printf("Node %s is already running; joining existing run\n", n)
		if n.done == nil {
			// This should theoretically not happen if n.Running is true,
			// but if it does, return a closed channel.
			n.mu.Unlock()
			c := make(chan struct{})
			close(c)
			return c
		}
		done := n.done
		n.mu.Unlock()
		return done
	}

	fmt.Printf("Running node %s\n", n)
	n.Running = true
	n.resultAvailable = false
	n.done = make(chan struct{})
	// We need to keep n.done in a local variable to return it,
	// because we unlock before returning.
	doneCh := n.done
	n.mu.Unlock()

	go func() {
		defer func() {
			n.mu.Lock()
			defer n.mu.Unlock()

			if r := recover(); r != nil {
				stack := debug.Stack()
				var err error
				if e, ok := r.(error); ok {
					err = fmt.Errorf("panic: %w\n%s", e, stack)
				} else {
					err = fmt.Errorf("panic: %v\n%s", r, stack)
				}
				n.result = NodeActionResult{Err: err}
				n.resultAvailable = true
			}
			n.Running = false
			if n.done != nil {
				close(n.done)
			}
		}()

		// Check context before starting
		if ctx.Err() != nil {
			n.mu.Lock()
			n.result = NodeActionResult{Err: ctx.Err()}
			n.resultAvailable = true
			n.mu.Unlock()
			return
		}

		// Wait on input ports
		var inputRuns []<-chan struct{}
		for _, inputNode := range n.Inputs() {
			rerunThisNode := rerunInputs && !inputNode.Pinned
			// Use thread-safe check
			if rerunThisNode || !inputNode.IsResultAvailable() {
				fmt.Printf("Node %s wants node %s to run\n", n, inputNode)
				inputRuns = append(inputRuns, inputNode.Run(ctx, rerunInputs))
			}
		}
		for _, inputRun := range inputRuns {
			select {
			case <-inputRun:
			case <-ctx.Done():
				n.mu.Lock()
				n.result = NodeActionResult{Err: ctx.Err()}
				n.resultAvailable = true
				n.mu.Unlock()
				return
			}
		}

		// If any inputs have errors, stop.
		for _, inputNode := range n.Inputs() {
			res, ok := inputNode.GetResult()
			if !ok {
				n.mu.Lock()
				n.result = NodeActionResult{Err: fmt.Errorf("input node %s produced no result", inputNode)}
				n.resultAvailable = true
				n.mu.Unlock()
				return
			}
			if res.Err != nil {
				n.mu.Lock()
				n.result = NodeActionResult{Err: fmt.Errorf("input node %s failed: %w", inputNode, res.Err)}
				n.resultAvailable = true
				n.mu.Unlock()
				return
			}
		}

		// Check for skipped inputs
		anySkipped := false
		for i := range n.InputPorts {
			if val, ok, _ := n.GetInputValue(i); ok && val.Skipped {
				anySkipped = true
				break
			}
		}

		if anySkipped {
			fmt.Printf("Node %s: skipping execution due to skipped input\n", n)
			outputs := make([]FlowValue, len(n.OutputPorts))
			for i := range outputs {
				outputs[i] = FlowValue{Type: &n.OutputPorts[i].Type, Skipped: true}
			}

			n.mu.Lock()
			n.result = NodeActionResult{Outputs: outputs}
			n.resultAvailable = true
			n.mu.Unlock()
			return
		}

		fmt.Printf("Node %s: all inputs are done\n", n)

		// Run action
		var resCh <-chan NodeActionResult
		if actionCtx, ok := n.Action.(NodeActionWithContext); ok {
			resCh = actionCtx.RunContext(ctx, n)
		} else {
			resCh = n.Action.Run(n)
		}

		select {
		case res := <-resCh:
			if res.Err == nil && len(res.Outputs) != len(n.OutputPorts) {
				n.mu.Lock()
				n.result = NodeActionResult{Err: fmt.Errorf("bad num outputs for %s: got %d, expected %d", n, len(n.OutputPorts), len(res.Outputs))}
				n.resultAvailable = true
				n.mu.Unlock()
				return
			}
			for i, output := range res.Outputs {
				if err := Typecheck(*output.Type, n.OutputPorts[i].Type); err != nil {
					n.mu.Lock()
					n.result = NodeActionResult{Err: fmt.Errorf("bad value type for %s output port %d: %v", n, i, err)}
					n.resultAvailable = true
					n.mu.Unlock()
					return
				}
			}
			n.mu.Lock()
			n.result = res
			n.resultAvailable = true
			n.mu.Unlock()
		case <-ctx.Done():
			n.mu.Lock()
			n.result = NodeActionResult{Err: ctx.Err()}
			n.resultAvailable = true
			n.mu.Unlock()
		}
	}()

	return doneCh
}

func (n *Node) IsResultAvailable() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.resultAvailable
}

func (n *Node) GetResult() (NodeActionResult, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.result, n.resultAvailable
}

func (n *Node) SetResult(res NodeActionResult) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.result = res
	n.resultAvailable = true
}

func (n *Node) ClearResult() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Close any open streams in the result
	if n.resultAvailable {
		for _, output := range n.result.Outputs {
			if output.Type != nil && output.Type.Kind == FSKindStream && output.StreamValue != nil {
				_ = output.StreamValue.Close()
			}
		}
	}

	n.resultAvailable = false
	n.result = NodeActionResult{}
}

func (n *Node) Inputs() []*Node {
	var res []*Node
	if n.Graph == nil {
		return nil
	}
	for _, wire := range n.Graph.Wires {
		if wire.EndNode == n && !slices.Contains(res, wire.StartNode) {
			res = append(res, wire.StartNode)
		}
	}
	return res
}

func (n *Node) GetInputWire(port int) (*Wire, bool) {
	if n.Graph == nil {
		return nil, false
	}
	for _, wire := range n.Graph.Wires {
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
		return FlowValue{}, false, fmt.Errorf("node %s has no input port %d", n, port)
	}

	if n.Graph == nil {
		return FlowValue{}, false, nil
	}

	for _, wire := range n.Graph.Wires {
		if wire.EndNode == n && wire.EndPort == port {
			wireValue, ok, err := wire.StartNode.GetOutputValue(wire.StartPort)
			if err != nil {
				return FlowValue{}, false, fmt.Errorf("failed to get output value from upstream node %s: %w", wire.StartNode, err)
			}
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

func (n *Node) GetOutputValue(port int) (FlowValue, bool, error) {
	if port >= len(n.OutputPorts) {
		return FlowValue{}, false, fmt.Errorf("node %s has no output port %d", n, port)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.resultAvailable {
		return FlowValue{}, false, nil
	}
	if len(n.OutputPorts) != len(n.result.Outputs) {
		return FlowValue{}, false, fmt.Errorf("incorrect number of output values for %s: got %d, expected %d", n, len(n.result.Outputs), len(n.OutputPorts))
	}
	return n.result.Outputs[port], true, nil
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
			n.InputPortPositions[i] = V2{X: bboxNode.X, Y: bboxPort.Y}
		}
	}
	for i := range n.OutputPorts {
		if portData, ok := clay.GetElementData(PortAnchorID(n, true, i)); ok {
			bboxPort := portData.BoundingBox
			n.OutputPortPositions[i] = V2{X: bboxPort.X + bboxPort.Width/2, Y: bboxPort.Y + bboxPort.Height/2}
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
	Tag() string // This is implemented automatically by go:generate.
	Serializable
}

type NodeActionWithContext interface {
	RunContext(ctx context.Context, n *Node) <-chan NodeActionResult
}

type NodeActionResult struct {
	Outputs []FlowValue
	Err     error
}

type NodeActionMeta struct {
	Tag   string
	Alloc func() NodeAction
}

func PortAnchorID(n *Node, output bool, index int) clay.ElementID {
	if output {
		return clay.IDI("NodeOutPortAnchor", n.ID*1000+index)
	}
	return clay.IDI("NodeInPortAnchor", n.ID*1000+index)
}

func Toposort(nodes []*Node, wires []*Wire) ([]*Node, error) {
	nodeMap := make(map[int]*Node)
	inDegree := make(map[int]int)
	adj := make(map[int][]int)

	for _, n := range nodes {
		nodeMap[n.ID] = n
		inDegree[n.ID] = 0
	}

	for _, w := range wires {
		if _, ok := nodeMap[w.StartNode.ID]; !ok {
			continue
		}
		if _, ok := nodeMap[w.EndNode.ID]; !ok {
			continue
		}

		adj[w.StartNode.ID] = append(adj[w.StartNode.ID], w.EndNode.ID)
		inDegree[w.EndNode.ID]++
	}

	var queue []*Node
	for _, n := range nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n)
		}
	}

	var result []*Node
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		result = append(result, u)

		for _, vID := range adj[u.ID] {
			inDegree[vID]--
			if inDegree[vID] == 0 {
				if v, ok := nodeMap[vID]; ok {
					queue = append(queue, v)
				}
			}
		}
	}

	if len(result) != len(nodes) {
		return nil, fmt.Errorf("cycle detected")
	}

	return result, nil
}
