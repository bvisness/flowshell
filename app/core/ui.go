package core

import (
	"fmt"
	"time"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Helpers and styles are in styles.go

// PushHistoryFunc should be set by app
var PushHistoryFunc func()

func PushHistory() {
	if PushHistoryFunc != nil {
		PushHistoryFunc()
	}
}

// Focus tracking
var UIFocus *clay.ElementID
var LastUIFocus clay.ElementID
var LastUIFocusValid bool

func IsFocused(id clay.ElementID) bool {
	return UIFocus != nil && id.ID == UIFocus.ID
}

// ---------------------------
// Widgets

type UIButtonConfig struct {
	El       clay.ElementDeclaration
	Disabled bool

	OnHover         clay.OnHoverFunc
	OnHoverUserData any

	OnClick         clay.OnHoverFunc
	OnClickUserData any

	ZIndex int16
}

func UIButton(id clay.ElementID, config UIButtonConfig, children ...func()) {
	clay.CLAY_LATE(id, func() clay.ElementDeclaration {
		config.El.CornerRadius = RA1
		config.El.BackgroundColor = util.Tern(clay.Hovered() && !config.Disabled, HoverWhite, clay.Color{})

		return config.El
	}, func() {
		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
			IsHoveringUI = true

			if !config.Disabled {
				UICursor = rl.MouseCursorPointingHand
				z := config.ZIndex
				if CurrentZIndex > z {
					z = CurrentZIndex
				}
				// Bump Z-index to ensure buttons are always above their background context (e.g. Node body)
				// Use max to ensure we don't go below context.
				// We rely on render order (last wins) for same Z-index.
				UIInput.RegisterPointerDown(elementID, pointerData, z)
			}

			if config.OnHover != nil {
				config.OnHover(elementID, pointerData, config.OnHoverUserData)
			}

			if !config.Disabled && UIInput.IsClick(elementID, pointerData) {
				UIFocus = nil
				if config.OnClick != nil {
					config.OnClick(elementID, pointerData, config.OnClickUserData)
				}
			}
		}, nil)

		for _, f := range children {
			f()
		}
	})
}

type UITextBoxConfig struct {
	El       clay.EL
	Disabled bool

	OnSubmit func(val string)

	ZIndex int16
}

func UITextBox(id clay.ElementID, str *string, config UITextBoxConfig, children ...func()) {
	// ... (focus logic stays same)
	if IsFocused(id) {
		if config.Disabled {
			UIFocus = nil
		} else {
			if rl.IsKeyPressed(rl.KeyEnter) {
				if config.OnSubmit != nil {
					config.OnSubmit(*str)
				}
				PushHistory() // Snapshot history on Submit

				UIFocus = nil
			} else {
				for r := rl.GetCharPressed(); r != 0; r = rl.GetCharPressed() {
					*str = *str + string(rune(r))
				}
				if rl.IsKeyPressed(rl.KeyBackspace) || rl.IsKeyPressedRepeat(rl.KeyBackspace) {
					if len(*str) > 0 {
						*str = (*str)[:len(*str)-1]
					}
				}
			}
		}
	}

	// Detect Blur
	if LastUIFocusValid && LastUIFocus == id && !IsFocused(id) {
		PushHistory()
	}

	clay.CLAY_LATE(id, func() clay.EL {
		config.El.Border = clay.B{Width: BA, Color: Gray}
		config.El.Layout.Padding = PVH(S1, S2)
		config.El.Layout.ChildAlignment.Y = clay.AlignYCenter
		config.El.BackgroundColor = DarkGray
		config.El.Clip = clay.CLIP{Horizontal: true}

		if clay.Hovered() {
			UICursor = rl.MouseCursorIBeam
		}

		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
			IsHoveringUI = true
			z := config.ZIndex
			if CurrentZIndex > z {
				z = CurrentZIndex
			}
			UIInput.RegisterPointerDown(elementID, pointerData, z+Z_OFFSET_INPUT_PRIORITY)

			if UIInput.IsClick(elementID, pointerData) {
				UIFocus = &elementID
			}
		}, nil)

		return config.El
	}, func() {
		clay.TEXT(*str, clay.T{TextColor: util.Tern(config.Disabled, LightGray, White)})
		// Fix: Use ID derived from parent ID instead of AUTO_ID
		UISpacer(clay.IDI("Spacer", int(id.ID)), WH(1, 16))
		if IsFocused(id) {
			clay.CLAY(clay.IDI("Cursor", int(id.ID)), clay.EL{
				Layout:          clay.LAY{Sizing: WH(2, 16)},
				BackgroundColor: White,
			}, func() {})
		}

		for _, f := range children {
			f()
		}
	})
}

