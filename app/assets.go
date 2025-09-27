package app

import rl "github.com/gen2brain/raylib-go/raylib"

var ImgPlay rl.Texture2D

func loadImages() {
	ImgPlay = rl.LoadTexture("assets/play-white.png")
}
