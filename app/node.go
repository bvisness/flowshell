package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type V2 = rl.Vector2

type Node struct {
	ID   int
	Pos  V2
	Name string

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

type NodePort struct {
	Name string
	Type FlowType
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
		return n.done
	}

	n.Running = true
	n.ResultAvailable = false
	n.done = make(chan struct{}, 1)

	go func() {
		// Wait on input ports
		var inputRuns []<-chan struct{}
		for _, inputNode := range NodeInputs(n) {
			if rerunInputs || !inputNode.ResultAvailable {
				inputRuns = append(inputRuns, inputNode.Run(rerunInputs))
			}
		}
		for _, inputRun := range inputRuns {
			<-inputRun
		}

		// If any inputs have errors, stop.
		for _, inputNode := range NodeInputs(n) {
			if !inputNode.ResultAvailable {
				panic(fmt.Errorf("input node %s should have had a result", inputNode))
			}
			if inputNode.Result.Err != nil {
				n.Running = false
				n.done <- struct{}{}
				n.done = nil
				return
			}
		}

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

type NodeAction interface {
	UpdateAndValidate(n *Node)
	UI(n *Node)
	Run(n *Node) <-chan NodeActionResult
	// TODO: Cancellation!
}

type NodeActionResult struct {
	Outputs []FlowValue
	Err     error
}

var nodeID = 0

func NewNodeID() int {
	nodeID++
	return nodeID
}

// --------------------------------
// Load File

// TODO: Make this node polymorphic on lists of strings
// (rename to "Load Files" dynamically)
func NewLoadFileNode(path string) *Node {
	formatDropdown := UIDropdown{
		Options: []UIDropdownOption{
			{Name: "Raw bytes", Value: "raw"},
			{Name: "CSV", Value: "csv"},
			{Name: "JSON", Value: "json"},
		},
	}

	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		formatDropdown.SelectByValue(ext[1:])
	}

	return &Node{
		ID:   NewNodeID(),
		Name: "Load File",

		InputPorts: []NodePort{{
			Name: "Path",
			Type: FlowType{Kind: FSKindBytes},
		}},
		OutputPorts: []NodePort{{
			Name: "Data",
			Type: FlowType{Kind: FSKindBytes},
		}},

		Action: &LoadFileAction{
			path:       path,
			format:     formatDropdown,
			csvNumbers: true,
		},
	}
}

type LoadFileAction struct {
	path string

	format UIDropdown

	// TODO: In reality this should be a more complex thing. For now we will just
	// always parse them as floats. (The "right way" to do it would be to have
	// CSV always parse as strings, but make it clear in the UI that they are
	// strings, and then have the user convert them to numbers. Perhaps this
	// could be done with a "Convert to Number" node that works on single
	// strings, lists, records, and tables. But perhaps you'd want to be able to
	// easily apply it to specific columns of a table? Maybe implicit conversion
	// to number would be ok within the Aggregate node and other nodes that do
	// math? Who knows. Very large design space. For now we just demo by always
	// parsing as float.
	csvNumbers bool
}

var _ NodeAction = &LoadFileAction{}

func (c *LoadFileAction) UpdateAndValidate(n *Node) {
	switch c.format.GetSelectedOption().Value {
	case "raw":
		n.OutputPorts[0].Type = FlowType{Kind: FSKindBytes}
	case "csv":
		n.OutputPorts[0].Type = FlowType{Kind: FSKindTable, ContainedType: &FlowType{Kind: FSKindAny}}
	case "json":
		n.OutputPorts[0].Type = FlowType{Kind: FSKindAny}
	}

	n.Valid = true
}

func (c *LoadFileAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          GROWH,
			ChildGap:        S2,
		},
	}, func() {
		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				Sizing:         GROWH,
				ChildAlignment: YCENTER,
			},
		}, func() {
			PortAnchor(n, false, 0)
			UITextBox(clay.IDI("LoadFilePath", n.ID), &c.path, UITextBoxConfig{
				El: clay.EL{
					Layout: clay.LAY{Sizing: GROWH},
				},
				Disabled: n.InputIsWired(0),
			})
			UISpacer(clay.AUTO_ID, W2)
			UIOutputPort(n, 0)
		})

		c.format.Do(clay.AUTO_ID, UIDropdownConfig{
			El: clay.EL{
				Layout: clay.LAY{Sizing: GROWH},
			},
			OnChange: func(before, after any) {
				n.ClearResult()
			},
		})
	})
}

