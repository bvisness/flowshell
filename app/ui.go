package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const MenuMinWidth = 200
const MenuMaxWidth = 0 // no max

const NodeMinWidth = 360

var nodes = []*Node{
	NewRunProcessNode("curl https://bvisness.me/about/"),
	NewListFilesNode("."),
	NewLinesNode(),
	NewLoadFileNode("go.mod"),
	NewTrimSpacesNode(),
}

func init() {
	nodes[0].Pos = V2{10, 10}
	nodes[1].Pos = V2{400, 10}
	nodes[2].Pos = V2{400, 200}
	nodes[3].Pos = V2{10, 200}
}

var wires = []*Wire{
	{
		StartNode: nodes[0], StartPort: 0,
		EndNode: nodes[2], EndPort: 0,
	},
}

var selectedNodeID = 0

func GetSelectedNode() (*Node, bool) {
	for _, node := range nodes {
		if node.ID == selectedNodeID {
			return node, true
		}
	}
	return nil, false
}

func NodeInputs(n *Node) []*Node {
	var res []*Node
	for _, wire := range wires {
		if wire.EndNode == n && !slices.Contains(res, wire.StartNode) {
			res = append(res, wire.StartNode)
		}
	}
	return res
}

var UICursor rl.MouseCursor

func beforeLayout() {
	if rl.IsFileDropped() {
		for i, filename := range rl.LoadDroppedFiles() {
			n := NewLoadFileNode(filename)
			n.Pos = V2(clay.V2(rl.GetMousePosition()).Plus(clay.V2{20, 20}.Times(float32(i))))
			nodes = append(nodes, n)
		}
	}

	for _, n := range nodes {
		// Node drag and drop
		drag.TryStartDrag(n, n.DragRect, n.Pos)
		if draggingThisNode, done, canceled := drag.State(n); draggingThisNode {
			n.Pos = drag.NewObjPosition()
			if done {
				if canceled {
					n.Pos = drag.ObjStart
				}
			}
		}

		// Starting new wires
		for i, portPos := range n.OutputPortPositions {
			portRect := rl.Rectangle{
				X:      portPos.X - PortDragRadius,
				Y:      portPos.Y - PortDragRadius,
				Width:  PortDragRadius * 2,
				Height: PortDragRadius * 2,
			}
			if drag.TryStartDrag(NewWireDragKey, portRect, V2{}) {
				NewWireSourceNode = n
				NewWireSourcePort = i
			}
		}
		for i, portPos := range n.InputPortPositions {
			portRect := rl.Rectangle{
				X:      portPos.X - PortDragRadius,
				Y:      portPos.Y - PortDragRadius,
				Width:  PortDragRadius * 2,
				Height: PortDragRadius * 2,
			}
			if wire, hasWire := n.GetInputWire(i); hasWire {
				if drag.TryStartDrag(NewWireDragKey, portRect, V2{}) {
					wires = slices.DeleteFunc(wires, func(w *Wire) bool { return w == wire })
					NewWireSourceNode = wire.StartNode
					NewWireSourcePort = wire.StartPort
				}
			}
		}
	}

	// Dropping new wires
	if draggingNewWire, done, canceled := drag.State(NewWireDragKey); draggingNewWire {
		if done && !canceled {
			// Loop over nodes to find any you may have dropped on
			for _, node := range nodes {
				for port, portPos := range node.InputPortPositions {
					portRect := rl.Rectangle{
						X:      portPos.X - PortDragRadius,
						Y:      portPos.Y - PortDragRadius,
						Width:  PortDragRadius * 2,
						Height: PortDragRadius * 2,
					}

					if rl.CheckCollisionPointRec(rl.GetMousePosition(), portRect) && node != NewWireSourceNode {
						// Delete existing wires into that port
						wires = slices.DeleteFunc(wires, func(wire *Wire) bool {
							return wire.EndNode == node && wire.EndPort == port
						})
						wires = append(wires, &Wire{
							StartNode: NewWireSourceNode, EndNode: node,
							StartPort: NewWireSourcePort, EndPort: port,
						})
					}
				}
			}
		}
	}
}

const NewWireDragKey = "NEW_WIRE"
const PortDragRadius = 5

var NewWireSourceNode *Node
var NewWireSourcePort int

