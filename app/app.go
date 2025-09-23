package app

import (
	"fmt"

	"github.com/bvisness/flowshell/clay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const windowWidth = 1920
const windowHeight = 1080

func Main() {
	// rl.SetConfigFlags(rl.FlagWindowResizable)

	rl.InitWindow(windowWidth, windowHeight, "Flowshell")
	defer rl.CloseWindow()

	monitorWidth := float32(rl.GetMonitorWidth(rl.GetCurrentMonitor()))
	monitorHeight := float32(rl.GetMonitorHeight(rl.GetCurrentMonitor()))
	// rl.SetWindowSize(windowWidth, windowHeight)
	rl.SetWindowPosition(int(monitorWidth/2-windowWidth/2), int(monitorHeight/2-windowHeight/2))

	rl.SetTargetFPS(int32(rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())))

	arena := clay.CreateArenaWithCapacity(clay.MinMemorySize())
	clay.Initialize(
		arena,
		clay.Dimensions{windowWidth, windowHeight},
		clay.ErrorHandler{ErrorHandlerFunction: handleClayErrors},
	)
	clay.SetMeasureTextFunction(func(str string, config *clay.TextElementConfig, userData any) clay.Dimensions {
		fontSize := config.FontSize
		if fontSize == 0 {
			fontSize = DefaultFontSize
		}
		font := LoadFont(config.FontID, int(fontSize))
		dims := rl.MeasureTextEx(font, str, float32(fontSize), float32(config.LetterSpacing))
		return clay.Dimensions{Width: dims.X, Height: dims.Y}
	}, nil)
	clay.SetDebugModeEnabled(true)

	rl.SetExitKey(0)
	for !rl.WindowShouldClose() {
		frame()
	}
}

func frame() {
	clay.SetLayoutDimensions(clay.D{windowWidth, windowHeight})
	clay.SetPointerState(
		clay.V2{float32(rl.GetMouseX()), float32(rl.GetMouseY())},
		rl.IsMouseButtonDown(rl.MouseButtonLeft),
	)
	clay.UpdateScrollContainers(false, clay.Vector2(rl.GetMouseWheelMoveV()), rl.GetFrameTime())

	clay.BeginLayout()
	{
		ui()
	}
	renderCommands := clay.EndLayout()

	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)
	for _, cmd := range renderCommands {
		bbox := cmd.BoundingBox
		switch cmd.CommandType {
		case clay.RenderCommandTypeRectangle:
			config := cmd.RenderData.Rectangle
			if config.CornerRadius.TopLeft > 0 {
				// The official raylib renderer does some truly insane stuff here that I cannot understand.
				rl.DrawRectangleRounded(rl.Rectangle(bbox), config.CornerRadius.TopLeft, 8, config.BackgroundColor.RGBA())
			} else {
				rl.DrawRectangle(int32(bbox.X), int32(bbox.Y), int32(bbox.Width), int32(bbox.Height), config.BackgroundColor.RGBA())
			}

		case clay.RenderCommandTypeBorder:
			// There's a whole lot of rounding that I'm doing differently here.

			config := cmd.RenderData.Border
			// Left border
			if config.Width.Left > 0 {
				rl.DrawRectangle(int32(bbox.X), int32(bbox.Y+config.CornerRadius.TopLeft), int32(config.Width.Left), int32(bbox.Height-config.CornerRadius.TopLeft-config.CornerRadius.BottomLeft), config.Color.RGBA())
			}
			// Right border
			if config.Width.Right > 0 {
				rl.DrawRectangle(int32(bbox.X+bbox.Width)-int32(config.Width.Right), int32(bbox.Y+config.CornerRadius.TopRight), int32(config.Width.Right), int32(bbox.Height-config.CornerRadius.TopRight-config.CornerRadius.BottomRight), config.Color.RGBA())
			}
			// Top border
			if config.Width.Top > 0 {
				rl.DrawRectangle(int32(bbox.X+config.CornerRadius.TopLeft), int32(bbox.Y), int32(bbox.Width-config.CornerRadius.TopLeft-config.CornerRadius.TopRight), int32(config.Width.Top), config.Color.RGBA())
			}
			// Bottom border
			if config.Width.Bottom > 0 {
				rl.DrawRectangle(int32(bbox.X+config.CornerRadius.BottomLeft), int32(bbox.Y+bbox.Height)-int32(config.Width.Bottom), int32(bbox.Width-config.CornerRadius.BottomLeft-config.CornerRadius.BottomRight), int32(config.Width.Bottom), config.Color.RGBA())
			}
			if config.CornerRadius.TopLeft > 0 {
				rl.DrawRing(rl.Vector2{bbox.X + config.CornerRadius.TopLeft, bbox.Y + config.CornerRadius.TopLeft}, config.CornerRadius.TopLeft-float32(config.Width.Top), config.CornerRadius.TopLeft, 180, 270, 10, config.Color.RGBA())
			}
			if config.CornerRadius.TopRight > 0 {
				rl.DrawRing(rl.Vector2{bbox.X + bbox.Width - config.CornerRadius.TopRight, bbox.Y + config.CornerRadius.TopRight}, config.CornerRadius.TopRight-float32(config.Width.Top), config.CornerRadius.TopRight, 270, 360, 10, config.Color.RGBA())
			}
			if config.CornerRadius.BottomLeft > 0 {
				rl.DrawRing(rl.Vector2{bbox.X + config.CornerRadius.BottomLeft, bbox.Y + bbox.Height - config.CornerRadius.BottomLeft}, config.CornerRadius.BottomLeft-float32(config.Width.Bottom), config.CornerRadius.BottomLeft, 90, 180, 10, config.Color.RGBA())
			}
			if config.CornerRadius.BottomRight > 0 {
				rl.DrawRing(rl.Vector2{bbox.X + bbox.Width - config.CornerRadius.BottomRight, bbox.Y + bbox.Height - config.CornerRadius.BottomRight}, config.CornerRadius.BottomRight-float32(config.Width.Bottom), config.CornerRadius.BottomRight, 0.1, 90, 10, config.Color.RGBA())
			}

		case clay.RenderCommandTypeText:
			text := cmd.RenderData.Text
			fontSize := text.FontSize
			if fontSize == 0 {
				fontSize = DefaultFontSize
			}
			font := LoadFont(text.FontID, int(fontSize))
			rl.DrawTextEx(font, text.StringContents, rl.Vector2(bbox.XY()), float32(fontSize), float32(text.LetterSpacing), text.TextColor.RGBA())

		// TODO: IMAGES

		case clay.RenderCommandTypeScissorStart:
			rl.BeginScissorMode(int32(bbox.X), int32(bbox.Y), int32(bbox.Width), int32(bbox.Height))
		case clay.RenderCommandTypeScissorEnd:
			rl.EndScissorMode()

			// TODO: CUSTOM
		}
	}
	rl.EndDrawing()
}

var ColorLight = clay.Color{224, 215, 210, 255}
var ColorRed = clay.Color{168, 66, 28, 255}
var ColorOrange = clay.Color{225, 138, 50, 255}

func ui() {
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

func sidebarItemComponent() {
	clay.CLAY_AUTO_ID(clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, 0), Height: clay.SizingFixed(50)}}, BackgroundColor: ColorOrange})
}

func handleClayErrors(errorData clay.ErrorData) {
	fmt.Println(errorData.ErrorText)
}
