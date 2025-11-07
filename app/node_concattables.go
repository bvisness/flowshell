package app

import (
	"errors"
	"fmt"
	"slices"

	"github.com/bvisness/flowshell/clay"
)

// GEN:NodeAction
type ConcatTablesAction struct{}

func NewConcatTablesNode() *Node {
	return &Node{
		ID:   NewNodeID(),
		Name: "Concatenate Tables",

		InputPorts: []NodePort{{
			Name: "Table 1",
			Type: NewAnyTableType(),
		}},
		OutputPorts: []NodePort{{
			Name: "Table",
			Type: NewAnyTableType(),
		}},

		Action: &ConcatTablesAction{},
	}
}

var _ NodeAction = &ConcatTablesAction{}

func (a *ConcatTablesAction) UpdateAndValidate(n *Node) {
	n.Valid = true

	allPortsWired := true
	for i := range n.InputPorts {
		wire, hasWire := n.GetInputWire(i)
		if !hasWire {
			allPortsWired = false
			n.OutputPorts[0].Type = NewAnyTableType()
		} else if i == 0 {
			n.OutputPorts[0].Type = wire.Type()
		}
	}

	if !allPortsWired {
		n.Valid = false
	}
}

func (a *ConcatTablesAction) UI(n *Node) {
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
			buttonStyle := clay.EL{
				Layout: clay.LAY{
					Sizing:         WH(24, 24),
					ChildAlignment: ALLCENTER,
				},
				Border: clay.B{Width: BA, Color: Gray},
			}
			buttonTextConfig := clay.T{FontID: InterSemibold, FontSize: F2, TextColor: White}

			UIButton(clay.AUTO_ID, UIButtonConfig{ // -
				El: buttonStyle,
				OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
					if len(n.InputPorts) > 1 {
						wires = slices.DeleteFunc(wires, func(w *Wire) bool {
							return w.EndNode == n && w.EndPort >= len(n.InputPorts)-1
						})
						n.InputPorts = n.InputPorts[:len(n.InputPorts)-1]
					}
				},
			}, func() {
				clay.TEXT("-", buttonTextConfig)
			})
			UISpacer(clay.AUTO_ID, W1)
			UIButton(clay.AUTO_ID, UIButtonConfig{ // +
				El: buttonStyle,
				OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
					n.InputPorts = append(n.InputPorts, NodePort{
						Name: fmt.Sprintf("Table %d", len(n.InputPorts)+1),
						Type: NewAnyTableType(),
					})
				},
			}, func() {
				clay.TEXT("+", buttonTextConfig)
			})
			UISpacer(clay.AUTO_ID, GROWH)
			UIOutputPort(n, 0)
		})

		for i := range n.InputPorts {
			UIInputPort(n, i)
		}
	})
}

func (a *ConcatTablesAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		firstInput, ok, err := n.GetInputValue(0)
		if !ok {
			panic("first input was not wired (validation should have caught this)")
		}
		if err != nil {
			res.Err = err
			return
		}
		expectedType := firstInput.Type

		var tableRowses [][][]FlowValueField
		for i := range n.InputPorts {
			input, ok, err := n.GetInputValue(i)
			if !ok {
				res.Err = errors.New("an input node is required")
				return
			}
			if err != nil {
				res.Err = err
				return
			}

			if err := Typecheck(*input.Type, *expectedType); err != nil {
				res.Err = fmt.Errorf("all tables should have the same type: expected %s but got %s", expectedType, input.Type)
				return
			}
			tableRowses = append(tableRowses, input.TableValue)
		}

		var finalTableRows [][]FlowValueField
		for _, rows := range tableRowses {
			finalTableRows = append(finalTableRows, rows...)
		}
		res = NodeActionResult{
			Outputs: []FlowValue{{
				Type:       expectedType,
				TableValue: finalTableRows,
			}},
		}
	}()

	return done
}

func (n *ConcatTablesAction) Serialize(s *Serializer) bool {
	return s.Ok()
}