func (c *LoadFileAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		content, err := os.ReadFile(c.path) // TODO: Get path from port
		if err != nil {
			res.Err = err
			return
		}

		switch format := c.format.GetSelectedOption().Value; format {
		case "raw":
			res = NodeActionResult{
				Outputs: []FlowValue{NewBytesValue(content)},
			}
		case "csv":
			r := csv.NewReader(bytes.NewReader(content))
			rows, err := r.ReadAll()
			if err != nil {
				res.Err = err
				return
			}

			// Special case: if we don't even get a row, synthesize an empty table with no columns.
			if len(rows) == 0 {
				res = NodeActionResult{
					Outputs: []FlowValue{{
						Type: &FlowType{
							Kind: FSKindTable,
							ContainedType: &FlowType{
								Kind:   FSKindRecord,
								Fields: nil,
							},
						},
					}},
				}
				return
			}

			tableRecordType := FlowType{Kind: FSKindRecord}
			for _, headerField := range rows[0] {
				tableRecordType.Fields = append(tableRecordType.Fields, FlowField{
					Name: headerField,
					Type: &FlowType{Kind: util.Tern(c.csvNumbers, FSKindFloat64, FSKindBytes)},
				})
			}

			// TODO(low): Be resilient against variable numbers of fields per row, potentially
			var tableRows [][]FlowValueField
			for _, row := range rows[1:] {
				var flowRow []FlowValueField
				for col, value := range row {
					flowRow = append(flowRow, FlowValueField{
						Name:  rows[0][col],
						Value: util.Tern(c.csvNumbers, NewFloat64Value(util.Must1(strconv.ParseFloat(value, 64)), 0), NewStringValue(value)),
					})
				}
				tableRows = append(tableRows, flowRow)
			}

			res = NodeActionResult{
				Outputs: []FlowValue{{
					Type: &FlowType{
						Kind:          FSKindTable,
						ContainedType: &tableRecordType,
					},
					TableValue: tableRows,
				}},
			}
		default:
			res.Err = fmt.Errorf("unknown format \"%v\"", format)
		}
	}()

	return done
}

// --------------------------------
// Run Process

func NewRunProcessNode(cmd string) *Node {
	return &Node{
		ID:   NewNodeID(),
		Name: "Run Process",

		InputPorts: nil,
		OutputPorts: []NodePort{
			{
				Name: "Stdout",
				Type: FlowType{Kind: FSKindBytes},
			},
			{
				Name: "Stderr",
				Type: FlowType{Kind: FSKindBytes},
			},
			{
				Name: "Combined Stdout/Stderr",
				Type: FlowType{Kind: FSKindBytes},
			},
		},

		Action: &RunProcessAction{
			CmdString: cmd,
		},
	}
}

type RunProcessAction struct {
	CmdString string

	state             RunProcessActionRuntimeState
	outputStreamMutex sync.Mutex
}

var _ NodeAction = &RunProcessAction{}

// The state that gets reset every time you run a command
type RunProcessActionRuntimeState struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc

	stdout   []byte
	stderr   []byte
	combined []byte

	err      error
	exitCode int
}

func (c *RunProcessAction) UpdateAndValidate(n *Node) {
	n.Valid = true
}

