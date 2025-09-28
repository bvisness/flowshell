package app

import rl "github.com/gen2brain/raylib-go/raylib"

var ImgPlay rl.Texture2D
var ImgRetry rl.Texture2D
var ImgDropdownDown rl.Texture2D
var ImgDropdownUp rl.Texture2D

// In a separate function because raylib must be initialized first.
func loadImages() {
	ImgPlay = rl.LoadTexture("assets/play-white.png")
	ImgRetry = rl.LoadTexture("assets/retry-white.png")
	ImgDropdownDown = rl.LoadTexture("assets/dropdown-down.png")
	ImgDropdownUp = rl.LoadTexture("assets/dropdown-up.png")
}
