package app

import (
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func renderClayCommands(commands []clay.RenderCommand) {
	for _, cmd := range commands {
		bbox := cmd.BoundingBox
		switch cmd.CommandType {
		case clay.RenderCommandTypeRectangle:
			config := cmd.RenderData.Rectangle
			if config.CornerRadius.TopLeft > 0 {
				// Why we have to do this for Raylib I do not understand.
				radius := config.CornerRadius.TopLeft * 2 / util.Tern(bbox.Width > bbox.Height, bbox.Height, bbox.Width)
				rl.DrawRectangleRounded(rl.Rectangle(bbox), radius, 8, config.BackgroundColor.RGBA())
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

		case clay.RenderCommandTypeImage:
			img := cmd.RenderData.Image
			tex := img.ImageData.(rl.Texture2D)
			tintColor := img.BackgroundColor
			if tintColor.R == 0 && tintColor.G == 0 && tintColor.B == 0 && tintColor.A == 0 {
				tintColor = clay.Color{255, 255, 255, 255}
			}
			rl.DrawTexturePro(
				tex,
				rl.Rectangle{0, 0, float32(tex.Width), float32(tex.Height)},
				rl.Rectangle(cmd.BoundingBox),
				rl.Vector2{},
				0,
				tintColor.RGBA(),
			)

		case clay.RenderCommandTypeScissorStart:
			rl.BeginScissorMode(int32(bbox.X), int32(bbox.Y), int32(bbox.Width), int32(bbox.Height))
		case clay.RenderCommandTypeScissorEnd:
			rl.EndScissorMode()

			// TODO: CUSTOM
		}
	}
}
