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

func ui() {
	nodes[0].Pos = V2{10, 10}
	nodes[1].Pos = V2{400, 10}
	nodes[2].Pos = V2{400, 200}

	// Sweep the graph, validating all nodes
	// TODO: TOPOSORT
	for _, node := range nodes {
		node.Action.Validate(node)
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
					Sizing:          clay.Sizing{Width: clay.SizingFixed(600), Height: clay.SizingGrow(1, 0)},
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
							if port.Type.Kind != output.Type.Kind {
								panic(fmt.Errorf("mismatched types: expected %v, got %v", port.Type.Kind, output.Type.Kind))
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

func UINode(node *Node) {
	clay.CLAY(clay.IDI("Node", node.ID), clay.EL{
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
			UISpacerH()
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
					clay.CLAY_AUTO_ID(clay.EL{
						Layout:          clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(float32(ImgPlay.Width)), Height: clay.SizingFixed(float32(ImgPlay.Height))}},
						Image:           clay.ImageElementConfig{ImageData: ImgPlay},
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
			Layout: clay.LAY{LayoutDirection: clay.TopToBottom, ChildGap: S2},
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
		config.El.BackgroundColor = util.Tern(clay.Hovered() && !config.Disabled, clay.Color{255, 255, 255, 20}, clay.Color{})

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

func UITextBox(id clay.ElementID, str *string, decl clay.ElementDeclaration) {
	decl.Border = clay.BorderElementConfig{Width: BA, Color: Gray}
	decl.Layout.Padding = PVH(S1, S2)
	decl.BackgroundColor = DarkGray

	clay.CLAY_LATE(id, func() clay.EL {
		if clay.Hovered() {
			UICursor = rl.MouseCursorIBeam
		}
		return decl
	}, func() {
		clay.TEXT(*str, clay.TextElementConfig{TextColor: White})
	})
}

func UISpacerH() {
	clay.CLAY_AUTO_ID(clay.EL{Layout: clay.LAY{Sizing: GROWH}})
}

func UITooltip(msg string) {
	clay.CLAY(clay.ID("Tooltip"), clay.EL{
		Floating: clay.FloatingElementConfig{
			AttachTo: clay.AttachToRoot,
			Offset:   clay.V2(rl.GetMousePosition()).Plus(clay.V2{0, 28}),
			ZIndex:   1000,
		},
		Layout:          clay.LAY{Padding: PA1},
		BackgroundColor: DarkGray,
		Border:          clay.BorderElementConfig{Color: Gray, Width: BA},
	}, func() {
		clay.TEXT(msg, clay.TextElementConfig{TextColor: White})
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
