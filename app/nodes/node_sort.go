package nodes

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

// GEN:NodeAction
type SortAction struct {
	Reverse bool
}

func NewSortNode() *core.Node {
	return &core.Node{
		Name: "Sort",

		InputPorts: []core.NodePort{{
			Name: "List",
			Type: core.NewListType(core.FlowType{Kind: core.FSKindBytes}),
		}},
		OutputPorts: []core.NodePort{{
			Name: "Sorted",
			Type: core.NewListType(core.FlowType{Kind: core.FSKindBytes}),
		}},

		Action: &SortAction{},
	}
}

var _ core.NodeAction = &SortAction{}

func (c *SortAction) Serialize(s *core.Serializer) bool {
	core.SBool(s, &c.Reverse)
	return s.Ok()
}

func (c *SortAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true

	if wire, ok := n.GetInputWire(0); ok {
		n.InputPorts[0].Type = wire.Type()
		n.OutputPorts[0].Type = wire.Type()

		if wire.Type().Kind != core.FSKindList {
			n.Valid = false
		}
	} else {
		n.Valid = false
	}
}

func (c *SortAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("SortContent", n.ID), clay.EL{
		Layout: clay.LAY{
			Sizing:         core.GROWH,
			ChildAlignment: core.YCENTER,
		},
	}, func() {
		core.UIInputPort(n, 0)
		core.UISpacer(clay.IDI("SortSpacerInput", n.ID), core.GROWH)

		// Reverse Checkbox
		clay.CLAY(clay.IDI("SortReverseContainer", n.ID), clay.EL{
			Layout: clay.LAY{ChildGap: core.S1, ChildAlignment: core.YCENTER},
		}, func() {
			core.UIButton(clay.IDI("SortReverseBtn", n.ID), core.UIButtonConfig{
				OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
					c.Reverse = !c.Reverse
				},
			}, func() {
				core.UIImage(clay.IDI("SortReverseIcon", n.ID), util.Tern(c.Reverse, core.ImgToggleDown, core.ImgToggleRight), clay.EL{}) // Using toggle icons for checkbox for now
			})
			clay.TEXT("Reverse", clay.TextElementConfig{TextColor: core.White})
		})

		core.UISpacer(clay.IDI("SortSpacerOutput", n.ID), core.GROWH)
		core.UIOutputPort(n, 0)
	})
}

func (c *SortAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult)

	go func() {
		var res core.NodeActionResult
		defer func() {
			if r := recover(); r != nil {
				res = core.NodeActionResult{Err: fmt.Errorf("panic in node %s: %v", n.Name, r)}
			}
			done <- res
			close(done)
		}()

		select {
		case <-ctx.Done():
			res.Err = ctx.Err()
			return
		default:
		}

		input, ok, err := n.GetInputValue(0)
		if !ok {
			res.Err = errors.New("input required")
			return
		}
		if err != nil {
			res.Err = err
			return
		}

		if input.Type.Kind != core.FSKindList {
			res.Err = errors.New("input must be a list")
			return
		}

		// Copy list to avoid modifying input
		sorted := make([]core.FlowValue, len(input.ListValue))
		copy(sorted, input.ListValue)

		// Recover from panic caused by cancellation
		defer func() {
			if r := recover(); r != nil {
				if r == ctx.Err() {
					res.Err = ctx.Err()
					// done <- res will happen in the outer defer
				} else {
					panic(r) // Re-panic if it's not our cancellation
				}
			}
		}()

		slices.SortFunc(sorted, func(a, b core.FlowValue) int {
			// Check for cancellation
			if ctx.Err() != nil {
				panic(ctx.Err())
			}

			res := 0
			switch a.Type.Kind {
			case core.FSKindBytes:
				res = cmp.Compare(string(a.BytesValue), string(b.BytesValue))
			case core.FSKindInt64:
				res = cmp.Compare(a.Int64Value, b.Int64Value)
			case core.FSKindFloat64:
				res = cmp.Compare(a.Float64Value, b.Float64Value)
			}

			if c.Reverse {
				return -res
			}
			return res
		})

		res.Outputs = []core.FlowValue{core.NewListValue(*input.Type.ContainedType, sorted)}
	}()

	return done
}

func (c *SortAction) Run(n *core.Node) <-chan core.NodeActionResult {
	return c.RunContext(context.Background(), n)
}
