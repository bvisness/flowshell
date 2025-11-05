package app

import (
	"bytes"
	"errors"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

type TrimSpacesAction struct{}

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

var _ Serializable[TrimSpacesAction] = TrimSpacesAction{}

func (TrimSpacesAction) Serialize(s *Serializer, n *TrimSpacesAction) error {
	return s.Err
}
