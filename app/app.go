package app

import rl "github.com/gen2brain/raylib-go/raylib"

var screenWidth = float32(1920)
var screenHeight = float32(1080)

const fontSize = 20

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

	rl.SetExitKey(0)
	for !rl.WindowShouldClose() {
		frame()
	}
}

func frame() {
	rl.BeginDrawing()

	rl.ClearBackground(rl.RayWhite)
	drawText(font[FRegular], "Congrats! You created your first window!", 190, 200, rl.Black)
	drawText(font[FBold], "And this text is bold, which is swankier.", 190, 224, rl.Black)
	drawText(font[FSemibold], "This, on the other hand? Semibold. Half the swank, double the swagger.", 190, 248, rl.Black)

	rl.EndDrawing()
}

func drawText(font rl.Font, text string, x float32, y float32, color rl.Color) {
	rl.DrawTextEx(font, text, rl.Vector2{X: x, Y: y}, fontSize, 0, color)
}
