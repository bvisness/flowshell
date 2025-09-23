package app

import (
	"fmt"

	"github.com/bvisness/flowshell/clay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var screenWidth = float32(1920)
var screenHeight = float32(1080)

const fontSize = 24

const (
	FRegular = iota
	FSemibold
	FBold

	FEndWeights
)

var font [FEndWeights]rl.Font

func Main() {
	rl.SetConfigFlags(rl.FlagWindowResizable)

	rl.InitWindow(int32(screenWidth), int32(screenHeight), "Flowshell")
	defer rl.CloseWindow()

	monWidth := float32(rl.GetMonitorWidth(rl.GetCurrentMonitor()))
	monHeight := float32(rl.GetMonitorHeight(rl.GetCurrentMonitor()))
	rl.SetWindowSize(int(monWidth*0.8), int(monHeight*0.8))
	rl.SetWindowPosition(int(monWidth*0.1), int(monHeight*0.1))

	rl.SetTargetFPS(int32(rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())))

	font[FRegular] = rl.LoadFontEx("assets/Inter-Regular.ttf", fontSize, nil)
	font[FSemibold] = rl.LoadFontEx("assets/Inter-SemiBold.ttf", fontSize, nil)
	font[FBold] = rl.LoadFontEx("assets/Inter-Bold.ttf", fontSize, nil)

	arena := clay.CreateArenaWithCapacity(clay.MinMemorySize())
	clay.Initialize(
		arena,
		clay.Dimensions{screenWidth, screenHeight},
		clay.ErrorHandler{ErrorHandlerFunction: handleClayErrors},
	)
	clay.SetMeasureTextFunction(func(str string, config *clay.TextElementConfig, userData any) clay.Dimensions {
		// TODO: Use font from command
		dims := rl.MeasureTextEx(font[FRegular], str, float32(config.FontSize), float32(config.LetterSpacing))
		return clay.Dimensions{dims.X, dims.Y}
	}, nil)
	clay.SetDebugModeEnabled(true)

	rl.SetExitKey(0)
	for !rl.WindowShouldClose() {
		frame()
	}
}

func frame() {
	clay.SetLayoutDimensions(clay.D{screenWidth, screenHeight})
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
			// TODO: use font ID from command
			rl.DrawTextEx(font[FRegular], text.StringContents, rl.Vector2(bbox.XY()), float32(text.FontSize), float32(text.LetterSpacing), text.TextColor.RGBA())

		// TODO: IMAGES

		case clay.RenderCommandTypeScissorStart:
			rl.BeginScissorMode(int32(bbox.X), int32(bbox.Y), int32(bbox.Width), int32(bbox.Height))
		case clay.RenderCommandTypeScissorEnd:
			rl.EndScissorMode()

			// TODO: CUSTOM
		}
	}
	rl.EndDrawing()

	// rl.BeginDrawing()
	// {
	// 	rl.ClearBackground(rl.RayWhite)
	// 	drawText(font[FRegular], "Congrats! You created your first window!", 190, 200, rl.Black)
	// 	drawText(font[FBold], "And this text is bold, which is swankier.", 190, 224, rl.Black)
	// 	drawText(font[FSemibold], "This, on the other hand? Semibold. Half the swank, double the swagger.", 190, 248, rl.Black)
	// }
	// rl.EndDrawing()
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
				clay.TEXT("Clay - UI Library", clay.TextElementConfig{FontSize: 24, TextColor: clay.Color{255, 255, 255, 255}})
			})

			for i := 0; i < 5; i++ {
				sidebarItemComponent()
			}
		})
		clay.CLAY(clay.ID("MainContent"), clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, 0), Height: clay.SizingGrow(0, 0)}}, BackgroundColor: ColorLight})
	})
}

func sidebarItemComponent() {
	clay.CLAY_AUTO_ID(clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, 0), Height: clay.SizingFixed(50)}}, BackgroundColor: ColorOrange})
}

func drawText(font rl.Font, text string, x float32, y float32, color rl.Color) {
	rl.DrawTextEx(font, text, rl.Vector2{X: x, Y: y}, fontSize, 0, color)
}

func handleClayErrors(errorData clay.ErrorData) {
	fmt.Println(errorData.ErrorText)
}