type OnChangeFunc func(before, after any)

type UIDropdown struct {
	Options  []UIDropdownOption
	Selected int

	open bool
}

type UIDropdownOption struct {
	Name  string
	Value any
}

func (d *UIDropdown) GetOption(i int) UIDropdownOption {
	if len(d.Options) == 0 {
		return UIDropdownOption{}
	}
	if i >= len(d.Options) {
		return d.Options[0]
	}
	return d.Options[i]
}

func (d *UIDropdown) GetSelectedOption() UIDropdownOption {
	return d.GetOption(d.Selected)
}

func (d *UIDropdown) SelectByName(name string) bool {
	for i, opt := range d.Options {
		if opt.Name == name {
			d.Selected = i
			return true
		}
	}
	return false
}

func (d *UIDropdown) SelectByValue(v any) bool {
	for i, opt := range d.Options {
		if opt.Value == v {
			d.Selected = i
			return true
		}
	}
	return false
}

type UIDropdownConfig struct {
	El       clay.EL
	OnChange OnChangeFunc
}

func (d *UIDropdown) Do(id clay.ElementID, config UIDropdownConfig) {
	config.El.Layout.Padding = clay.Padding{}
	config.El.Layout.ChildAlignment.Y = clay.AlignYCenter
	config.El.Border = clay.BorderElementConfig{Width: BA, Color: Gray}
	config.El.BackgroundColor = DarkGray

	clay.CLAY(id, config.El, func() {
		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
			IsHoveringUI = true
		}, nil)

		// Label wrapper with seeded ID
		clay.CLAY(clay.IDI("LabelWrap", int(id.ID)), clay.EL{
			Layout: clay.LAY{
				Padding: PVH(S1, S2),
				Sizing:  GROWH,
			},
		}, func() {
			clay.TEXT(d.GetSelectedOption().Name, clay.TextElementConfig{TextColor: White})
		})

		// Button with seeded ID
		UIButton(clay.IDI("ArrowBtn", int(id.ID)), UIButtonConfig{
			El: clay.EL{
				Layout: clay.LAY{
					ChildAlignment: ALLCENTER,
					Sizing:         GROWV,
					Padding:        PA2,
				},
				Border: clay.B{Width: clay.BW{Left: 1}, Color: Gray},
			},
			OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
				d.open = !d.open
			},
		}, func() {
			UIImage(clay.IDI("ArrowIcon", int(id.ID)), util.Tern(d.open, ImgDropdownUp, ImgDropdownDown), clay.EL{
				BackgroundColor: LightGray,
			})
		})

		if d.open {
			WithZIndex(Z_DROPDOWN_FG, func() {
				// Transparent blocker to close dropdown when clicking outside
				clay.CLAY(id.With("blocker"), clay.EL{
					Layout: clay.LAY{
						Sizing: WH(float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())),
					},
					Floating: clay.FloatingElementConfig{
						AttachTo: clay.AttachToRoot,
						ZIndex:   Z_DROPDOWN_BG,
					},
				}, func() {
					clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						UIInput.RegisterPointerDown(elementID, pointerData, Z_DROPDOWN_BG)
						if UIInput.IsClick(elementID, pointerData) {
							d.open = false
						}
					}, nil)
				})

				// Dropdown list with seeded ID
				clay.CLAY(clay.IDI("List", int(id.ID)), clay.EL{
					Layout: clay.LAY{
						LayoutDirection: clay.TopToBottom,
						Sizing:          GROWH,
					},
					Floating: clay.FLOAT{
						AttachTo:     clay.AttachToParent,
						AttachPoints: clay.FloatingAttachPoints{Parent: clay.AttachPointLeftBottom},
						ZIndex:       Z_DROPDOWN_FG,
						// PointerCaptureMode: clay.PointercaptureModeCapture,
					},
					Border: clay.B{
						Width: clay.BW{
							Top:  0,
							Left: 1, Right: 1,
							Bottom: 1,

							BetweenChildren: 1,
						},
						Color: Gray,
					},
					BackgroundColor: DarkGray,
				}, func() {
					clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						IsHoveringUI = true
					}, nil)

					for i, opt := range d.Options {
						// Capture for closure
						idx := i
						val := opt.Value

						UIButton(clay.IDI("DropdownOpt", i+int(id.ID)), UIButtonConfig{
							El: clay.EL{
								Layout: clay.LAY{
									Padding: PVH(S1, S2),
									Sizing:  GROWH,
								},
							},
							ZIndex: Z_DROPDOWN_FG,
							OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
								selectedBefore := d.Selected
								d.Selected = idx
								d.open = false
								if d.Selected != selectedBefore {
									PushHistory()
								}
								if config.OnChange != nil {
									config.OnChange(d.GetOption(selectedBefore).Value, val)
								}
							},
						}, func() {
							clay.TEXT(opt.Name, clay.TextElementConfig{TextColor: White})
						})
					}
				})
			})
		}
	})
}