func (c *RunProcessAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          GROWH,
			ChildGap:        S2,
		},
	}, func() {
		UITextBox(clay.IDI("RunProcessCmd", n.ID), &c.CmdString, UITextBoxConfig{
			El: clay.EL{Layout: clay.LAY{Sizing: GROWH}},
		})

		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				LayoutDirection: clay.TopToBottom,
				Sizing:          GROWH,
				ChildAlignment:  XRIGHT,
			},
		}, func() {
			UIOutputPort(n, 0)
			UIOutputPort(n, 1)
			UIOutputPort(n, 2)
		})
	})
}

func (c *RunProcessAction) Run(n *Node) <-chan NodeActionResult {
	pieces := strings.Split(c.CmdString, " ")
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, pieces[0], pieces[1:]...)

	done := make(chan NodeActionResult)

	c.state = RunProcessActionRuntimeState{
		cmd:    cmd,
		cancel: cancel,
	}

	cmd.Stdout = &multiSliceWriter{
		mu: &c.outputStreamMutex,
		a:  &c.state.stdout,
		b:  &c.state.combined,
	}
	cmd.Stderr = &multiSliceWriter{
		mu: &c.outputStreamMutex,
		a:  &c.state.stderr,
		b:  &c.state.combined,
	}

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		c.state.err = c.state.cmd.Run()
		if c.state.err != nil {
			// TODO: Extract exit code
		}

		res = NodeActionResult{
			Err: c.state.err,
			Outputs: []FlowValue{
				{
					Type:       &FlowType{Kind: FSKindBytes},
					BytesValue: c.state.stdout,
				},
				{
					Type:       &FlowType{Kind: FSKindBytes},
					BytesValue: c.state.stderr,
				},
				{
					Type:       &FlowType{Kind: FSKindBytes},
					BytesValue: c.state.combined,
				},
			},
		}
	}()

	return done
}

// --------------------------------
// List Files

func NewListFilesNode(dir string) *Node {
	return &Node{
		ID:   NewNodeID(),
		Name: "List Files",

		InputPorts: []NodePort{{
			Name: "Directory Path",
			Type: FlowType{Kind: FSKindBytes},
		}},
		OutputPorts: []NodePort{{
			Name: "Files",
			Type: FlowType{Kind: FSKindTable, ContainedType: FSFile},
		}},

		Action: &ListFilesAction{
			Dir: dir,
		},
	}
}

type ListFilesAction struct {
	Dir string
}

var _ NodeAction = &ListFilesAction{}

func (c *ListFilesAction) UpdateAndValidate(n *Node) {
	n.Valid = true
}

func (c *ListFilesAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			Sizing:         GROWH,
			ChildAlignment: YCENTER,
		},
	}, func() {
		PortAnchor(n, false, 0)
		UITextBox(clay.IDI("ListFilesDir", n.ID), &c.Dir, UITextBoxConfig{
			El:       clay.EL{Layout: clay.LAY{Sizing: GROWH}},
			Disabled: n.InputIsWired(0),
		})
		UISpacer(clay.AUTO_ID, W2)
		UIOutputPort(n, 0)
	})
}

func (c *ListFilesAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		wireDir, hasWire, err := n.GetInputValue(0)
		if err != nil {
			res.Err = err
			return
		}
		entries, err := os.ReadDir(util.Tern(hasWire, string(wireDir.BytesValue), c.Dir))
		if err != nil {
			res.Err = err
			return
		}

		var rows [][]FlowValueField
		for _, entry := range entries {
			info, err := entry.Info()
			if errors.Is(err, os.ErrNotExist) {
				// This can happen if a file was deleted since the dir was listed. Unlikely but hey.
				continue
			} else if err != nil {
				res.Err = err
				return
			}

			row := []FlowValueField{
				{Name: "name", Value: NewStringValue(entry.Name())},
				{Name: "type", Value: NewStringValue(util.Tern(entry.IsDir(), "dir", "file"))},
				{Name: "size", Value: NewInt64Value(info.Size(), FSUnitBytes)},
				{Name: "modified", Value: NewTimestampValue(info.ModTime())},
			}
			rows = append(rows, row)
		}

		res = NodeActionResult{
			Outputs: []FlowValue{{
				Type:       &FlowType{Kind: FSKindTable, ContainedType: FSFile},
				TableValue: rows,
			}},
		}
	}()

	return done
}

