package nodes

import (
	"context"
	"fmt"
	"slices"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

// GEN:NodeAction
type SelectColumnsAction struct {
	SelectedColumns []string
}

func NewSelectColumnsNode() *core.Node {
	return &core.Node{
		Name: "Select Columns",

		InputPorts: []core.NodePort{{
			Name: "Input",
			Type: core.FlowType{Kind: core.FSKindTable},
		}},
		OutputPorts: []core.NodePort{{
			Name: "Output",
			Type: core.FlowType{Kind: core.FSKindTable},
		}},

		Action: &SelectColumnsAction{},
	}
}

var _ core.NodeAction = &SelectColumnsAction{}

func (c *SelectColumnsAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true

	input, wired := n.GetInputWire(0)
	if !wired {
		n.Valid = false
		n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindTable} // Generic table
		return
	}

	if input.Type().Kind != core.FSKindTable {
		n.Valid = false // Should be enforced by port type, but good to check
		return
	}

	// Calculate output type
	var newFields []core.FlowField
	if input.Type().ContainedType != nil {
		for _, field := range input.Type().ContainedType.Fields {
			if slices.Contains(c.SelectedColumns, field.Name) {
				newFields = append(newFields, field)
			}
		}
	}

	// If nothing selected, maybe default to all? Or none?
	// Let's say if list is empty, select all by default?
	// Or maybe user explicitly wants empty table.
	// For now, if SelectedColumns is nil/empty, we might want to initialize it with all columns if this is the first run?
	// But UpdateAndValidate is called every frame. We shouldn't mutate state here if possible, or be careful.
	// Actually, initializing on first connect is a nice UX.

	// Let's rely on UI to let user select.

	n.OutputPorts[0].Type = core.FlowType{
		Kind: core.FSKindTable,
		ContainedType: &core.FlowType{
			Kind:   core.FSKindRecord,
			Fields: newFields,
		},
	}
}

func (c *SelectColumnsAction) UI(n *core.Node) {
	input, wired := n.GetInputWire(0)

	clay.CLAY(clay.IDI("SelectContent", n.ID), clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          core.GROWH,
			ChildGap:        core.S2,
		},
	}, func() {
		clay.CLAY(clay.IDI("SelectColumnsInOut", n.ID), clay.EL{
			Layout: clay.LAY{
				Sizing:         core.GROWH,
				ChildAlignment: core.YCENTER,
			},
		}, func() {
			core.UIInputPort(n, 0)
			core.UISpacer(clay.IDI("SelectColumnsSpacer", n.ID), core.GROWH)
			core.UIOutputPort(n, 0)
		})

		if wired && input.Type().Kind == core.FSKindTable && input.Type().ContainedType != nil {
			clay.CLAY(clay.IDI("SelectColumnsList", n.ID), clay.EL{
				Layout: clay.LAY{
					LayoutDirection: clay.TopToBottom,
					ChildGap:        core.S1,
				},
			}, func() {
				clay.TEXT("Columns:", clay.TextElementConfig{TextColor: core.LightGray, FontSize: core.F1})

				for i, field := range input.Type().ContainedType.Fields {
					isSelected := slices.Contains(c.SelectedColumns, field.Name)

					core.UIButton(clay.IDI(fmt.Sprintf("SelectColBtn%d", i), n.ID), core.UIButtonConfig{
						El: clay.EL{
							Layout: clay.LAY{ChildGap: core.S2, ChildAlignment: core.YCENTER},
						},
						OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							if isSelected {
								c.SelectedColumns = slices.DeleteFunc(c.SelectedColumns, func(s string) bool { return s == field.Name })
							} else {
								c.SelectedColumns = append(c.SelectedColumns, field.Name)
							}
							n.ClearResult()
						},
					}, func() {
						// Checkbox
						clay.CLAY(clay.IDI(fmt.Sprintf("SelectColCheck%d", i), n.ID), clay.EL{
							Layout:          clay.LAY{Sizing: core.WH(16, 16)},
							Border:          clay.B{Width: core.BA, Color: core.White},
							BackgroundColor: util.Tern(isSelected, core.Blue, clay.Color{}),
						})
						clay.TEXT(field.Name, clay.TextElementConfig{TextColor: core.White})
					})
				}
			})
		} else if !wired {
			clay.TEXT("Connect a table", clay.TextElementConfig{TextColor: core.Gray})
		}
	})
}

func (c *SelectColumnsAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult)
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				select {
				case done <- core.NodeActionResult{Err: fmt.Errorf("panic in node %s: %v", n.Name, r)}:
				case <-ctx.Done():
				}
			}
		}()

		select {
		case <-ctx.Done():
			done <- core.NodeActionResult{Err: ctx.Err()}
			return
		default:
		}

		input, ok, err := n.GetInputValue(0)
		if !ok || err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}

		if input.Type.Kind != core.FSKindTable {
			done <- core.NodeActionResult{Err: fmt.Errorf("input must be a table")}
			return
		}

		// Map selected columns to indices
		var colIndices []int
		var newFields []core.FlowField

		if input.Type.ContainedType != nil {
			for i, field := range input.Type.ContainedType.Fields {
				if slices.Contains(c.SelectedColumns, field.Name) {
					colIndices = append(colIndices, i)
					newFields = append(newFields, field)
				}
			}
		}

		var newRows [][]core.FlowValueField
		for _, row := range input.TableValue {
			// Check context
			select {
			case <-ctx.Done():
				done <- core.NodeActionResult{Err: ctx.Err()}
				return
			default:
			}

			var newRow []core.FlowValueField
			for _, idx := range colIndices {
				newRow = append(newRow, row[idx])
			}
			newRows = append(newRows, newRow)
		}

		done <- core.NodeActionResult{
			Outputs: []core.FlowValue{{
				Type: &core.FlowType{
					Kind: core.FSKindTable,
					ContainedType: &core.FlowType{
						Kind:   core.FSKindRecord,
						Fields: newFields,
					},
				},
				TableValue: newRows,
			}},
		}
	}()
	return done
}

func (c *SelectColumnsAction) Run(n *core.Node) <-chan core.NodeActionResult {
	return c.RunContext(context.Background(), n)
}

func (c *SelectColumnsAction) Serialize(s *core.Serializer) bool {
	n := len(c.SelectedColumns)
	core.SInt(s, &n)
	if !s.Encode {
		c.SelectedColumns = make([]string, n)
	}
	for i := 0; i < n; i++ {
		core.SStr(s, &c.SelectedColumns[i])
	}
	return s.Ok()
}
