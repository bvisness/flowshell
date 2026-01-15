package app

import (
	"flag"
	"fmt"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/util"

	_ "github.com/bvisness/flowshell/app/nodes"
	"github.com/bvisness/flowshell/clay"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const windowWidth = 1920

var ShouldQuit bool

func Main() {
	// Parse CLI arguments
	debugPtr := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	core.DebugMode = *debugPtr

	CurrentSettings = LoadSettings()
	core.ApplyTheme(CurrentSettings.Theme)

	rl.SetTraceLogLevel(rl.LogError)
	rl.SetConfigFlags(rl.FlagWindowResizable)
	// rl.InitWindow(windowWidth, windowHeight, "Flowshell")
	rl.InitWindow(int32(CurrentSettings.WindowWidth), int32(CurrentSettings.WindowHeight), "Flowshell")
	defer rl.CloseWindow()

	if CurrentSettings.WindowMaximized {
		rl.MaximizeWindow()
	}

	monitorWidth := float32(rl.GetMonitorWidth(rl.GetCurrentMonitor()))
	monitorHeight := float32(rl.GetMonitorHeight(rl.GetCurrentMonitor()))

	// Center if not maximized and looks like default or sane
	// Actually InitWindow usually centers or puts it somewhere.
	// If we want to restore position we need to save it too.
	// User only asked for window state (implied size/maximized).
	// Let's just center if it's the first run (default size).
	if CurrentSettings.WindowWidth == 1920 && CurrentSettings.WindowHeight == 1080 {
		rl.SetWindowPosition(int(monitorWidth/2-float32(CurrentSettings.WindowWidth)/2), int(monitorHeight/2-float32(CurrentSettings.WindowHeight)/2))
	}

	rl.SetTargetFPS(int32(rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())))

	defer func() {
		// Save Settings on exit
		CurrentSettings.WindowWidth = rl.GetScreenWidth()
		CurrentSettings.WindowHeight = rl.GetScreenHeight()
		CurrentSettings.WindowMaximized = rl.IsWindowMaximized()
		if err := SaveSettings(CurrentSettings); err != nil {
			fmt.Printf("Error saving settings: %v\n", err)
		}
	}()

	initImages()

	// Load Plugins
	pluginNodes, err := LoadPlugins("plugins")
	if err != nil {
		fmt.Printf("Error loading plugins: %v\n", err)
	} else {
		nodeTypes = append(nodeTypes, pluginNodes...)
	}

	// Load Initial Flows from CLI
	flowFiles := flag.Args()
	if len(flowFiles) > 0 {
		var mergedGraph *core.Graph

		for _, file := range flowFiles {
			g, err := core.LoadGraph(file)
			if err != nil {
				fmt.Printf("Error loading flow file %s: %v\n", file, err)
				continue
			}
			if mergedGraph == nil {
				mergedGraph = g
				CurrentFilename = file
			} else {
				fmt.Printf("Merging %s...\n", file)
				core.MergeGraph(mergedGraph, g)
			}
		}

		if mergedGraph != nil {
			CurrentGraph = mergedGraph
			selectedNodeID = 0
			// Clear selection map if it exists, or just verify it's init
			if SelectedNodes == nil {
				SelectedNodes = make(map[int]struct{})
			} else {
				clear(SelectedNodes)
			}

			// If duplicate/merged, maybe clear filename to avoid overwriting first file
			if len(flowFiles) > 1 {
				CurrentFilename = ""
			}
		}
	}

	clay.SetMaxElementCount(1 << 16)
	arena := clay.CreateArenaWithCapacity(uintptr(clay.MinMemorySize()))
	clay.Initialize(
		arena,
		clay.Dimensions{Width: float32(CurrentSettings.WindowWidth), Height: float32(CurrentSettings.WindowHeight)}, // Initial size
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

	rl.SetExitKey(0)
	for !rl.WindowShouldClose() && !ShouldQuit {
		frame()
	}
}

func frame() {
	core.Drag.Update()

	// Handle Zoom Input
	wheel := rl.GetMouseWheelMove()
	// Preliminary check. We will confirm this at the end of the frame after UI determines if it consumed the wheel.
	preliminaryShouldZoom := wheel != 0 && !core.IsHoveringUI

	if rl.IsKeyPressed(rl.KeyF9) {
		clay.SetDebugModeEnabled(!clay.IsDebugModeEnabled())
	}

	// Update core.Graph Logic
	topoErr := UpdateGraph()

	// Reset global UI hover state for this frame.
	core.IsHoveringUI = false
	core.IsHoveringPanel = false

	clayPointerMouseDown := rl.IsMouseButtonDown(rl.MouseButtonLeft)
	if core.Drag.Dragging {
		clayPointerMouseDown = false
	}
	core.UIInput.BeginFrame(clayPointerMouseDown)

	// --- Layout 1: Nodes (World Space -> Screen Space Mapped) ---
	// We map the Clay layout to the screen directly, but manually position nodes
	// using WorldToScreen. This allows us to handle infinite canvas interactions
	// correctly (Clay ignores inputs outside its layout bounds) while keeping
	// the UI elements at a constant pixel size (no semantic zoom distortion).
	clay.SetPointerState(
		clay.V2{X: float32(rl.GetMouseX()), Y: float32(rl.GetMouseY())},
		clayPointerMouseDown,
	)
	screenWidth := float32(rl.GetScreenWidth())
	screenHeight := float32(rl.GetScreenHeight())
	clay.SetLayoutDimensions(clay.D{Width: screenWidth, Height: screenHeight})

	scrollDelta := clay.Vector2{X: 0, Y: 0}
	if !preliminaryShouldZoom {
		scrollDelta = clay.Vector2(rl.GetMouseWheelMoveV()).Times(4)
	}
	clay.UpdateScrollContainers(false, scrollDelta, rl.GetFrameTime())

	clay.BeginLayout()
	UINodes(topoErr)
	nodesRenderCommands := clay.EndLayout()

	// Update cached layout info based on the World Space layout
	afterLayout()

	// --- Layout 2: Overlay (Screen Space) ---
	clay.SetPointerState(
		clay.V2{X: float32(rl.GetMouseX()), Y: float32(rl.GetMouseY())},
		clayPointerMouseDown,
	)
	clay.SetLayoutDimensions(clay.D{Width: screenWidth, Height: screenHeight})

	clay.BeginLayout()
	UIOverlay(topoErr)
	overlayRenderCommands := clay.EndLayout()

	processInput()

	core.UIInput.EndFrame()

	// Final check for zoom: Only zoom canvas if UI didn't consume the wheel event
	if preliminaryShouldZoom && !core.UIInput.MouseWheelConsumed {
		wheel := rl.GetMouseWheelMove()
		Camera.ZoomAt(rl.GetMousePosition(), util.Tern(wheel > 0, float32(1.1), float32(0.9)))
	}

	rl.BeginDrawing()
	rl.ClearBackground(core.Night.RGBA())

	// World Space (Mapped to Screen Space)
	renderWorldOverlays()
	renderClayCommands(nodesRenderCommands)

	// Screen Space
	renderClayCommands(overlayRenderCommands)
	renderScreenOverlays()

	rl.EndDrawing()
	clay.ReleaseFrameMemory()

	// Update focus tracking
	if core.UIFocus != nil {
		core.LastUIFocus = *core.UIFocus
		core.LastUIFocusValid = true
	} else {
		core.LastUIFocusValid = false
	}
}

func handleClayErrors(errorData clay.ErrorData) {
	fmt.Printf("CLAY ERROR: %s\n", errorData.ErrorText)
}
