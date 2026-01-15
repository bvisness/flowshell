package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// --- Join Text ---

// GEN:NodeAction
type JoinTextAction struct{}

func NewJoinTextNode() *core.Node {
	return &core.Node{
		Name: "Join Text",
		InputPorts: []core.NodePort{
			{Name: "List", Type: core.FlowType{Kind: core.FSKindList, ContainedType: &core.FlowType{Kind: core.FSKindBytes}}},
			{Name: "Separator", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		Action: &JoinTextAction{},
	}
}

var _ core.NodeAction = &JoinTextAction{}

func (a *JoinTextAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
}

func (a *JoinTextAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("JoinSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})
		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 1)
		})
	})
}

func (a *JoinTextAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		valList, _, err1 := n.GetInputValue(0)
		valSep, _, err2 := n.GetInputValue(1)

		if err1 != nil || err2 != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad inputs: %v, %v", err1, err2)}
			return
		}

		sep := ""
		if valSep.Type.Kind == core.FSKindBytes {
			sep = string(valSep.BytesValue)
		}

		var parts []string
		if valList.Type.Kind == core.FSKindList {
			for _, item := range valList.ListValue {
				if item.Type.Kind == core.FSKindBytes {
					parts = append(parts, string(item.BytesValue))
				} else {
					// Best effort conversion
					parts = append(parts, fmt.Sprintf("%v", core.FlowValueToNative(item)))
				}
			}
		}

		result := strings.Join(parts, sep)

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{core.NewStringValue(result)},
		}
	}()
	return done
}

func (a *JoinTextAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *JoinTextAction) Serialize(s *core.Serializer) bool { return s.Ok() }

// --- Split Text ---

// GEN:NodeAction
type SplitTextAction struct{}

func NewSplitTextNode() *core.Node {
	return &core.Node{
		Name: "Split Text",
		InputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
			{Name: "Separator", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "List", Type: core.NewListType(core.FlowType{Kind: core.FSKindBytes})},
		},
		Action: &SplitTextAction{},
	}
}

var _ core.NodeAction = &SplitTextAction{}

func (a *SplitTextAction) UpdateAndValidate(n *core.Node) { n.Valid = true }
func (a *SplitTextAction) UI(n *core.Node) {
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

func (a *SplitTextAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		valText, _, err1 := n.GetInputValue(0)
		valSep, _, err2 := n.GetInputValue(1)

		if err1 != nil || err2 != nil {
			done <- core.NodeActionResult{Err: fmt.Errorf("bad inputs: %v, %v", err1, err2)}
			return
		}

		text := string(valText.BytesValue)
		sep := string(valSep.BytesValue)

		parts := strings.Split(text, sep)
		var list []core.FlowValue
		for _, p := range parts {
			list = append(list, core.NewStringValue(p))
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{core.NewListValue(core.FlowType{Kind: core.FSKindBytes}, list)},
		}
	}()
	return done
}

func (a *SplitTextAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *SplitTextAction) Serialize(s *core.Serializer) bool { return s.Ok() }

// --- Case Convert ---

type CaseMode int

const (
	CaseUpper CaseMode = iota
	CaseLower
	CaseTitle
)

// GEN:NodeAction
type CaseConvertAction struct {
	Mode CaseMode
}

func NewCaseConvertNode() *core.Node {
	return &core.Node{
		Name: "Change Case",
		InputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		Action: &CaseConvertAction{Mode: CaseUpper},
	}
}

var _ core.NodeAction = &CaseConvertAction{}

func (a *CaseConvertAction) UpdateAndValidate(n *core.Node) { n.Valid = true }

func (a *CaseConvertAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("CaseRow1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("CaseSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})

		// Mode Selection
		clay.CLAY(clay.IDI("CaseRow2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildGap: core.S2},
		}, func() {
			modes := []struct {
				L string
				M CaseMode
			}{
				{"UPPER", CaseUpper}, {"lower", CaseLower}, {"Title", CaseTitle},
			}
			for _, m := range modes {
				active := a.Mode == m.M
				core.UIButton(clay.IDI("CaseBtn"+m.L, n.ID), core.UIButtonConfig{
					OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
						a.Mode = m.M
					},
					El: clay.EL{BackgroundColor: util.Tern(active, core.Blue, core.Charcoal)},
				}, func() {
					clay.TEXT(m.L, clay.TextElementConfig{TextColor: core.White, FontSize: 14})
				})
			}
		})
	})
}

func (a *CaseConvertAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		valText, _, err := n.GetInputValue(0)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		text := string(valText.BytesValue)
		var res string
		switch a.Mode {
		case CaseUpper:
			res = strings.ToUpper(text)
		case CaseLower:
			res = strings.ToLower(text)
		case CaseTitle:
			res = cases.Title(language.English).String(text)
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{core.NewStringValue(res)},
		}
	}()
	return done
}

func (a *CaseConvertAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *CaseConvertAction) Serialize(s *core.Serializer) bool {
	core.SInt(s, &a.Mode)
	return s.Ok()
}

// --- Format String ---

// GEN:NodeAction
type FormatStringAction struct {
	Format string
}

func NewFormatStringNode() *core.Node {
	return &core.Node{
		Name: "Format String",
		InputPorts: []core.NodePort{
			{Name: "Value", Type: core.FlowType{Kind: core.FSKindAny}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Text", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		Action: &FormatStringAction{Format: "Val: %v"},
	}
}

var _ core.NodeAction = &FormatStringAction{}

func (a *FormatStringAction) UpdateAndValidate(n *core.Node) { n.Valid = true }

func (a *FormatStringAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("NodeContent", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("FmtRow1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("FmtSpacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})

		clay.CLAY(clay.IDI("FmtRow2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			core.UITextBox(clay.IDI("FormatStr", n.ID), &a.Format, core.UITextBoxConfig{
				El: clay.EL{Layout: clay.LAY{Sizing: core.GROWH}},
			})
		})
	})
}

func (a *FormatStringAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult, 1)
	go func() {
		val, _, err := n.GetInputValue(0)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		native := core.FlowValueToNative(val)
		res := fmt.Sprintf(a.Format, native)

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{core.NewStringValue(res)},
		}
	}()
	return done
}

func (a *FormatStringAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *FormatStringAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &a.Format)
	return s.Ok()
}