func UISpacer(id clay.ElementID, sizing clay.Sizing) {
	clay.CLAY(id, clay.EL{Layout: clay.LAY{Sizing: sizing}})
}

func UIImage(id clay.ElementID, img rl.Texture2D, decl clay.EL) {
	decl.Layout.Sizing = WH(float32(img.Width), float32(img.Height))
	decl.Image = clay.ImageElementConfig{ImageData: img}
	clay.CLAY(id, decl)
}

func UITooltip(msg string) {
	clay.CLAY_AUTO_ID(clay.EL{
		Floating: clay.FloatingElementConfig{
			AttachTo: clay.AttachToRoot,
			Offset:   clay.V2(rl.GetMousePosition()).Plus(clay.V2{X: 0, Y: 28}),
			ZIndex:   Z_TOOLTIP,
		},
		Layout:          clay.LAY{Padding: PA1},
		BackgroundColor: DarkGray,
		Border:          clay.BorderElementConfig{Color: Gray, Width: BA},
	}, func() {
		clay.TEXT(msg, clay.TextElementConfig{TextColor: White})
	})
}

func UICheckbox(id clay.ElementID, checked *bool, label string) {
	clay.CLAY(clay.IDI("CheckboxWrapper", int(id.ID)), clay.EL{
		Layout: clay.LAY{ChildGap: S1, ChildAlignment: YCENTER},
	}, func() {
		UIButton(id, UIButtonConfig{
			OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
				*checked = !*checked
				PushHistory()
			},
		}, func() {
			UIImage(clay.IDI("CheckIcon", int(id.ID)), util.Tern(*checked, ImgToggleDown, ImgToggleRight), clay.EL{})
		})
		clay.TEXT(label, clay.TextElementConfig{TextColor: White})
	})
}

// PortAnchorID is in node.go

func PortAnchor(node *Node, isOutput bool, port int) {
	// Give it a small non-zero size to ensure it has a valid bounding box for layout calculations
	// 12x12 ensures easy clicking
	clay.CLAY(PortAnchorID(node, isOutput, port), clay.EL{
		Layout:          clay.LAY{Sizing: WH(12, 12)},
		BackgroundColor: LightGray,
		CornerRadius:    RA(6),
	})
}

func UIInputPort(n *Node, port int) {
	clay.CLAY(clay.IDI(fmt.Sprintf("InputPort%d", port), n.ID), clay.EL{
		Layout: clay.LAY{ChildAlignment: YCENTER},
	}, func() {
		PortAnchor(n, false, port)
		clay.TEXT(n.InputPorts[port].Name, clay.TextElementConfig{TextColor: White})
	})
}

func UIOutputPort(n *Node, port int) {
	clay.CLAY(clay.IDI(fmt.Sprintf("OutputPort%d", port), n.ID), clay.EL{
		Layout: clay.LAY{ChildAlignment: YCENTER},
	}, func() {
		clay.TEXT(n.OutputPorts[port].Name, clay.TextElementConfig{TextColor: White})
		PortAnchor(n, true, port)
	})
}

func FormatBytes(n int64) string {
	if n < 1_000 {
		return fmt.Sprintf("%d B", n)
	} else if n < 1_000_000 {
		return fmt.Sprintf("%.1f KB", float32(n)/1_000)
	} else if n < 1_000_000_000 {
		return fmt.Sprintf("%.1f MB", float32(n)/1_000_000)
	} else if n < 1_000_000_000_000 {
		return fmt.Sprintf("%.1f GB", float32(n)/1_000_000_000)
	} else {
		return fmt.Sprintf("%.1f TB", float32(n)/1_000_000_000_000)
	}
}