// --------------------------------
// Lines

func NewLinesNode() *Node {
	return &Node{
		ID:   NewNodeID(),
		Name: "Lines",

		InputPorts: []NodePort{{
			Name: "Text",
			Type: FlowType{Kind: FSKindBytes},
		}},
		OutputPorts: []NodePort{{
			Name: "Lines",
			Type: FlowType{Kind: FSKindList, ContainedType: &FlowType{Kind: FSKindBytes}},
		}},

		Action: &LinesAction{
			IncludeCarriageReturns: runtime.GOOS == "windows",
		},
	}
}

type LinesAction struct {
	IncludeCarriageReturns bool
}

var _ NodeAction = &LinesAction{}

func (c *LinesAction) UpdateAndValidate(n *Node) {
	n.Valid = true

	if _, ok := n.GetInputWire(0); !ok {
		n.Valid = false
	}
}

func (l *LinesAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{Sizing: GROWH, ChildGap: S2},
	}, func() {
		clay.CLAY_AUTO_ID(clay.EL{ // inputs
			Layout: clay.LAY{
				LayoutDirection: clay.TopToBottom,
				Sizing:          GROWH,
				ChildAlignment:  clay.ChildAlignment{Y: clay.AlignYCenter},
			},
		}, func() {
			UIInputPort(n, 0)
		})
		clay.CLAY_AUTO_ID(clay.EL{ // outputs
			Layout: clay.LAY{
				LayoutDirection: clay.TopToBottom,
				Sizing:          GROWH,
				ChildAlignment:  clay.ChildAlignment{X: clay.AlignXRight, Y: clay.AlignYCenter},
			},
		}, func() {
			UIOutputPort(n, 0)
		})
	})

	// TODO: Checkbox for carriage returns
}

var LFSplit = regexp.MustCompile(`\n`)
var CRLFSplit = regexp.MustCompile(`\r?\n`)

func (l *LinesAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		text, ok, err := n.GetInputValue(0)
		if !ok {
			panic(fmt.Errorf("node %s: no text input, should have been caught by validation", n))
		}
		if err != nil {
			res.Err = err
			return
		}
		linesStrs := util.Tern(l.IncludeCarriageReturns, CRLFSplit, LFSplit).Split(string(text.BytesValue), -1)
		lines := util.Map(linesStrs, func(line string) FlowValue { return NewStringValue(line) })

		res = NodeActionResult{
			Outputs: []FlowValue{{
				Type:      &FlowType{Kind: FSKindList, ContainedType: &FlowType{Kind: FSKindBytes}},
				ListValue: lines,
			}},
		}
	}()

	return done
}

// ---------------------------
// Trim Spaces

func NewTrimSpacesNode() *Node {
	return &Node{
		ID:   NewNodeID(),
		Name: "Trim Spaces",

		InputPorts: []NodePort{{
			Name: "Text",
			Type: FlowType{Kind: FSKindBytes},
		}},
		OutputPorts: []NodePort{{
			Name: "Trimmed",
			Type: FlowType{Kind: FSKindBytes},
		}},

		Action: &TrimSpacesAction{},
	}
}

type TrimSpacesAction struct{}

var _ NodeAction = &TrimSpacesAction{}

