package app

import (
	"github.com/bvisness/flowshell/clay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const MenuMinWidth = 200
const MenuMaxWidth = 0 // no max

const NodeMinWidth = 360

var node = &Node{
	ID:  1,
	Pos: V2{100, 100},
	Cmd: NodeCmd{
		Cmd: "test --foo",
	},
}

var ImgPlay rl.Texture2D

func loadImages() {
	ImgPlay = rl.LoadTexture("assets/play.png")
}

func ui() {
	clay.CLAY(clay.ID("Background"), clay.EL{
		Layout: clay.LAY{
			Sizing: clay.Sizing{Width: clay.SizingGrow(1, 0), Height: clay.SizingGrow(1, 0)},
		},
		BackgroundColor: Night,
	}, func() {
		UINode(node)
	})
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
			clay.CLAY(clay.ID("PlayButton"), clay.EL{
				Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(float32(ImgPlay.Width)), Height: clay.SizingFixed(float32(ImgPlay.Height))}},
				Image:  clay.ImageElementConfig{ImageData: ImgPlay},
			})
		})
		clay.CLAY(clay.ID("NodeBody"), clay.EL{
			Layout: clay.LAY{Sizing: GROWH, Padding: PA2},
		}, func() {
			UITextBox(clay.ID("Cmd"), &node.Cmd.Cmd, clay.EL{Layout: clay.LAY{Sizing: GROWH}})
		})
	})
}

func UITextBox(id clay.ElementID, str *string, decl clay.ElementDeclaration) {
	decl.Border = clay.BorderElementConfig{Width: BA, Color: Gray}
	decl.Layout.Padding = PVH(S1, S2)
	decl.BackgroundColor = DarkGray

	clay.CLAY(id, decl, func() {
		clay.TEXT(*str, clay.TextElementConfig{TextColor: White})
	})
}

func UISpacerH() {
	clay.CLAY_AUTO_ID(clay.EL{Layout: clay.LAY{Sizing: GROWH}})
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
