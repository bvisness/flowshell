package app

import (
	"errors"
	"fmt"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

type AggregateAction struct {
	ops UIDropdown
}

var aggOptions = []UIDropdownOption{
	{Name: "Min", Value: AggOpMin},
	{Name: "Max", Value: AggOpMax},
	{Name: "Mean", Value: AggOpMean},
}

func NewAggregateNode(op string) *Node {
	action := AggregateAction{
		ops: UIDropdown{
			Options: aggOptions,
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

var _ Serializable[AggregateAction] = AggregateAction{}

func (AggregateAction) Serialize(s *Serializer, n *AggregateAction) error {
	if s.Encode {
		s.WriteStr(n.ops.GetSelectedOption().Name)
	} else {
		selected, _ := s.ReadStr()
		n.ops = UIDropdown{Options: aggOptions}
		n.ops.SelectByName(selected)
		util.Assert(n.ops.GetSelectedOption().Name == selected, "aggregate %s should have been selected, but %s was instead", selected, n.ops.GetSelectedOption().Name)
	}
	return s.Err
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
