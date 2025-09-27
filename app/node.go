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
	"strings"
	"sync"
	"time"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type V2 = rl.Vector2

type Node struct {
	ID   int
	Pos  V2
	Name string

	InputPorts          []NodePort
	OutputPorts         []NodePort
	InputPortPositions  []V2
	OutputPortPositions []V2

	Action NodeAction
	Valid  bool

	Running bool
	done    chan struct{}

	ResultAvailable bool
	Result          NodeActionResult
}

func (n *Node) String() string {
	return fmt.Sprintf("Node#%d(%s)", n.ID, n.Name)
}

func (n *Node) ClayID() clay.ElementID {
	return clay.IDI("Node", n.ID)
}

type NodePort struct {
	Name string
	Type FlowType
}

type Wire struct {
	StartNode, EndNode *Node
	StartPort, EndPort int
}

func (n *Node) Run() <-chan struct{} {
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
			if !inputNode.ResultAvailable {
				inputRuns = append(inputRuns, inputNode.Run())
			}
		}
		for _, inputRun := range inputRuns {
			<-inputRun
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

func (n *Node) UpdatePortPositions() {
	n.InputPortPositions = make([]V2, len(n.InputPorts))
	n.OutputPortPositions = make([]V2, len(n.OutputPorts))

	bboxNode := util.Must1B(clay.GetElementData(n.ClayID())).BoundingBox
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
			path:   path,
			format: formatDropdown,
		},
	}
}

type LoadFileAction struct {
	path string

	format UIDropdown
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
			UITextBox(clay.AUTO_ID, &c.path, clay.EL{
				Layout: clay.LAY{Sizing: GROWH},
			})
			UISpacer(W2)
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
					Type: &FlowType{Kind: FSKindBytes},
				})
			}

			// TODO(low): Be resilient against variable numbers of fields per row, potentially
			var tableRows [][]FlowValueField
			for _, row := range rows[1:] {
				var flowRow []FlowValueField
				for col, value := range row {
					flowRow = append(flowRow, FlowValueField{
						Name:  rows[0][col],
						Value: NewStringValue(value),
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
		UITextBox(clay.AUTO_ID, &c.CmdString, clay.EL{Layout: clay.LAY{Sizing: GROWH}})

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
	UITextBox(clay.ID("Cmd"), &c.Dir, clay.EL{Layout: clay.LAY{Sizing: GROWH}})
}

func (c *ListFilesAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		entries, err := os.ReadDir(c.Dir)
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

func NewBytesValue(bytes []byte) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindBytes}, BytesValue: bytes}
}

func NewStringValue(str string) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindBytes}, BytesValue: []byte(str)}
}

func NewInt64Value(v int64, unit FlowUnit) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindInt64, Unit: unit}, Int64Value: v}
}

func NewTimestampValue(t time.Time) FlowValue {
	return FlowValue{Type: FSTimestamp, Int64Value: t.Unix()}
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
