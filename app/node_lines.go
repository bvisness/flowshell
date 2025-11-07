package app

import (
	"fmt"
	"regexp"
	"runtime"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

// GEN:NodeAction
type LinesAction struct {
	IncludeCarriageReturns bool
}

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

func (n *LinesAction) Serialize(s *Serializer) bool {
	SBool(s, &n.IncludeCarriageReturns)
	return s.Ok()
}