func (c *TrimSpacesAction) UpdateAndValidate(n *Node) {
	n.Valid = true

	wire, hasWire := n.GetInputWire(0)
	if hasWire && Typecheck(wire.Type(), NewListType(FlowType{Kind: FSKindBytes})) == nil {
		n.InputPorts[0] = NodePort{
			Name: "Text items",
			Type: NewListType(FlowType{Kind: FSKindBytes}),
		}
		n.OutputPorts[0] = NodePort{
			Name: "Trimmed",
			Type: NewListType(FlowType{Kind: FSKindBytes}),
		}
	} else {
		n.InputPorts[0] = NodePort{
			Name: "Text",
			Type: FlowType{Kind: FSKindBytes},
		}
		n.OutputPorts[0] = NodePort{
			Name: "Trimmed",
			Type: FlowType{Kind: FSKindBytes},
		}
	}

	if !hasWire {
		n.Valid = false
	}
}

func (l *TrimSpacesAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			Sizing:         GROWH,
			ChildAlignment: YCENTER,
		},
	}, func() {
		UIInputPort(n, 0)
		UISpacer(clay.AUTO_ID, GROWH)
		UIOutputPort(n, 0)
	})
}

func (l *TrimSpacesAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		input, ok, err := n.GetInputValue(0)
		if !ok {
			res.Err = errors.New("an input node is required")
			return
		}
		if err != nil {
			res.Err = err
			return
		}

		if input.Type.Kind == FSKindBytes {
			res = NodeActionResult{
				Outputs: []FlowValue{NewBytesValue(bytes.TrimSpace(input.BytesValue))},
			}
		} else {
			res = NodeActionResult{
				Outputs: []FlowValue{NewListValue(
					FlowType{Kind: FSKindBytes},
					util.Map(input.ListValue, func(fv FlowValue) FlowValue {
						return NewBytesValue(bytes.TrimSpace(fv.BytesValue))
					}),
				)},
			}
		}
	}()

	return done
}

// ---------------------------
// Aggregate

func NewAggregateNode(op string) *Node {
	action := AggregateAction{
		ops: UIDropdown{
			Options: []UIDropdownOption{
				{Name: "Min", Value: AggOpMin},
				{Name: "Max", Value: AggOpMax},
				{Name: "Mean", Value: AggOpMean},
			},
		},
	}
	action.ops.SelectByName(op)

	return &Node{
		ID:   NewNodeID(),
		Name: "Aggregate",

		InputPorts: []NodePort{{
			Name: "Input",
			Type: FlowType{Kind: FSKindAny},
		}},
		OutputPorts: []NodePort{{
			Name: "Result",
			Type: FlowType{Kind: FSKindAny},
		}},

		Action: &action,
	}
}

type AggregateAction struct {
	ops UIDropdown
}

var _ NodeAction = &AggregateAction{}

func (a *AggregateAction) UpdateAndValidate(n *Node) {
	n.Valid = true

	wire, hasWire := n.GetInputWire(0)
	if hasWire {
		if Typecheck(wire.Type(), NewListType(FlowType{Kind: FSKindInt64})) == nil {
			// List[Int64] -> Int64
			n.OutputPorts[0].Type = FlowType{Kind: FSKindInt64}
		} else if Typecheck(wire.Type(), NewListType(FlowType{Kind: FSKindFloat64})) == nil {
			// List[Float64] -> Float64
			n.OutputPorts[0].Type = FlowType{Kind: FSKindFloat64}
		} else if Typecheck(wire.Type(), NewAnyTableType()) == nil {
			// Table[Any] -> Table[Any] (only numeric columns aggregated, other columns cleared)
			n.OutputPorts[0].Type = wire.Type()
		} else {
			// Dunno, catch it at runtime
			n.OutputPorts[0].Type = FlowType{Kind: FSKindAny}
		}
	} else {
		n.OutputPorts[0].Type = FlowType{Kind: FSKindAny}
	}

	if !hasWire {
		n.Valid = false
	}
}

func (a *AggregateAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          GROWH,
			ChildGap:        S2,
		},
	}, func() {
		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				Sizing:         GROWH,
				ChildAlignment: YCENTER,
			},
		}, func() {
			UIInputPort(n, 0)
			UISpacer(clay.AUTO_ID, GROWH)
			UIOutputPort(n, 0)
		})

		a.ops.Do(clay.AUTO_ID, UIDropdownConfig{
			El: clay.EL{
				Layout: clay.LAY{Sizing: GROWH},
			},
		})
	})
}

