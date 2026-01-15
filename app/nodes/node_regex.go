package nodes

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
)

// GEN:NodeAction
type RegexMatchAction struct{}

func NewRegexMatchNode() *core.Node {
	return &core.Node{
		Name: "Regex Match",
		InputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Pattern", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Match", Type: core.FlowType{Kind: core.FSKindInt64}}, // 0 or 1 for bool
		},
		Action: &RegexMatchAction{},
	}
}

var _ core.NodeAction = &RegexMatchAction{}

func (a *RegexMatchAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true // Validation could check if pattern compiles
}

func (a *RegexMatchAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("MatchSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 1)
		})
	})
}

func (a *RegexMatchAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		val0, _, err0 := n.GetInputValue(0)
		val1, _, err1 := n.GetInputValue(1)

		if err0 != nil || err1 != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad inputs: %v, %v", err0, err1)}
			return
		}

		text := string(val0.BytesValue)
		pattern := string(val1.BytesValue)

		re, err := regexp.Compile(pattern)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		match := re.MatchString(text)
		val := int64(0)
		if match {
			val = 1
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewInt64Value(val, 0),
			},
		}
	}()
	return done
}

func (a *RegexMatchAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *RegexMatchAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}

// GEN:NodeAction
type RegexFindAllAction struct{}

func NewRegexFindAllNode() *core.Node {
	return &core.Node{
		Name: "Regex Find All",
		InputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Pattern", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Matches", Type: core.NewListType(core.FlowType{Kind: core.FSKindBytes})},
		},
		Action: &RegexFindAllAction{},
	}
}

var _ core.NodeAction = &RegexFindAllAction{}

func (a *RegexFindAllAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *RegexFindAllAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("FindAllSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 1)
		})
	})
}

func (a *RegexFindAllAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		val0, _, err0 := n.GetInputValue(0)
		val1, _, err1 := n.GetInputValue(1)

		if err0 != nil || err1 != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad inputs: %v, %v", err0, err1)}
			return
		}

		text := string(val0.BytesValue)
		pattern := string(val1.BytesValue)

		re, err := regexp.Compile(pattern)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		matches := re.FindAllString(text, -1)
		var flowMatches []core.FlowValue
		for _, m := range matches {
			flowMatches = append(flowMatches, core.NewStringValue(m))
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewListValue(core.FlowType{Kind: core.FSKindBytes}, flowMatches),
			},
		}
	}()
	return done
}

func (a *RegexFindAllAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *RegexFindAllAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}

// GEN:NodeAction
type RegexReplaceAction struct{}

func NewRegexReplaceNode() *core.Node {
	return &core.Node{
		Name: "Regex Replace",
		InputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Pattern", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Replacement", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Result", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		Action: &RegexReplaceAction{},
	}
}

var _ core.NodeAction = &RegexReplaceAction{}

func (a *RegexReplaceAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *RegexReplaceAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("ReplaceSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 1)
		})
		clay.CLAY(clay.IDI("Row3", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 2)
		})
	})
}

func (a *RegexReplaceAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		val0, _, err0 := n.GetInputValue(0)
		val1, _, err1 := n.GetInputValue(1)
		val2, _, err2 := n.GetInputValue(2)

		if err0 != nil || err1 != nil || err2 != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad inputs: %v, %v, %v", err0, err1, err2)}
			return
		}

		text := string(val0.BytesValue)
		pattern := string(val1.BytesValue)
		repl := string(val2.BytesValue)

		re, err := regexp.Compile(pattern)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		result := re.ReplaceAllString(text, repl)

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewStringValue(result),
			},
		}
	}()
	return done
}

func (a *RegexReplaceAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *RegexReplaceAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}

// GEN:NodeAction
type RegexSplitAction struct{}

func NewRegexSplitNode() *core.Node {
	return &core.Node{
		Name: "Regex Split",
		InputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Pattern", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Parts", Type: core.NewListType(core.FlowType{Kind: core.FSKindBytes})},
		},
		Action: &RegexSplitAction{},
	}
}

var _ core.NodeAction = &RegexSplitAction{}

func (a *RegexSplitAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *RegexSplitAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("SplitSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 1)
		})
	})
}

func (a *RegexSplitAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		val0, _, err0 := n.GetInputValue(0)
		val1, _, err1 := n.GetInputValue(1)

		if err0 != nil || err1 != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad inputs: %v, %v", err0, err1)}
			return
		}

		text := string(val0.BytesValue)
		pattern := string(val1.BytesValue)

		re, err := regexp.Compile(pattern)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		parts := re.Split(text, -1)
		var flowParts []core.FlowValue
		for _, p := range parts {
			flowParts = append(flowParts, core.NewStringValue(p))
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{
				core.NewListValue(core.FlowType{Kind: core.FSKindBytes}, flowParts),
			},
		}
	}()
	return done
}

func (a *RegexSplitAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *RegexSplitAction) Serialize(s *core.Serializer) bool {
	return s.Ok()
}
