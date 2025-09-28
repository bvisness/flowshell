package app

import (
	"fmt"

	"github.com/bvisness/flowshell/clay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const windowWidth = 1920
const windowHeight = 1080

// const windowWidth = 1280
// const windowHeight = 720

func Main() {
	// rl.SetConfigFlags(rl.FlagWindowResizable)

	rl.InitWindow(windowWidth, windowHeight, "Flowshell")
	defer rl.CloseWindow()

	monitorWidth := float32(rl.GetMonitorWidth(rl.GetCurrentMonitor()))
	monitorHeight := float32(rl.GetMonitorHeight(rl.GetCurrentMonitor()))
	// rl.SetWindowSize(windowWidth, windowHeight)
	rl.SetWindowPosition(int(monitorWidth/2-windowWidth/2), int(monitorHeight/2-windowHeight/2))
	rl.SetTargetFPS(int32(rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())))

	loadImages()

	clay.SetMaxElementCount(1 << 18)
	arena := clay.CreateArenaWithCapacity(uintptr(clay.MinMemorySize()))
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
	drag.Update()
	beforeLayout()

	clay.SetLayoutDimensions(clay.D{windowWidth, windowHeight})
	clay.SetPointerState(
		clay.V2{float32(rl.GetMouseX()), float32(rl.GetMouseY())},
		rl.IsMouseButtonDown(rl.MouseButtonLeft),
	)
	clay.UpdateScrollContainers(false, clay.Vector2(rl.GetMouseWheelMoveV()).Times(4), rl.GetFrameTime())

	clay.BeginLayout()
	{
		ui()
	}
	commands := clay.EndLayout()

	afterLayout()

	rl.BeginDrawing()
	rl.ClearBackground(rl.RayWhite)
	renderClayCommands(commands)
	renderOverlays()
	rl.EndDrawing()
}

func handleClayErrors(errorData clay.ErrorData) {
	fmt.Printf("CLAY ERROR: %s\n", errorData.ErrorText)
}
