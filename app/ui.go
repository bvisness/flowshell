package app

import (
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const MenuMinWidth = 200
const MenuMaxWidth = 0 // no max

const NodeMinWidth = 360

var node = &Node{
	ID:  1,
	Pos: V2{100, 100},
	Cmd: NodeCmd{
		CmdString: "curl https://bvisness.me/about/",
	},
}

var ImgPlay rl.Texture2D

func loadImages() {
	ImgPlay = rl.LoadTexture("assets/play.png")
}

var UICursor rl.MouseCursor

func ui() {
	clay.CLAY(clay.ID("Background"), clay.EL{
		Layout:          clay.LAY{Sizing: GROWALL},
		BackgroundColor: Night,
		Border: clay.BorderElementConfig{
			Width: clay.BorderWidth{BetweenChildren: 1},
			Color: Gray,
		},
	}, func() {
		clay.CLAY(clay.ID("NodeCanvas"), clay.EL{
			Layout: clay.LAY{Sizing: GROWALL},
		}, func() {
			UINode(node)
		})
		clay.CLAY(clay.ID("Output"), clay.EL{
			Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(600)}, Padding: PA2},
		}, func() {
			var output []byte
			if err := node.Cmd.Err(); err != nil {
				output = []byte(err.Error())
			} else {
				output = node.Cmd.CombinedOutput()
			}
			if len(output) == 0 {
				output = []byte("No output yet")
			}

			clay.TEXT(string(output), clay.TextElementConfig{FontID: JetBrainsMono, FontSize: F3, TextColor: White})
		})
	})

	rl.SetMouseCursor(UICursor)
	UICursor = rl.MouseCursorDefault
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
		Border: clay.BorderElementConfig{
			Color: Gray,
			Width: BA,
		},
		CornerRadius: RA2,
	}, func() {
		clay.CLAY(clay.ID("NodeHeader"), clay.EL{
			Layout: clay.LAY{
				Sizing:         GROWH,
				Padding:        PD(1, 0, 0, 0, PVH(S1, S2)),
				ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter},
			},
			BackgroundColor: Charcoal,
		}, func() {
			clay.TEXT("Node", clay.TextElementConfig{FontID: InterSemibold, FontSize: F3, TextColor: White})
			UISpacerH()
			if node.Running {
				clay.TEXT("Running...", clay.TextElementConfig{TextColor: White})
			}
			UIButton(clay.ID("PlayButton"),
				UIButtonConfig{
					El: clay.EL{Layout: clay.LAY{Padding: PA1}},
					OnClick: func(elementID clay.ElementID, pointerData clay.PointerData, userData any) {
						node.Run()
					},
				},
				func() {
					clay.CLAY_AUTO_ID(clay.EL{
						Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(float32(ImgPlay.Width)), Height: clay.SizingFixed(float32(ImgPlay.Height))}},
						Image:  clay.ImageElementConfig{ImageData: ImgPlay},
					})

					if clay.Hovered() {
						UITooltip("Run command")
					}
				},
			)
		})
		clay.CLAY(clay.ID("NodeBody"), clay.EL{
			Layout: clay.LAY{Sizing: GROWH, Padding: PA2},
		}, func() {
			UITextBox(clay.ID("Cmd"), &node.Cmd.CmdString, clay.EL{Layout: clay.LAY{Sizing: GROWH}})
		})
	})
}

type UIButtonConfig struct {
	El clay.ElementDeclaration

	OnHover         clay.OnHoverFunc
	OnHoverUserData any

	OnClick         clay.OnHoverFunc
	OnClickUserData any
}

func UIButton(id clay.ElementID, config UIButtonConfig, children ...func()) {
	clay.CLAY_LATE(id, func() clay.ElementDeclaration {
		config.El.CornerRadius = RA1
		config.El.BackgroundColor = util.Tern(clay.Hovered(), clay.Color{255, 255, 255, 20}, clay.Color{})

		clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
			UICursor = rl.MouseCursorPointingHand

			if config.OnHover != nil {
				config.OnHover(elementID, pointerData, config.OnHoverUserData)
			}

			// TODO: Check global UI state to see what UI component the click started on
			if pointerData.State == clay.PointerDataReleasedThisFrame {
				if config.OnClick != nil {
					config.OnClick(elementID, pointerData, config.OnClickUserData)
				}
			}
		}, nil)

		return config.El
	}, children...)
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
		Floating:        clay.FloatingElementConfig{AttachTo: clay.AttachToRoot, Offset: clay.V2(rl.GetMousePosition()).Plus(clay.V2{0, 28})},
		Layout:          clay.LAY{Padding: PA1},
		BackgroundColor: DarkGray,
		Border:          clay.BorderElementConfig{Color: Gray, Width: BA},
	}, func() {
		clay.TEXT(msg, clay.TextElementConfig{TextColor: White})
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