func (a *AggregateAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		input, ok, err := n.GetInputValue(0)
		if !ok {
			res.Err = errors.New("an input node is required")
			return
		}
		if err != nil {
			res.Err = err
			return
		}

		op := a.ops.GetSelectedOption().Value.(AggOp)
		switch input.Type.Kind {
		case FSKindList:
			agged, err := op(input.ListValue, *input.Type.ContainedType)
			if err != nil {
				res.Err = err
				return
			}
			res = NodeActionResult{
				Outputs: []FlowValue{agged},
			}
		case FSKindTable:
			aggedRow := make([]FlowValueField, len(input.Type.ContainedType.Fields))
			for col, field := range input.Type.ContainedType.Fields {
				agged, err := op(input.ColumnValues(col), *field.Type)
				if err != nil {
					res.Err = fmt.Errorf("for column %s: %v", field.Name, err)
					return
				}
				aggedRow[col] = FlowValueField{
					Name:  field.Name,
					Value: agged,
				}
			}
			res = NodeActionResult{
				Outputs: []FlowValue{{
					Type:       input.Type,
					TableValue: [][]FlowValueField{aggedRow},
				}},
			}
		default:
			res.Err = fmt.Errorf("can only aggregate lists or tables, not %s", input.Type)
		}
	}()

	return done
}

type AggOp = func(vals []FlowValue, t FlowType) (FlowValue, error)

var _ AggOp = AggOpMin
var _ AggOp = AggOpMax
var _ AggOp = AggOpMean

func AggOpMin(vals []FlowValue, t FlowType) (FlowValue, error) {
	if len(vals) == 0 {
		// Zero value of the desired type, if no values at all
		return FlowValue{Type: &t}, nil
	}

	switch t.Kind {
	case FSKindInt64:
		res := vals[0].Int64Value
		for _, v := range vals {
			res = util.Min(res, v.Int64Value)
		}
		return FlowValue{Type: &t, Int64Value: res}, nil
	case FSKindFloat64:
		res := vals[0].Float64Value
		for _, v := range vals {
			res = util.Min(res, v.Float64Value)
		}
		return FlowValue{Type: &t, Float64Value: res}, nil
	default:
		return FlowValue{}, fmt.Errorf("cannot min values of type %s", t)
	}
}

func AggOpMax(vals []FlowValue, t FlowType) (FlowValue, error) {
	if len(vals) == 0 {
		// Zero value of the desired type, if no values at all
		return FlowValue{Type: &t}, nil
	}

	switch t.Kind {
	case FSKindInt64:
		res := vals[0].Int64Value
		for _, v := range vals {
			res = util.Max(res, v.Int64Value)
		}
		return FlowValue{Type: &t, Int64Value: res}, nil
	case FSKindFloat64:
		res := vals[0].Float64Value
		for _, v := range vals {
			res = util.Max(res, v.Float64Value)
		}
		return FlowValue{Type: &t, Float64Value: res}, nil
	default:
		return FlowValue{}, fmt.Errorf("cannot max values of type %s", t)
	}
}

func AggOpMean(vals []FlowValue, t FlowType) (FlowValue, error) {
	if len(vals) == 0 {
		// Zero value of the desired type, if no values at all
		return FlowValue{Type: &t}, nil
	}

	switch t.Kind {
	case FSKindInt64:
		var sum int64
		for _, v := range vals {
			sum += v.Int64Value
		}
		return FlowValue{Type: &t, Int64Value: sum / int64(len(vals))}, nil
	case FSKindFloat64:
		var sum float64
		for _, v := range vals {
			sum += v.Float64Value
		}
		return FlowValue{Type: &t, Float64Value: sum / float64(len(vals))}, nil
	default:
		return FlowValue{}, fmt.Errorf("cannot average values of type %s", t)
	}
}