func ui() {
	// Sweep the graph, validating all nodes
	// TODO: TOPOSORT
	for _, node := range nodes {
		node.Action.UpdateAndValidate(node)
	}

	clay.CLAY(clay.ID("Background"), clay.EL{
		Layout:          clay.LAY{Sizing: GROWALL},
		BackgroundColor: Night,
		Border:          clay.BorderElementConfig{Width: BTW, Color: Gray},
	}, func() {
		clay.CLAY(clay.ID("NodeCanvas"), clay.EL{
			Layout: clay.LAY{Sizing: GROWALL},
		}, func() {
			for _, node := range nodes {
				UINode(node)
			}
		})
		clay.CLAY_LATE(clay.ID("Output"), func() clay.EL {
			return clay.EL{
				Layout: clay.LAY{
					LayoutDirection: clay.TopToBottom,
					Sizing:          clay.Sizing{Width: clay.SizingFixed(windowWidth * 0.30), Height: clay.SizingGrow(1, 0)},
					Padding:         PA2,
				},
				Clip: clay.ClipElementConfig{
					Vertical:    true,
					Horizontal:  true,
					ChildOffset: clay.GetScrollOffset(),
				},
			}
		}, func() {
			if selectedNode, ok := GetSelectedNode(); ok {
				if selectedNode.ResultAvailable {
					result := selectedNode.Result
					if result.Err == nil {
						for outputIndex, output := range result.Outputs {
							port := selectedNode.OutputPorts[outputIndex]
							if err := Typecheck(*output.Type, port.Type); err != nil {
								panic(err)
							}

							clay.TEXT(port.Name, clay.TextElementConfig{FontID: InterSemibold, TextColor: White})
							UIFlowValue(output)
						}
					} else {
						clay.TEXT(result.Err.Error(), clay.TextElementConfig{TextColor: Red})
					}
				}
			}
		})
	})

	rl.SetMouseCursor(UICursor)
	UICursor = rl.MouseCursorDefault
}

func afterLayout() {
	for _, node := range nodes {
		node.UpdateLayoutInfo()
	}
}

func renderOverlays() {
	// Render wires
	for _, wire := range wires {
		rl.DrawLineBezier(
			rl.Vector2(wire.StartNode.OutputPortPositions[wire.StartPort]),
			rl.Vector2(wire.EndNode.InputPortPositions[wire.EndPort]),
			1,
			LightGray.RGBA(),
		)
	}
	if draggingNewWire, _, _ := drag.State(NewWireDragKey); draggingNewWire {
		rl.DrawLineBezier(
			rl.Vector2(NewWireSourceNode.OutputPortPositions[NewWireSourcePort]),
			rl.GetMousePosition(),
			1,
			LightGray.RGBA(),
		)
	}

	for _, node := range nodes {
		for _, portPos := range append(node.InputPortPositions, node.OutputPortPositions...) {
			rl.DrawCircle(int32(portPos.X), int32(portPos.Y), 4, White.RGBA())
		}
	}
}

func UINode(node *Node) {
	clay.CLAY(node.ClayID(), clay.EL{
		Floating: clay.FloatingElementConfig{
			AttachTo: clay.AttachToParent,
			Offset:   clay.Vector2(node.Pos),
		},

		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          clay.Sizing{Width: clay.SizingFit(NodeMinWidth, 0)},
		},
		BackgroundColor: DarkGray,
		Border: util.Tern(selectedNodeID == node.ID,
			clay.BorderElementConfig{
				Color: Blue,
				Width: BA2,
			},
			clay.BorderElementConfig{
				Color: Gray,
				Width: BA,
			},
		),
		CornerRadius: RA2,
	}, func() {
		clay.CLAY_AUTO_ID(clay.EL{ // Node header
			Layout: clay.LAY{
				Sizing:         GROWH,
				Padding:        PD(1, 0, 0, 0, PVH(S1, S2)),
				ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter},
			},
			BackgroundColor: Charcoal,
		}, func() {
			clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
				// TODO: Hook into global system for mouse events
				if pointerData.State == clay.PointerDataReleasedThisFrame {
					selectedNodeID = node.ID
				}
			}, nil)

			clay.TEXT(node.Name, clay.TextElementConfig{FontID: InterSemibold, FontSize: F3, TextColor: White})
			UISpacer(node.DragHandleClayID(), GROWALL)
			if node.Running {
				clay.TEXT("Running...", clay.TextElementConfig{TextColor: White})
			}

			playButtonDisabled := !node.Valid || node.Running
			UIButton(clay.AUTO_ID, // Play button
				UIButtonConfig{
					El:       clay.EL{Layout: clay.LAY{Padding: PA1}},
					Disabled: playButtonDisabled,
					OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						node.Run()
					},
				},
				func() {
					UIImage(clay.AUTO_ID, ImgPlay, clay.EL{
						BackgroundColor: util.Tern(playButtonDisabled, LightGray, PlayButtonGreen),
					})

					if clay.Hovered() {
						UITooltip("Run command")
					}
				},
			)
		})
		clay.CLAY_AUTO_ID(clay.EL{ // Node body
			Layout: clay.LAY{Sizing: GROWH, Padding: PA2},
		}, func() {
			node.Action.UI(node)
		})
	})
}