func UIFlowValue(seed clay.ElementID, v FlowValue) {
	clay.CLAY(seed, clay.EL{}, func() {
		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
			if data, ok := clay.GetElementData(elementID); ok {
				Drag.TryStartDrag(FlowValueDrag{Value: v}, rl.Rectangle(data.BoundingBox), V2{})
			}
		}, nil)

		switch v.Type.Kind {
		case FSKindBytes:
			if len(v.BytesValue) == 0 {
				clay.TEXT("<no data>", clay.TextElementConfig{FontID: JetBrainsMono, TextColor: LightGray})
			} else {
				clay.TEXT(string(v.BytesValue), clay.TextElementConfig{FontID: JetBrainsMono, TextColor: White})
			}
		case FSKindInt64:
			var str string
			if v.Type.WellKnownType == FSWKTTimestamp {
				str = time.Unix(v.Int64Value, 0).Format(time.RFC1123)
			} else if v.Type.Unit == FSUnitBytes {
				str = FormatBytes(v.Int64Value)
			} else {
				str = fmt.Sprintf("%d", v.Int64Value)
			}
			clay.TEXT(str, clay.TextElementConfig{TextColor: White})
		case FSKindFloat64:
			var str string
			str = fmt.Sprintf("%v", v.Float64Value)
			clay.TEXT(str, clay.TextElementConfig{TextColor: White})
		case FSKindList:
			clay.CLAY(seed.With("ListGen"), clay.EL{ // list items
				Layout: clay.LAY{LayoutDirection: clay.TopToBottom, ChildGap: S1},
			}, func() {
				for i, item := range v.ListValue {
					itemSeed := seed.WithI("Item", i)
					clay.CLAY(itemSeed, clay.EL{ // list item
						Layout: clay.LAY{ChildGap: S2},
					}, func() {
						clay.TEXT(fmt.Sprintf("%d", i), clay.TextElementConfig{FontID: InterSemibold, TextColor: White})
						UIFlowValue(itemSeed.With("Val"), item)
					})
				}
			})
		case FSKindTable:
			clay.CLAY(seed.With("TableGen"), clay.EL{ // Table
				Border: clay.B{Width: BA_BTW, Color: Gray},
			}, func() {
				for col, field := range v.Type.ContainedType.Fields {
					// Use proper seeding for column to avoid global collisions
					colSeed := seed.WithI("Col", col)
					clay.CLAY(colSeed, clay.EL{ // Table col
						Layout: clay.LAY{LayoutDirection: clay.TopToBottom},
						Border: clay.B{Width: BTW, Color: Gray},
					}, func() {
						clay.CLAY(colSeed.WithI("HeaderGen", col), clay.EL{ // Header cell
							Layout: clay.LAY{Padding: PVH(S2, S3)},
						}, func() {
							clay.TEXT(field.Name, clay.TextElementConfig{FontID: InterSemibold, TextColor: White})
						})

						for row, val := range v.ColumnValues(col) {
							cellSeed := colSeed.WithI("Cell", row)
							clay.CLAY(cellSeed, clay.EL{
								Layout: clay.LAY{Padding: PVH(S2, S3)},
							}, func() {
								UIFlowValue(cellSeed.With("Val"), val)
							})
						}
					})
				}
			})
		case FSKindRecord:
			clay.CLAY(seed.With("RecordGen"), clay.EL{
				Layout: clay.LAY{LayoutDirection: clay.TopToBottom, ChildGap: S1},
			}, func() {
				for i, field := range v.RecordValue {
					fieldSeed := seed.WithI("Field", i)
					clay.CLAY(fieldSeed, clay.EL{
						Layout: clay.LAY{ChildGap: S2, ChildAlignment: YCENTER},
					}, func() {
						clay.TEXT(field.Name+":", clay.TextElementConfig{FontID: InterSemibold, TextColor: LightGray})
						UIFlowValue(fieldSeed.With("Val"), field.Value)
					})
				}
			})
		case FSKindStream:
			clay.TEXT("<stream>", clay.TextElementConfig{FontID: JetBrainsMono, TextColor: LightGray})
		default:
			clay.TEXT(fmt.Sprintf("Unknown data type: %v", v.Type.Kind), clay.TextElementConfig{TextColor: White})
		}
	})
}

type FlowValueDrag struct {
	NodeID int
	Port   int
	Value  FlowValue
}

func (d FlowValueDrag) DragKey() string {
	return fmt.Sprintf("FlowValueDrag-%d-%d", d.NodeID, d.Port)
}
