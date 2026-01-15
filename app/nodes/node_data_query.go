package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
	"github.com/tidwall/gjson"
)

// GEN:NodeAction
type JsonQueryAction struct {
	Query string
}

func NewJsonQueryNode() *core.Node {
	return &core.Node{
		Name: "JSON Query",
		Action: &JsonQueryAction{
			Query: "foo.bar",
		},
		InputPorts: []core.NodePort{
			{Name: "JSON", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Result", Type: core.FlowType{Kind: core.FSKindAny}},
		},
	}
}

func (a *JsonQueryAction) UpdateAndValidate(n *core.Node) { n.Valid = true }

func (a *JsonQueryAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("JsonQUI", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("Spacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})

		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			clay.TEXT("GJSON Query:", clay.TextElementConfig{FontSize: 12, TextColor: core.LightGray})
		})

		clay.CLAY(clay.IDI("Row3", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			core.UITextBox(clay.IDI("QueryStr", n.ID), &a.Query, core.UITextBoxConfig{
				El: clay.EL{Layout: clay.LAY{Sizing: core.GROWH}},
			})
		})
	})
}

func (a *JsonQueryAction) Run(n *core.Node) <-chan core.NodeActionResult {
	ch := make(chan core.NodeActionResult, 1)
	go func() {
		defer close(ch)
		inputVal, ok, err := n.GetInputValue(0)
		if err != nil {
			ch <- core.NodeActionResult{Err: err}
			return
		}
		if !ok {
			ch <- core.NodeActionResult{Err: fmt.Errorf("missing input")}
			return
		}

		jsonStr := string(inputVal.BytesValue)
		// Parse using gjson
		res := gjson.Get(jsonStr, a.Query)

		val := res.Value()
		flowVal, err := core.NativeToFlowValue(val)
		if err != nil {
			ch <- core.NodeActionResult{Err: fmt.Errorf("conversion failed: %w", err)}
			return
		}

		// Auto-promote List[Record] to Table
		if flowVal.Type.Kind == core.FSKindList && flowVal.Type.ContainedType != nil && flowVal.Type.ContainedType.Kind == core.FSKindRecord {
			// Convert ListValue to TableValue
			var tableRows [][]core.FlowValueField
			for _, item := range flowVal.ListValue {
				tableRows = append(tableRows, item.RecordValue)
			}

			// Construct Table type
			tableType := core.NewTableType(flowVal.Type.ContainedType.Fields)

			flowVal = core.FlowValue{
				Type:       &tableType,
				TableValue: tableRows,
			}
		}

		ch <- core.NodeActionResult{Outputs: []core.FlowValue{flowVal}}
	}()
	return ch
}

func (a *JsonQueryAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *JsonQueryAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &a.Query)
	return s.Ok()
}

// GEN:NodeAction
type XmlQueryAction struct {
	XPath string
}

func NewXmlQueryNode() *core.Node {
	return &core.Node{
		Name: "XML Query",
		Action: &XmlQueryAction{
			XPath: "//item",
		},
		InputPorts: []core.NodePort{
			{Name: "XML", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
		OutputPorts: []core.NodePort{
			{Name: "Result", Type: core.FlowType{Kind: core.FSKindBytes}},
		},
	}
}

func (a *XmlQueryAction) UpdateAndValidate(n *core.Node) { n.Valid = true }

func (a *XmlQueryAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("XmlQUI", n.ID), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: core.GROWH, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Row1", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("Spacer1", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})

		clay.CLAY(clay.IDI("Row2", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			clay.TEXT("XPath:", clay.TextElementConfig{FontSize: 12, TextColor: core.LightGray})
		})

		clay.CLAY(clay.IDI("Row3", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			core.UITextBox(clay.IDI("XPathStr", n.ID), &a.XPath, core.UITextBoxConfig{
				El: clay.EL{Layout: clay.LAY{Sizing: core.GROWH}},
			})
		})
	})
}

func (a *XmlQueryAction) Run(n *core.Node) <-chan core.NodeActionResult {
	ch := make(chan core.NodeActionResult, 1)
	go func() {
		defer close(ch)
		inputVal, ok, err := n.GetInputValue(0)
		if err != nil {
			ch <- core.NodeActionResult{Err: err}
			return
		}
		if !ok {
			ch <- core.NodeActionResult{Err: fmt.Errorf("missing input")}
			return
		}

		xmlStr := string(inputVal.BytesValue)
		doc, err := xmlquery.Parse(strings.NewReader(xmlStr))
		if err != nil {
			ch <- core.NodeActionResult{Err: fmt.Errorf("failed to parse XML: %v", err)}
			return
		}

		list, err := xmlquery.QueryAll(doc, a.XPath)
		if err != nil {
			ch <- core.NodeActionResult{Err: fmt.Errorf("xpath query failed: %v", err)}
			return
		}

		var builder strings.Builder
		for i, n := range list {
			if i > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(n.OutputXML(true))
		}

		ch <- core.NodeActionResult{Outputs: []core.FlowValue{core.NewStringValue(builder.String())}}
	}()
	return ch
}

func (a *XmlQueryAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	return a.Run(n)
}

func (a *XmlQueryAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &a.XPath)
	return s.Ok()
}