func UIInputPort(n *Node, port int) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{ChildAlignment: YCENTER},
	}, func() {
		PortAnchor(n, false, port)
		clay.TEXT(n.InputPorts[port].Name, clay.TextElementConfig{TextColor: White})
	})
}

func UIOutputPort(n *Node, port int) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{ChildAlignment: YCENTER},
	}, func() {
		clay.TEXT(n.OutputPorts[port].Name, clay.TextElementConfig{TextColor: White})
		PortAnchor(n, true, port)
	})
}

func UIFlowValue(v FlowValue) {
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
	case FSKindList:
		clay.CLAY_AUTO_ID(clay.EL{ // list items
			Layout: clay.LAY{LayoutDirection: clay.TopToBottom, ChildGap: S1},
		}, func() {
			for i, item := range v.ListValue {
				clay.CLAY_AUTO_ID(clay.EL{ // list item
					Layout: clay.LAY{ChildGap: S2},
				}, func() {
					clay.TEXT(fmt.Sprintf("%d", i), clay.TextElementConfig{FontID: InterSemibold, TextColor: White})
					UIFlowValue(item)
				})
			}
		})
	case FSKindTable:
		clay.CLAY_AUTO_ID(clay.EL{ // Table
			Layout: clay.LAY{LayoutDirection: clay.TopToBottom},
			Border: clay.BorderElementConfig{Width: clay.BorderWidth{Left: 1, Right: 1, Top: 1, Bottom: 1, BetweenChildren: 1}, Color: Gray},
		}, func() {
			clay.CLAY_AUTO_ID(clay.EL{ // Table header
				Border: clay.BorderElementConfig{Width: BTW, Color: Gray},
			}, func() {
				for _, field := range v.Type.ContainedType.Fields {
					clay.CLAY_AUTO_ID(clay.EL{
						Layout: clay.LAY{Padding: PVH(S2, S3)},
					}, func() {
						clay.TEXT(field.Name, clay.TextElementConfig{FontID: InterSemibold, TextColor: White})
					})
				}
			})
			for _, row := range v.TableValue {
				clay.CLAY_AUTO_ID(clay.EL{ // Table row
					Border: clay.BorderElementConfig{Width: BTW, Color: Gray},
				}, func() {
					for _, field := range row {
						clay.CLAY_AUTO_ID(clay.EL{
							Layout: clay.LAY{Padding: PVH(S2, S3)},
						}, func() {
							UIFlowValue(field.Value)
						})
					}
				})
			}
		})
	default:
		clay.TEXT("Unknown data type", clay.TextElementConfig{TextColor: White})
	}
}

type UIButtonConfig struct {
	El       clay.ElementDeclaration
	Disabled bool

	OnHover         clay.OnHoverFunc
	OnHoverUserData any

	OnClick         clay.OnHoverFunc
	OnClickUserData any
}

