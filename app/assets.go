package app

import (
	"embed"
	"path/filepath"

	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed assets
var assets embed.FS

var ImgPlay rl.Texture2D
var ImgRetry rl.Texture2D
var ImgPushpin rl.Texture2D
var ImgPushpinOutline rl.Texture2D
var ImgDropdownDown rl.Texture2D
var ImgDropdownUp rl.Texture2D

// In a separate function because raylib must be initialized first.
func initImages() {
	ImgPlay = LoadAssetTexture("assets/play-white.png")
	ImgRetry = LoadAssetTexture("assets/retry-white.png")
	ImgPushpin = LoadAssetTexture("assets/pushpin-white.png")
	ImgPushpinOutline = LoadAssetTexture("assets/pushpin-outline-white.png")
	ImgDropdownDown = LoadAssetTexture("assets/dropdown-down.png")
	ImgDropdownUp = LoadAssetTexture("assets/dropdown-up.png")
}

func LoadAssetTexture(path string) rl.Texture2D {
	data := util.Must1(assets.ReadFile(path))
	img := rl.LoadImageFromMemory(filepath.Ext(path), data, int32(len(data)))
	return rl.LoadTextureFromImage(img)
}

func LoadAssetFont(path string, fontSize int32) rl.Font {
	data := util.Must1(assets.ReadFile(path))
	return rl.LoadFontFromMemory(filepath.Ext(path), data, fontSize, nil)
}
