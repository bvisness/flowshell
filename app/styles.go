package app

import "github.com/bvisness/flowshell/clay"

var Night = clay.Color{12, 14, 17, 255}
var Charcoal = clay.Color{20, 22, 25, 255}
var DarkGray = clay.Color{33, 36, 40, 255}
var Gray = clay.Color{67, 71, 79, 255}
var White = clay.Color{250, 250, 252, 255}

const S1 = 4
const S2 = 8
const S3 = 16

var PA1 = clay.PaddingAll(S1)
var PA2 = clay.PaddingAll(S2)
var PA3 = clay.PaddingAll(S3)

func PVH(v, h uint16) clay.Padding {
	return clay.Padding{
		Left:   h,
		Right:  h,
		Top:    v,
		Bottom: v,
	}
}

func PD(t, r, b, l uint16, padding clay.Padding) clay.Padding {
	return clay.Padding{
		Left:   padding.Left + l,
		Right:  padding.Right + r,
		Top:    padding.Top + t,
		Bottom: padding.Bottom + b,
	}
}

var BA = clay.BorderWidth{Left: 1, Right: 1, Top: 1, Bottom: 1}

const F1 = 12
const F2 = 16
const F3 = 18

const R1 = 2
const R2 = 4
const R3 = 6

func RA(radius float32) clay.CornerRadius {
	return clay.CornerRadius{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

var RA1 = RA(R1)
var RA2 = RA(R2)
var RA3 = RA(R3)

var GROWH = clay.Sizing{Width: clay.SizingGrow(1, 0)}
var GROWV = clay.Sizing{Height: clay.SizingGrow(1, 0)}