func UIButton(id clay.ElementID, config UIButtonConfig, children ...func()) {
	clay.CLAY_LATE(id, func() clay.ElementDeclaration {
		config.El.CornerRadius = RA1
		config.El.BackgroundColor = util.Tern(clay.Hovered() && !config.Disabled, HoverWhite, clay.Color{})

		return config.El
	}, func() {
		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
			if !config.Disabled {
				UICursor = rl.MouseCursorPointingHand
			}

			if config.OnHover != nil {
				config.OnHover(elementID, pointerData, config.OnHoverUserData)
			}

			// TODO: Check global UI state to see what UI component the click started on
			if !config.Disabled && pointerData.State == clay.PointerDataReleasedThisFrame {
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
}

func UITextBox(id clay.ElementID, str *string, config UITextBoxConfig) {
	config.El.Border = clay.BorderElementConfig{Width: BA, Color: Gray}
	config.El.Layout.Padding = PVH(S1, S2)
	config.El.BackgroundColor = DarkGray

	clay.CLAY_LATE(id, func() clay.EL {
		if clay.Hovered() {
			UICursor = rl.MouseCursorIBeam
		}
		return config.El
	}, func() {
		clay.TEXT(*str, clay.TextElementConfig{TextColor: util.Tern(config.Disabled, LightGray, White)})
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

func (d *UIDropdown) SelectByValue(v any) {
	for i, opt := range d.Options {
		if opt.Value == v {
			d.Selected = i
			break
		}
	}
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
		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				Padding: PVH(S1, S2),
				Sizing:  GROWH,
			},
		}, func() {
			clay.TEXT(d.GetSelectedOption().Name, clay.TextElementConfig{TextColor: White})
		})
		UIButton(clay.AUTO_ID, UIButtonConfig{
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
			UIImage(clay.AUTO_ID, util.Tern(d.open, ImgDropdownUp, ImgDropdownDown), clay.EL{
				BackgroundColor: LightGray,
			})
		})

		if d.open {
			clay.CLAY_AUTO_ID(clay.EL{
				Layout: clay.LAY{
					LayoutDirection: clay.TopToBottom,
					Sizing:          GROWH,
				},
				Floating: clay.FLOAT{
					AttachTo:     clay.AttachToParent,
					AttachPoints: clay.FloatingAttachPoints{Parent: clay.AttachPointLeftBottom},
					ZIndex:       ZTOP,
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
				for i, opt := range d.Options {
					clay.CLAY_AUTO_ID_LATE(func() clay.EL {
						return clay.EL{
							Layout: clay.LAY{
								Padding: PVH(S1, S2),
								Sizing:  GROWH,
							},
							BackgroundColor: util.Tern(clay.Hovered(), HoverWhite, clay.Color{}),
						}
					}, func() {
						clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
							if pointerData.State == clay.PointerDataReleasedThisFrame {
								selectedBefore := d.Selected
								d.Selected = userData.(int)
								d.open = false
								if config.OnChange != nil {
									config.OnChange(d.GetOption(selectedBefore).Value, d.GetOption(d.Selected).Value)
								}
							}
						}, i)
						clay.TEXT(opt.Name, clay.T{TextColor: White})
					})
				}
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
	clay.CLAY(clay.ID("Tooltip"), clay.EL{
		Floating: clay.FloatingElementConfig{
			AttachTo: clay.AttachToRoot,
			Offset:   clay.V2(rl.GetMousePosition()).Plus(clay.V2{0, 28}),
			ZIndex:   ZTOP,
		},
		Layout:          clay.LAY{Padding: PA1},
		BackgroundColor: DarkGray,
		Border:          clay.BorderElementConfig{Color: Gray, Width: BA},
	}, func() {
		clay.TEXT(msg, clay.TextElementConfig{TextColor: White})
	})
}

func PortAnchorID(node *Node, isOutput bool, port int) clay.ElementID {
	return clay.ID(fmt.Sprintf("N%d%s%d", node.ID, util.Tern(isOutput, "O", "I"), port))
}

func PortAnchor(node *Node, isOutput bool, port int) {
	clay.CLAY(PortAnchorID(node, isOutput, port), clay.EL{})
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

func menu() {
	clay.CLAY(clay.ID("RightClickMenu"), clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          clay.Sizing{Width: clay.SizingFit(MenuMinWidth, MenuMaxWidth)},
		},
		Floating:        clay.FloatingElementConfig{AttachTo: clay.AttachToRoot, Offset: clay.V2{float32(rl.GetMouseX()), float32(rl.GetMouseY())}},
		BackgroundColor: DarkGray,
	}, func() {
		clay.CLAY(clay.ID("thing"), clay.EL{Layout: clay.LayoutConfig{Padding: PVH(S2, S3)}}, func() {
			clay.TEXT("Hi! I'm text!", clay.TextElementConfig{TextColor: White})
		})
	})
}

func clayExample() {
	var ColorLight = clay.Color{224, 215, 210, 255}
	var ColorRed = clay.Color{168, 66, 28, 255}
	var ColorOrange = clay.Color{225, 138, 50, 255}

	sidebarItemComponent := func() {
		clay.CLAY_AUTO_ID(clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, 0), Height: clay.SizingFixed(50)}}, BackgroundColor: ColorOrange})
	}

	clay.CLAY(clay.ID("OuterContainer"), clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{clay.SizingGrow(0, 0), clay.SizingGrow(0, 0)}, Padding: clay.PaddingAll(16), ChildGap: 16}, BackgroundColor: clay.Color{250, 250, 255, 255}}, func() {
		clay.CLAY(clay.ID("Sidebar"), clay.EL{
			Layout:          clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(300), Height: clay.SizingGrow(0, 0)}, Padding: clay.PaddingAll(16), ChildGap: 16},
			BackgroundColor: ColorLight,
		}, func() {
			clay.CLAY(clay.ID("ProfilePictureOuter"), clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, 0)}, Padding: clay.PaddingAll(16), ChildGap: 16, ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter}}, BackgroundColor: ColorRed}, func() {
				clay.TEXT("Clay - UI Library", clay.TextElementConfig{FontID: InterBold, FontSize: 24, TextColor: clay.Color{255, 255, 255, 255}})
			})

			for range 5 {
				sidebarItemComponent()
			}
		})
		clay.CLAY(clay.ID("MainContent"), clay.EL{Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingGrow(0, 0), Height: clay.SizingGrow(0, 0)}, Padding: clay.PaddingAll(16), ChildGap: 8}, BackgroundColor: ColorLight}, func() {
			for f := range FontsEnd {
				clay.TEXT(fontFiles[f], clay.TextElementConfig{FontID: f, TextColor: clay.Color{0, 0, 0, 255}})
			}
		})
	})
}
