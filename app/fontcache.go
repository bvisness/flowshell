package app

import rl "github.com/gen2brain/raylib-go/raylib"

type Font = uint16

const (
	InterRegular Font = iota
	InterSemibold
	InterBold
	JetBrainsMono

	FontsEnd
)

var fontFiles = [FontsEnd]string{
	"assets/Inter-Regular.ttf",
	"assets/Inter-SemiBold.ttf",
	"assets/Inter-Bold.ttf",
	"assets/JetBrainsMono-Regular.ttf",
}

const DefaultFontSize = 16

type FontDesc struct {
	Font Font
	Size int
}

var fontCache = make(map[FontDesc]rl.Font)

func LoadFont(font Font, size int) rl.Font {
	desc := FontDesc{font, size}
	if cached, ok := fontCache[FontDesc{font, size}]; ok {
		return cached
	}

	dpi := rl.GetWindowScaleDPI()

	rlfont := LoadAssetFont(fontFiles[font], int32(size*int(dpi.X)))
	fontCache[desc] = rlfont

	return rlfont
}
