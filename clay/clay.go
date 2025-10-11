package clay

/*
#include <stdlib.h>
#include "clay.h"

typedef void (*Clay_ErrorHandlerFunc)(Clay_ErrorData errorText);
void clayErrorCallback_cgo(Clay_ErrorData errorText);

typedef Clay_Dimensions (*Clay_MeasureTextFunc)(Clay_StringSlice text, Clay_TextElementConfig *config, void *userData);
Clay_Dimensions clayMeasureTextCallback_cgo(Clay_StringSlice text, Clay_TextElementConfig *config, void *userData);

typedef void (*Clay_OnHoverCallback)(Clay_ElementId elementId, Clay_PointerData pointerData, intptr_t userData);
void clayOnHoverCallback_cgo(Clay_ElementId elementId, Clay_PointerData pointerData, intptr_t userData);

Clay_SizingAxis Clay_SetSizingAxisMinMax(Clay_SizingAxis in, Clay_SizingMinMax minMax);
Clay_SizingAxis Clay_SetSizingAxisPercent(Clay_SizingAxis in, float percent);
void Clay_SetContextErrorHandler(Clay_Context* ctx, Clay_ErrorHandler eh);
Clay_RectangleRenderData Clay_GetRenderDataRectangle(Clay_RenderData d);
Clay_BorderRenderData Clay_GetRenderDataBorder(Clay_RenderData d);
Clay_TextRenderData Clay_GetRenderDataText(Clay_RenderData d);
Clay_ImageRenderData Clay_GetRenderDataImage(Clay_RenderData d);
Clay_ClipRenderData Clay_GetRenderDataClip(Clay_RenderData d);
Clay_CustomRenderData Clay_GetRenderDataCustom(Clay_RenderData d);
*/
import "C"

import (
	"image/color"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/bvisness/flowshell/util"
)

var pinner runtime.Pinner

// ----------------------
// Macros

func PaddingAll(padding uint16) Padding {
	return Padding{padding, padding, padding, padding}
}

func SizingFit(min, max float32) SizingAxis {
	return SizingAxis{MinMax: SizingMinMax{min, max}, Type: SizingTypeFit}
}

func SizingGrow(min, max float32) SizingAxis {
	return SizingAxis{MinMax: SizingMinMax{min, max}, Type: SizingTypeGrow}
}

func SizingFixed(fixed float32) SizingAxis {
	return SizingAxis{MinMax: SizingMinMax{fixed, fixed}, Type: SizingTypeFixed}
}

func SizingPercent(percentOfParent float32) SizingAxis {
	return SizingAxis{Percent: percentOfParent, Type: SizingTypePercent}
}

// ----------------------
// Data types

// Note: Clay_String is not guaranteed to be null terminated. It may be if
// created from a literal C string, but it is also used to represent slices.
type String struct {
	// Set this boolean to true if the char* data underlying this string will live
	// for the entire lifetime of the program. This will automatically be set for
	// strings created with CLAY_STRING, as the macro requires a string literal.
	IsStaticallyAllocated bool
	Length                int32
	// The underlying character memory. Note: this will not be copied and will not
	// extend the lifetime of the underlying memory.
	Chars unsafe.Pointer
}

func (r String) C() C.Clay_String {
	return C.Clay_String{
		isStaticallyAllocated: C.bool(r.IsStaticallyAllocated),

		length: C.int32_t(r.Length),
		chars:  (*C.char)(r.Chars),
	}
}

// Clay_Arena is a memory arena structure that is used by clay to manage its internal allocations.
// Rather than creating it by hand, it's easier to use Clay_CreateArenaWithCapacityAndMemory()
type Arena struct {
	inner C.Clay_Arena
}

func (r Arena) C() C.Clay_Arena {
	return r.inner
}

type Dimensions struct {
	Width, Height float32
}

type D = Dimensions

func (r Dimensions) C() C.Clay_Dimensions {
	return C.Clay_Dimensions{
		width:  C.float(r.Width),
		height: C.float(r.Height),
	}
}

type Vector2 struct {
	X, Y float32
}

type V2 = Vector2

func (r Vector2) Plus(b Vector2) Vector2 {
	return Vector2{
		X: r.X + b.X,
		Y: r.Y + b.Y,
	}
}

func (r Vector2) Minus(b Vector2) Vector2 {
	return Vector2{
		X: r.X - b.X,
		Y: r.Y - b.Y,
	}
}

func (r Vector2) Times(b float32) Vector2 {
	return Vector2{
		X: r.X * b,
		Y: r.Y * b,
	}
}

func (r Vector2) C() C.Clay_Vector2 {
	return C.Clay_Vector2{
		x: C.float(r.X),
		y: C.float(r.Y),
	}
}

// awful name
func Vector22Go(r C.Clay_Vector2) Vector2 {
	return Vector2{
		X: float32(r.x),
		Y: float32(r.y),
	}
}

// Internally clay conventionally represents colors as 0-255, but interpretation
// is up to the renderer.
type Color struct {
	R, G, B, A float32
}

func (c Color) RGBA() color.RGBA {
	return color.RGBA{
		R: uint8(c.R),
		G: uint8(c.G),
		B: uint8(c.B),
		A: uint8(c.A),
	}
}

func (r Color) C() C.Clay_Color {
	return C.Clay_Color{
		r: C.float(r.R),
		g: C.float(r.G),
		b: C.float(r.B),
		a: C.float(r.A),
	}
}

func Color2Go(r C.Clay_Color) Color {
	return Color{
		R: float32(r.r),
		G: float32(r.g),
		B: float32(r.b),
		A: float32(r.a),
	}
}

type BoundingBox struct {
	X, Y, Width, Height float32
}

func (r BoundingBox) XY() Vector2 {
	return Vector2{r.X, r.Y}
}

func BoundingBox2Go(r C.Clay_BoundingBox) BoundingBox {
	return BoundingBox{
		X:      float32(r.x),
		Y:      float32(r.y),
		Width:  float32(r.width),
		Height: float32(r.height),
	}
}

// Primarily created via the CLAY_ID(), CLAY_IDI(), CLAY_ID_LOCAL() and
// CLAY_IDI_LOCAL() macros. Represents a hashed string ID used for identifying
// and finding specific clay UI elements, required by functions such as
// Clay_PointerOver() and Clay_GetElementData().
type ElementID struct {
	// The resulting hash generated from the other fields.
	ID uint32
	// A numerical offset applied after computing the hash from
	// stringId.
	Offset uint32
	// A base hash value to start from, for example the parent
	// element ID is used when calculating CLAY_ID_LOCAL().
	BaseID uint32
	// The string id to hash.
	StringID string
}

func (r ElementID) C() C.Clay_ElementId {
	return C.Clay_ElementId{
		id:     C.uint32_t(r.ID),
		offset: C.uint32_t(r.Offset),
		baseId: C.uint32_t(r.BaseID),
		// stringId: String2ClayString(r.StringID), // TODO? Can we just do all string hashes on the Go side?
	}
}

func ElementID2Go(r C.Clay_ElementId) ElementID {
	return ElementID{
		ID:       uint32(r.id),
		Offset:   uint32(r.offset),
		BaseID:   uint32(r.baseId),
		StringID: C.GoStringN(r.stringId.chars, r.stringId.length),
	}
}

// Controls the "radius", or corner rounding of elements, including rectangles,
// borders and images. The rounding is determined by drawing a circle inset into
// the element corner by (radius, radius) pixels.
type CornerRadius struct {
	TopLeft     float32
	TopRight    float32
	BottomLeft  float32
	BottomRight float32
}

func (r CornerRadius) C() C.Clay_CornerRadius {
	return C.Clay_CornerRadius{
		topLeft:     C.float(r.TopLeft),
		topRight:    C.float(r.TopRight),
		bottomLeft:  C.float(r.BottomLeft),
		bottomRight: C.float(r.BottomRight),
	}
}

func CornerRadius2Go(r C.Clay_CornerRadius) CornerRadius {
	return CornerRadius{
		TopLeft:     float32(r.topLeft),
		TopRight:    float32(r.topRight),
		BottomLeft:  float32(r.bottomLeft),
		BottomRight: float32(r.bottomRight),
	}
}

// Controls the direction in which child elements will be automatically laid out.
type LayoutDirection uint8

const (
	LeftToRight LayoutDirection = iota // (Default) Lays out child elements from left to right with increasing x.
	TopToBottom                        // Lays out child elements from top to bottom with increasing y.
)

func (r LayoutDirection) C() C.Clay_LayoutDirection {
	return C.Clay_LayoutDirection(r)
}

// Controls the alignment along the x axis (horizontal) of child elements.
type LayoutAlignmentX uint8

const (
	AlignXLeft   LayoutAlignmentX = iota // (Default) Aligns child elements to the left hand side of this element, offset by padding.width.left
	AlignXRight                          // Aligns child elements to the right hand side of this element, offset by padding.width.right
	AlignXCenter                         // Aligns child elements horizontally to the center of this element
)

func (r LayoutAlignmentX) C() C.Clay_LayoutAlignmentX {
	return C.Clay_LayoutAlignmentX(r)
}

// Controls the alignment along the y axis (vertical) of child elements.
type LayoutAlignmentY uint8

const (
	AlignYTop    LayoutAlignmentY = iota // (Default) Aligns child elements to the top of this element, offset by padding.width.top
	AlignYBottom                         // Aligns child elements to the bottom of this element, offset by padding.width.bottom
	AlignYCenter                         // Aligns child elements vertically to the center of this element
)

func (r LayoutAlignmentY) C() C.Clay_LayoutAlignmentY {
	return C.Clay_LayoutAlignmentY(r)
}

// Controls how the element takes up space inside its parent container.
type SizingType uint8

const (
	SizingTypeFit     SizingType = iota // (default) Wraps tightly to the size of the element's contents.
	SizingTypeGrow                      // Expands along this axis to fill available space in the parent element, sharing it with other GROW elements.
	SizingTypePercent                   // Expects 0-1 range. Clamps the axis size to a percent of the parent container's axis size minus padding and child gaps.
	SizingTypeFixed                     // Clamps the axis size to an exact size in pixels.
)

func (r SizingType) C() C.Clay__SizingType {
	return C.Clay__SizingType(r)
}

// Controls how child elements are aligned on each axis.
type ChildAlignment struct {
	X LayoutAlignmentX // Controls alignment of children along the x axis.
	Y LayoutAlignmentY // Controls alignment of children along the y axis.
}
type CA = ChildAlignment

func (r ChildAlignment) C() C.Clay_ChildAlignment {
	return C.Clay_ChildAlignment{
		x: r.X.C(),
		y: r.Y.C(),
	}
}

// Controls the minimum and maximum size in pixels that this element is allowed to grow or shrink to,
// overriding sizing types such as FIT or GROW.
type SizingMinMax struct {
	Min float32 // The smallest final size of the element on this axis will be this value in pixels.
	Max float32 // The largest final size of the element on this axis will be this value in pixels.
}

func (r SizingMinMax) C() C.Clay_SizingMinMax {
	return C.Clay_SizingMinMax{
		min: C.float(r.Min),
		max: C.float(r.Max),
	}
}

// Controls the sizing of this element along one axis inside its parent container.
type SizingAxis struct {
	MinMax  SizingMinMax // Controls the minimum and maximum size in pixels that this element is allowed to grow or shrink to, overriding sizing types such as FIT or GROW.
	Percent float32      // Expects 0-1 range. Clamps the axis size to a percent of the parent container's axis size minus padding and child gaps.
	Type    SizingType   // Controls how the element takes up space inside its parent container.
}

func (r SizingAxis) C() C.Clay_SizingAxis {
	base := C.Clay_SizingAxis{
		_type: r.Type.C(),
	}
	if r.Type == SizingTypePercent {
		return C.Clay_SetSizingAxisPercent(base, C.float(r.Percent))
	} else {
		return C.Clay_SetSizingAxisMinMax(base, r.MinMax.C())
	}
}

// Controls the sizing of this element along one axis inside its parent container.
type Sizing struct {
	Width  SizingAxis // Controls the width sizing of the element, along the x axis.
	Height SizingAxis // Controls the height sizing of the element, along the y axis.
}

func (r Sizing) C() C.Clay_Sizing {
	return C.Clay_Sizing{
		width:  r.Width.C(),
		height: r.Height.C(),
	}
}

// Controls "padding" in pixels, which is a gap between the bounding box of this element and where its children
// will be placed.
type Padding struct {
	Left, Right, Top, Bottom uint16
}

func (r Padding) C() C.Clay_Padding {
	return C.Clay_Padding{
		left:   C.uint16_t(r.Left),
		right:  C.uint16_t(r.Right),
		top:    C.uint16_t(r.Top),
		bottom: C.uint16_t(r.Bottom),
	}
}

// Controls various settings that affect the size and position of an element, as well as the sizes and positions
// of any child elements.
type LayoutConfig struct {
	Sizing          Sizing          // Controls the sizing of this element inside it's parent container, including FIT, GROW, PERCENT and FIXED sizing.
	Padding         Padding         // Controls "padding" in pixels, which is a gap between the bounding box of this element and where its children will be placed.
	ChildGap        uint16          // Controls the gap in pixels between child elements along the layout axis (horizontal gap for LEFT_TO_RIGHT, vertical gap for TOP_TO_BOTTOM).
	ChildAlignment  ChildAlignment  // Controls how child elements are aligned on each axis.
	LayoutDirection LayoutDirection // Controls the direction in which child elements will be automatically laid out.
}

// Controls various settings that affect the size and position of an element, as well as the sizes and positions
// of any child elements.
type LAY = LayoutConfig

func (r LayoutConfig) C() C.Clay_LayoutConfig {
	return C.Clay_LayoutConfig{
		sizing:          r.Sizing.C(),
		padding:         r.Padding.C(),
		childGap:        C.uint16_t(r.ChildGap),
		childAlignment:  r.ChildAlignment.C(),
		layoutDirection: r.LayoutDirection.C(),
	}
}

// Controls how text "wraps", that is how it is broken into multiple lines when
// there is insufficient horizontal space.
type TextElementConfigWrapMode uint8

const (
	// (default) breaks on whitespace characters.
	TextWrapWords TextElementConfigWrapMode = iota
	// Don't break on space characters, only on newlines.
	TextWrapNewlines
	// Disable text wrapping entirely.
	TextWrapNone
)

// Controls how wrapped lines of text are horizontally aligned within the outer
// text bounding box.
type TextAlignment uint8

const (
	// (default) Horizontally aligns wrapped lines of text to the left hand side
	// of their bounding box.
	TextAlignLeft TextAlignment = iota
	// Horizontally aligns wrapped lines of text to the center of their bounding
	// box.
	TextAlignCenter
	// Horizontally aligns wrapped lines of text to the right hand side of their
	// bounding box.
	TextAlignRight
)

// Controls various functionality related to text elements.
type TextElementConfig struct {
	// A pointer that will be transparently passed through to the resulting render
	// command.
	UserData any
	// The RGBA color of the font to render, conventionally specified as 0-255.
	TextColor Color
	// An integer transparently passed to Clay_MeasureText to identify the font to
	// use. The debug view will pass fontId = 0 for its internal text.
	FontID uint16
	// Controls the size of the font. Handled by the function provided to
	// Clay_MeasureText.
	FontSize uint16
	// Controls extra horizontal spacing between characters. Handled by the
	// function provided to Clay_MeasureText.
	LetterSpacing uint16
	// Controls additional vertical space between wrapped lines of text.
	LineHeight uint16
	// Controls how text "wraps", that is how it is broken into multiple lines
	// when there is insufficient horizontal space. CLAY_TEXT_WRAP_WORDS (default)
	// breaks on whitespace characters. CLAY_TEXT_WRAP_NEWLINES doesn't break on
	// space characters, only on newlines. CLAY_TEXT_WRAP_NONE disables wrapping
	// entirely.
	WrapMode TextElementConfigWrapMode
	// Controls how wrapped lines of text are horizontally aligned within the
	// outer text bounding box. CLAY_TEXT_ALIGN_LEFT (default) - Horizontally
	// aligns wrapped lines of text to the left hand side of their bounding box.
	// CLAY_TEXT_ALIGN_CENTER - Horizontally aligns wrapped lines of text to the
	// center of their bounding box. CLAY_TEXT_ALIGN_RIGHT - Horizontally aligns
	// wrapped lines of text to the right hand side of their bounding box.
	TextAlignment TextAlignment
}
type T = TextElementConfig

func (r TextElementConfig) C() C.Clay_TextElementConfig {
	return C.Clay_TextElementConfig{
		// userData: unsafe.Pointer(&r.UserData), // TODO: UserData registry
		textColor:     r.TextColor.C(),
		fontId:        C.uint16_t(r.FontID),
		fontSize:      C.uint16_t(r.FontSize),
		letterSpacing: C.uint16_t(r.LetterSpacing),
		lineHeight:    C.uint16_t(r.LineHeight),
		wrapMode:      C.Clay_TextElementConfigWrapMode(r.WrapMode),
		textAlignment: C.Clay_TextAlignment(r.TextAlignment),
	}
}

func TextElementConfig2Go(r C.Clay_TextElementConfig) TextElementConfig {
	return TextElementConfig{
		// UserData: , // TODO: UserData registry
		TextColor:     Color2Go(r.textColor),
		FontID:        uint16(r.fontId),
		FontSize:      uint16(r.fontSize),
		LetterSpacing: uint16(r.letterSpacing),
		LineHeight:    uint16(r.lineHeight),
		WrapMode:      TextElementConfigWrapMode(r.wrapMode),
		TextAlignment: TextAlignment(r.textAlignment),
	}
}

// Controls various settings related to aspect ratio scaling element.
type AspectRatioElementConfig struct {
	// A float representing the target "Aspect ratio" for an
	// element, which is its final width divided by its final
	// height.
	AspectRatio float32
}
type ASPECT = AspectRatioElementConfig

func (r AspectRatioElementConfig) C() C.Clay_AspectRatioElementConfig {
	return C.Clay_AspectRatioElementConfig{
		aspectRatio: C.float(r.AspectRatio),
	}
}

// Controls various settings related to image elements.
type ImageElementConfig struct {
	// A transparent pointer used to pass image data through to
	// the renderer.
	ImageData any
}
type IMG = ImageElementConfig

func (r ImageElementConfig) C() C.Clay_ImageElementConfig {
	var imageHandlePtr unsafe.Pointer
	if r.ImageData != nil {
		imageHandle := cgo.NewHandle(r.ImageData)
		imageHandlePtr = unsafe.Pointer(&imageHandle)
	}
	// pinner.Pin(imageHandlePtr)

	return C.Clay_ImageElementConfig{
		imageData: imageHandlePtr,
	}
}

// Controls where a floating element is offset relative to its parent element.
// Note: see
// https://github.com/user-attachments/assets/b8c6dfaa-c1b1-41a4-be55-013473e4a6ce
// for a visual explanation.
type FloatingAttachPointType uint8

const (
	AttachPointLeftTop FloatingAttachPointType = iota
	AttachPointLeftCenter
	AttachPointLeftBottom
	AttachPointCenterTop
	AttachPointCenterCenter
	AttachPointCenterBottom
	AttachPointRightTop
	AttachPointRightCenter
	AttachPointRightBottom
)

// Controls where a floating element is offset relative to its parent element.
type FloatingAttachPoints struct {
	// Controls the origin point on a floating element that attaches
	// to its parent.
	Element FloatingAttachPointType
	// Controls the origin point on the parent element that the
	// floating element attaches to.
	Parent FloatingAttachPointType
}

func (r FloatingAttachPoints) C() C.Clay_FloatingAttachPoints {
	return C.Clay_FloatingAttachPoints{
		element: C.Clay_FloatingAttachPointType(r.Element),
		parent:  C.Clay_FloatingAttachPointType(r.Parent),
	}
}

// Controls how mouse pointer events like hover and click are captured or passed
// through to elements underneath a floating element.
type PointerCaptureMode uint8

const (
	// (default) "Capture" the pointer event and don't allow events like hover
	// and click to pass through to elements underneath.
	PointercaptureModeCapture PointerCaptureMode = iota

	// Transparently pass through pointer events like hover and click to
	// elements underneath the floating element.
	PointercaptureModePassthrough
)

// Controls which element a floating element is "attached" to (i.e. relative
// offset from).
type FloatingAttachToElement uint8

const (
	// (default) Disables floating for this element.
	AttachToNone FloatingAttachToElement = iota
	// Attaches this floating element to its parent, positioned based on the
	// .attachPoints and .offset fields.
	AttachToParent
	// Attaches this floating element to an element with a specific ID,
	// specified with the .parentId field. positioned based on the .attachPoints
	// and .offset fields.
	AttachToElementWithID
	// Attaches this floating element to the root of the layout, which combined
	// with the .offset field provides functionality similar to "absolute
	// positioning".
	AttachToRoot
)

// Controls whether or not a floating element is clipped to the same clipping
// rectangle as the element it's attached to.
type FloatingClipToElement uint8

const (
	// (default) - The floating element does not inherit clipping.
	ClipToNone FloatingClipToElement = iota
	// The floating element is clipped to the same clipping rectangle as the
	// element it's attached to.
	ClipToAttachedParent
)

// Controls various settings related to "floating" elements, which are elements
// that "float" above other elements, potentially overlapping their boundaries,
// and not affecting the layout of sibling or parent elements.
type FloatingElementConfig struct {
	// Offsets this floating element by the provided x,y coordinates from its
	// attachPoints.
	Offset Vector2
	// Expands the boundaries of the outer floating element without affecting its
	// children.
	Expand Dimensions
	// When used in conjunction with .attachTo = CLAY_ATTACH_TO_ELEMENT_WITH_ID,
	// attaches this floating element to the element in the hierarchy with the
	// provided ID. Hint: attach the ID to the other element with .id =
	// CLAY_ID("yourId"), and specify the id the same way, with .parentId =
	// CLAY_ID("yourId").id
	ParentID uint32
	// Controls the z index of this floating element and all its children.
	// Floating elements are sorted in ascending z order before output. zIndex is
	// also passed to the renderer for all elements contained within this floating
	// element.
	ZIndex int16
	// Controls how mouse pointer events like hover and click are captured or
	// passed through to elements underneath / behind a floating element. Enum is
	// of the form CLAY_ATTACH_POINT_foo_bar. See Clay_FloatingAttachPoints for
	// more details. Note: see <img
	// src="https://github.com/user-attachments/assets/b8c6dfaa-c1b1-41a4-be55-013473e4a6ce
	// /> and <img
	// src="https://github.com/user-attachments/assets/ebe75e0d-1904-46b0-982d-418f929d1516
	// /> for a visual explanation.
	AttachPoints FloatingAttachPoints
	// Controls how mouse pointer events like hover and click are captured or
	// passed through to elements underneath a floating element.
	// CLAY_POINTER_CAPTURE_MODE_CAPTURE (default) - "Capture" the pointer event
	// and don't allow events like hover and click to pass through to elements
	// underneath. CLAY_POINTER_CAPTURE_MODE_PASSTHROUGH - Transparently pass
	// through pointer events like hover and click to elements underneath the
	// floating element.
	PointerCaptureMode PointerCaptureMode
	// Controls which element a floating element is "attached" to (i.e. relative
	// offset from). CLAY_ATTACH_TO_NONE (default) - Disables floating for this
	// element. CLAY_ATTACH_TO_PARENT - Attaches this floating element to its
	// parent, positioned based on the .attachPoints and .offset fields.
	// CLAY_ATTACH_TO_ELEMENT_WITH_ID - Attaches this floating element to an
	// element with a specific ID, specified with the .parentId field. positioned
	// based on the .attachPoints and .offset fields. CLAY_ATTACH_TO_ROOT -
	// Attaches this floating element to the root of the layout, which combined
	// with the .offset field provides functionality similar to "absolute
	// positioning".
	AttachTo FloatingAttachToElement
	// Controls whether or not a floating element is clipped to the same clipping
	// rectangle as the element it's attached to. CLAY_CLIP_TO_NONE (default) -
	// The floating element does not inherit clipping.
	// CLAY_CLIP_TO_ATTACHED_PARENT - The floating element is clipped to the same
	// clipping rectangle as the element it's attached to.
	ClipTo FloatingClipToElement
}
type FLOAT = FloatingElementConfig

func (r FloatingElementConfig) C() C.Clay_FloatingElementConfig {
	return C.Clay_FloatingElementConfig{
		offset:             r.Offset.C(),
		expand:             r.Expand.C(),
		parentId:           C.uint32_t(r.ParentID),
		zIndex:             C.int16_t(r.ZIndex),
		attachPoints:       r.AttachPoints.C(),
		pointerCaptureMode: C.Clay_PointerCaptureMode(r.PointerCaptureMode),
		attachTo:           C.Clay_FloatingAttachToElement(r.AttachTo),
		clipTo:             C.Clay_FloatingClipToElement(r.ClipTo),
	}
}

// Controls various settings related to custom elements.
type CustomElementConfig struct {
	// A transparent pointer through which you can pass custom data to the
	// renderer. Generates CUSTOM render commands.
	CustomData any
}

func (r CustomElementConfig) C() C.Clay_CustomElementConfig {
	return C.Clay_CustomElementConfig{
		// TODO: Another registry!!!!
	}
}

// Controls the axis on which an element switches to "scrolling", which clips
// the contents and allows scrolling in that direction.
type ClipElementConfig struct {
	Horizontal bool // Clip overflowing elements on the X axis.
	Vertical   bool // Clip overflowing elements on the Y axis.

	// Offsets the x,y positions of all child elements.
	// Used primarily for scrolling containers.
	ChildOffset Vector2
}
type CLIP = ClipElementConfig

func (r ClipElementConfig) C() C.Clay_ClipElementConfig {
	return C.Clay_ClipElementConfig{
		horizontal:  C.bool(r.Horizontal),
		vertical:    C.bool(r.Vertical),
		childOffset: r.ChildOffset.C(),
	}
}

// Controls the widths of individual element borders.
type BorderWidth struct {
	Left, Right, Top, Bottom uint16

	// Creates borders between each child element, depending on the
	// .layoutDirection. e.g. for LEFT_TO_RIGHT, borders will be vertical lines,
	// and for TOP_TO_BOTTOM borders will be horizontal lines. .betweenChildren
	// borders will result in individual RECTANGLE render commands being
	// generated.
	BetweenChildren uint16
}
type BW = BorderWidth

func (r BorderWidth) C() C.Clay_BorderWidth {
	return C.Clay_BorderWidth{
		left:            C.uint16_t(r.Left),
		right:           C.uint16_t(r.Right),
		top:             C.uint16_t(r.Top),
		bottom:          C.uint16_t(r.Bottom),
		betweenChildren: C.uint16_t(r.BetweenChildren),
	}
}

func BorderWidth2Go(r C.Clay_BorderWidth) BorderWidth {
	return BorderWidth{
		Left:            uint16(r.left),
		Right:           uint16(r.right),
		Top:             uint16(r.top),
		Bottom:          uint16(r.bottom),
		BetweenChildren: uint16(r.betweenChildren),
	}
}

// Controls settings related to element borders.
type BorderElementConfig struct {
	// Controls the color of all borders with width > 0. Conventionally
	// represented as 0-255, but interpretation is up to the renderer.
	Color Color
	// Controls the widths of individual borders. At least one of these
	// should be > 0 for a BORDER render command to be generated.
	Width BorderWidth
}
type B = BorderElementConfig

func (r BorderElementConfig) C() C.Clay_BorderElementConfig {
	return C.Clay_BorderElementConfig{
		color: r.Color.C(),
		width: r.Width.C(),
	}
}

// Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_TEXT
type TextRenderData struct {
	// A string slice containing the text to be rendered.
	// Note: this is not guaranteed to be null terminated.
	StringContents string
	// Conventionally represented as 0-255 for each channel, but interpretation is
	// up to the renderer.
	TextColor Color
	// An integer representing the font to use to render this text, transparently
	// passed through from the text declaration.
	FontID   uint16
	FontSize uint16
	// Specifies the extra whitespace gap in pixels between each character.
	LetterSpacing uint16
	// The height of the bounding box for this line of text.
	LineHeight uint16
}

func TextRenderData2Go(r C.Clay_TextRenderData) TextRenderData {
	return TextRenderData{
		StringContents: C.GoStringN(r.stringContents.chars, r.stringContents.length),
		TextColor:      Color2Go(r.textColor),
		FontID:         uint16(r.fontId),
		FontSize:       uint16(r.fontSize),
		LetterSpacing:  uint16(r.letterSpacing),
		LineHeight:     uint16(r.lineHeight),
	}
}

// Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_RECTANGLE
type RectangleRenderData struct {
	// The solid background color to fill this rectangle with. Conventionally
	// represented as 0-255 for each channel, but interpretation is up to the
	// renderer.
	BackgroundColor Color
	// Controls the "radius", or corner rounding of elements, including
	// rectangles, borders and images. The rounding is determined by drawing a
	// circle inset into the element corner by (radius, radius) pixels.
	CornerRadius CornerRadius
}

func RectangleRenderData2Go(r C.Clay_RectangleRenderData) RectangleRenderData {
	return RectangleRenderData{
		BackgroundColor: Color2Go(r.backgroundColor),
		CornerRadius:    CornerRadius2Go(r.cornerRadius),
	}
}

// Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_IMAGE
type ImageRenderData struct {
	// The tint color for this image. Note that the default value is 0,0,0,0 and
	// should likely be interpreted as "untinted". Conventionally represented as
	// 0-255 for each channel, but interpretation is up to the renderer.
	BackgroundColor Color
	// Controls the "radius", or corner rounding of this image.
	// The rounding is determined by drawing a circle inset into the element
	// corner by (radius, radius) pixels.
	CornerRadius CornerRadius
	// A pointer transparently passed through from the original element
	// definition, typically used to represent image data.
	ImageData any
}

func ImageRenderData2Go(r C.Clay_ImageRenderData) ImageRenderData {
	return ImageRenderData{
		BackgroundColor: Color2Go(r.backgroundColor),
		CornerRadius:    CornerRadius2Go(r.cornerRadius),
		ImageData:       util.Tern(r.imageData != nil, (*cgo.Handle)(r.imageData).Value(), nil),
	}
}

// Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_CUSTOM
type CustomRenderData struct {
	// Passed through from .backgroundColor in the original element declaration.
	// Conventionally represented as 0-255 for each channel, but interpretation is
	// up to the renderer.
	BackgroundColor Color
	// Controls the "radius", or corner rounding of this custom element.
	// The rounding is determined by drawing a circle inset into the element
	// corner by (radius, radius) pixels.
	CornerRadius CornerRadius
	// A pointer transparently passed through from the original element
	// definition.
	CustomData any
}

// Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_SCISSOR_START || commandType == CLAY_RENDER_COMMAND_TYPE_SCISSOR_END
type ClipRenderData struct {
	Horizontal bool
	Vertical   bool
}

func ClipRenderData2Go(r C.Clay_ClipRenderData) ClipRenderData {
	return ClipRenderData{
		Horizontal: bool(r.horizontal),
		Vertical:   bool(r.vertical),
	}
}

// Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_BORDER
type BorderRenderData struct {
	// Controls a shared color for all this element's borders.
	// Conventionally represented as 0-255 for each channel, but interpretation is up to the renderer.
	Color Color
	// Specifies the "radius", or corner rounding of this border element.
	// The rounding is determined by drawing a circle inset into the element corner by (radius, radius) pixels.
	CornerRadius CornerRadius
	// Controls individual border side widths.
	Width BorderWidth
}

func BorderRenderData2Go(r C.Clay_BorderRenderData) BorderRenderData {
	return BorderRenderData{
		Color:        Color2Go(r.color),
		CornerRadius: CornerRadius2Go(r.cornerRadius),
		Width:        BorderWidth2Go(r.width),
	}
}

// A struct union containing data specific to this command's .commandType
type RenderData struct {
	Rectangle RectangleRenderData // Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_RECTANGLE
	Text      TextRenderData      // Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_TEXT
	Image     ImageRenderData     // Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_IMAGE
	Custom    CustomRenderData    // Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_CUSTOM
	Border    BorderRenderData    // Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_BORDER
	Clip      ClipRenderData      // Render command data when commandType == CLAY_RENDER_COMMAND_TYPE_SCISSOR_START|END
}

func RenderData2Go(r C.Clay_RenderData, ty RenderCommandType) RenderData {
	switch ty {
	case RenderCommandTypeRectangle:
		return RenderData{Rectangle: RectangleRenderData2Go(C.Clay_GetRenderDataRectangle(r))}
	case RenderCommandTypeBorder:
		return RenderData{Border: BorderRenderData2Go(C.Clay_GetRenderDataBorder(r))}
	case RenderCommandTypeText:
		return RenderData{Text: TextRenderData2Go(C.Clay_GetRenderDataText(r))}
	case RenderCommandTypeImage:
		return RenderData{Image: ImageRenderData2Go(C.Clay_GetRenderDataImage(r))}
	case RenderCommandTypeScissorStart, RenderCommandTypeScissorEnd:
		return RenderData{Clip: ClipRenderData2Go(C.Clay_GetRenderDataClip(r))}
	default:
		return RenderData{}
	}
}

type ElementData struct {
	BoundingBox BoundingBox
	Found       bool
}

func ElementData2Go(r C.Clay_ElementData) ElementData {
	return ElementData{
		BoundingBox: BoundingBox2Go(r.boundingBox),
		Found:       bool(r.found),
	}
}

// Used by renderers to determine specific handling for each render command.
type RenderCommandType uint8

const (
	// This command type should be skipped.
	RenderCommandTypeNone RenderCommandType = iota
	// The renderer should draw a solid color rectangle.
	RenderCommandTypeRectangle
	// The renderer should draw a colored border inset into the bounding box.
	RenderCommandTypeBorder
	// The renderer should draw text.
	RenderCommandTypeText
	// The renderer should draw an image.
	RenderCommandTypeImage
	// The renderer should begin clipping all future draw commands, only rendering content that falls within the provided boundingBox.
	RenderCommandTypeScissorStart
	// The renderer should finish any previously active clipping, and begin rendering elements in full again.
	RenderCommandTypeScissorEnd
	// The renderer should provide a custom implementation for handling this render command based on its .customData
	RenderCommandTypeCustom
)

type RenderCommand struct {
	BoundingBox BoundingBox // A rectangular box that fully encloses this UI element, with the position relative to the root of the layout.
	RenderData  RenderData  // A struct union containing data specific to this command's commandType.
	UserData    any         // A pointer transparently passed through from the original element declaration.
	ID          uint32      // The id of this element, transparently passed through from the original element declaration.

	// The z order required for drawing this command correctly.
	// Note: the render command array is already sorted in ascending order, and will produce correct results if drawn in naive order.
	// This field is intended for use in batching renderers for improved performance.
	ZIndex int16

	// Specifies how to handle rendering of this command.
	// CLAY_RENDER_COMMAND_TYPE_RECTANGLE - The renderer should draw a solid color rectangle.
	// CLAY_RENDER_COMMAND_TYPE_BORDER - The renderer should draw a colored border inset into the bounding box.
	// CLAY_RENDER_COMMAND_TYPE_TEXT - The renderer should draw text.
	// CLAY_RENDER_COMMAND_TYPE_IMAGE - The renderer should draw an image.
	// CLAY_RENDER_COMMAND_TYPE_SCISSOR_START - The renderer should begin clipping all future draw commands, only rendering content that falls within the provided boundingBox.
	// CLAY_RENDER_COMMAND_TYPE_SCISSOR_END - The renderer should finish any previously active clipping, and begin rendering elements in full again.
	// CLAY_RENDER_COMMAND_TYPE_CUSTOM - The renderer should provide a custom implementation for handling this render command based on its .customData
	CommandType RenderCommandType
}

func RenderCommand2Go(r C.Clay_RenderCommand) RenderCommand {
	return RenderCommand{
		BoundingBox: BoundingBox2Go(r.boundingBox),
		RenderData:  RenderData2Go(r.renderData, RenderCommandType(r.commandType)),
		UserData:    r.userData, // TODO: Another Go-side registry?
		ID:          uint32(r.id),
		ZIndex:      int16(r.zIndex),
		CommandType: RenderCommandType(r.commandType),
	}
}

// Represents the current state of interaction with clay this frame.
type PointerDataInteractionState uint8

const (
	// A left mouse click, or touch occurred this frame.
	PointerDataPressedThisFrame PointerDataInteractionState = iota
	// The left mouse button click or touch happened at some point in the past,
	// and is still currently held down this frame.
	PointerDataPressed
	// The left mouse button click or touch was released this frame.
	PointerDataReleasedThisFrame
	// The left mouse button click or touch is not currently down / was released
	// at some point in the past.
	PointerDataReleased
)

// Information on the current state of pointer interactions this frame.
type PointerData struct {
	// The position of the mouse / touch / pointer relative to the root of the
	// layout.
	Position Vector2
	// Represents the current state of interaction with clay this frame.
	// CLAY_POINTER_DATA_PRESSED_THIS_FRAME - A left mouse click, or touch
	// occurred this frame. CLAY_POINTER_DATA_PRESSED - The left mouse button
	// click or touch happened at some point in the past, and is still currently
	// held down this frame. CLAY_POINTER_DATA_RELEASED_THIS_FRAME - The left
	// mouse button click or touch was released this frame.
	// CLAY_POINTER_DATA_RELEASED - The left mouse button click or touch is not
	// currently down / was released at some point in the past.
	State PointerDataInteractionState
}

func PointerData2Go(r C.Clay_PointerData) PointerData {
	return PointerData{
		Position: Vector22Go(r.position),
		State:    PointerDataInteractionState(r.state),
	}
}

type EL = ElementDeclaration
type ElementDeclaration struct {
	// Controls various settings that affect the size and position of an element,
	// as well as the sizes and positions of any child elements.
	Layout LayoutConfig
	// Controls the background color of the resulting element.
	// By convention specified as 0-255, but interpretation is up to the renderer.
	// If no other config is specified, .backgroundColor will generate a RECTANGLE
	// render command, otherwise it will be passed as a property to IMAGE or
	// CUSTOM render commands.
	BackgroundColor Color
	// Controls the "radius", or corner rounding of elements, including
	// rectangles, borders and images.
	CornerRadius CornerRadius
	// Controls settings related to aspect ratio scaling.
	AspectRatio AspectRatioElementConfig
	// Controls settings related to image elements.
	Image ImageElementConfig
	// Controls whether and how an element "floats", which means it layers over
	// the top of other elements in z order, and doesn't affect the position and
	// size of siblings or parent elements. Note: in order to activate floating,
	// .floating.attachTo must be set to something other than the default value.
	Floating FloatingElementConfig
	// Used to create CUSTOM render commands, usually to render element types not
	// supported by Clay.
	Custom CustomElementConfig
	// Controls whether an element should clip its contents, as well as providing
	// child x,y offset configuration for scrolling.
	Clip ClipElementConfig
	// Controls settings related to element borders, and will generate BORDER
	// render commands.
	Border BorderElementConfig
	// A pointer that will be transparently passed through to resulting render
	// commands.
	UserData any
}

func (r ElementDeclaration) C() C.Clay_ElementDeclaration {
	return C.Clay_ElementDeclaration{
		layout:          r.Layout.C(),
		backgroundColor: r.BackgroundColor.C(),
		cornerRadius:    r.CornerRadius.C(),
		aspectRatio:     r.AspectRatio.C(),
		image:           r.Image.C(),
		floating:        r.Floating.C(),
		custom:          r.Custom.C(),
		clip:            r.Clip.C(),
		border:          r.Border.C(),
		// TODO: userdata? another registry?
	}
}

type ErrorType uint8

const (
	// A text measurement function wasn't provided using Clay_SetMeasureTextFunction(), or the provided function was null.
	CLAY_ERROR_TYPE_TEXT_MEASUREMENT_FUNCTION_NOT_PROVIDED ErrorType = iota

	// Clay attempted to allocate its internal data structures but ran out of space.
	// The arena passed to Clay_Initialize was created with a capacity smaller than that required by Clay_MinMemorySize().
	CLAY_ERROR_TYPE_ARENA_CAPACITY_EXCEEDED

	// Clay ran out of capacity in its internal array for storing elements. This limit can be increased with Clay_SetMaxElementCount().
	CLAY_ERROR_TYPE_ELEMENTS_CAPACITY_EXCEEDED

	// Clay ran out of capacity in its internal array for storing elements. This limit can be increased with Clay_SetMaxMeasureTextCacheWordCount().
	CLAY_ERROR_TYPE_TEXT_MEASUREMENT_CAPACITY_EXCEEDED

	// Two elements were declared with exactly the same ID within one layout.
	CLAY_ERROR_TYPE_DUPLICATE_ID

	// A floating element was declared using CLAY_ATTACH_TO_ELEMENT_ID and either an invalid .parentId was provided or no element with the provided .parentId was found.
	CLAY_ERROR_TYPE_FLOATING_CONTAINER_PARENT_NOT_FOUND

	// An element was declared that using CLAY_SIZING_PERCENT but the percentage value was over 1. Percentage values are expected to be in the 0-1 range.
	CLAY_ERROR_TYPE_PERCENTAGE_OVER_1

	// Clay encountered an internal error. It would be wonderful if you could report this so we can fix it!
	CLAY_ERROR_TYPE_INTERNAL_ERROR

	// Clay__OpenElement was called more times than Clay__CloseElement, so there were still remaining open elements when the layout ended.
	CLAY_ERROR_TYPE_UNBALANCED_OPEN_CLOSE
)

// Data to identify the error that clay has encountered.
type ErrorData struct {
	// Represents the type of error clay encountered while computing layout.
	// CLAY_ERROR_TYPE_TEXT_MEASUREMENT_FUNCTION_NOT_PROVIDED - A text measurement function wasn't provided using Clay_SetMeasureTextFunction(), or the provided function was null.
	// CLAY_ERROR_TYPE_ARENA_CAPACITY_EXCEEDED - Clay attempted to allocate its internal data structures but ran out of space. The arena passed to Clay_Initialize was created with a capacity smaller than that required by Clay_MinMemorySize().
	// CLAY_ERROR_TYPE_ELEMENTS_CAPACITY_EXCEEDED - Clay ran out of capacity in its internal array for storing elements. This limit can be increased with Clay_SetMaxElementCount().
	// CLAY_ERROR_TYPE_TEXT_MEASUREMENT_CAPACITY_EXCEEDED - Clay ran out of capacity in its internal array for storing elements. This limit can be increased with Clay_SetMaxMeasureTextCacheWordCount().
	// CLAY_ERROR_TYPE_DUPLICATE_ID - Two elements were declared with exactly the same ID within one layout.
	// CLAY_ERROR_TYPE_FLOATING_CONTAINER_PARENT_NOT_FOUND - A floating element was declared using CLAY_ATTACH_TO_ELEMENT_ID and either an invalid .parentId was provided or no element with the provided .parentId was found.
	// CLAY_ERROR_TYPE_PERCENTAGE_OVER_1 - An element was declared that using CLAY_SIZING_PERCENT but the percentage value was over 1. Percentage values are expected to be in the 0-1 range.
	// CLAY_ERROR_TYPE_INTERNAL_ERROR - Clay encountered an internal error. It would be wonderful if you could report this so we can fix it!
	ErrorType ErrorType

	// A string containing human-readable error text that explains the error in more detail.
	ErrorText string

	// A transparent pointer passed through from when the error handler was first provided.
	UserData any
}

func ErrorData2Go(r C.Clay_ErrorData) ErrorData {
	return ErrorData{
		ErrorType: ErrorType(r.errorType),
		ErrorText: C.GoStringN(r.errorText.chars, r.errorText.length),
		UserData:  r.userData,
	}
}

// A wrapper struct around Clay's error handler function.
type ErrorHandler struct {
	// A user provided function to call when Clay encounters an error during layout.
	ErrorHandlerFunction func(errorText ErrorData)

	// A pointer that will be transparently passed through to the error handler when it is called.
	UserData any
}

func (r ErrorHandler) C() C.Clay_ErrorHandler {
	return C.Clay_ErrorHandler{
		errorHandlerFunction: C.Clay_ErrorHandlerFunc(C.clayErrorCallback_cgo),
		// no userdata; we keep userdata on the go side
	}
}

func MinMemorySize() uint32 {
	return uint32(C.Clay_MinMemorySize())
}

func CreateArenaWithCapacity(capacity uintptr) Arena {
	return Arena{
		inner: C.Clay_CreateArenaWithCapacityAndMemory(C.size_t(capacity), C.malloc(C.size_t(capacity))),
	}
}

func Initialize(arena Arena, layoutDimensions Dimensions, errorHandler ErrorHandler) {
	// Start with empty error handler
	C.Clay_Initialize(arena.C(), layoutDimensions.C(), C.Clay_ErrorHandler{})

	// After initialization, if an error handler was provided, set it up
	if errorHandler.ErrorHandlerFunction != nil {
		ctx := C.Clay_GetCurrentContext()
		C.Clay_SetContextErrorHandler(ctx, errorHandler.C())
		errorHandlers[ctx] = errorHandler
	}
}

// Sets the state of the "pointer" (i.e. the mouse or touch) in Clay's internal data. Used for detecting and responding to mouse events in the debug view,
// as well as for Clay_Hovered() and scroll element handling.
func SetPointerState(position Vector2, pointerDown bool) {
	C.Clay_SetPointerState(position.C(), C.bool(pointerDown))
}

// Returns the internally stored scroll offset for the currently open element.
// Generally intended for use with clip elements to create scrolling containers.
func GetScrollOffset() Vector2 {
	return Vector22Go(C.Clay_GetScrollOffset())
}

// Updates the state of Clay's internal scroll data, updating scroll content positions if scrollDelta is non zero, and progressing momentum scrolling.
// - enableDragScrolling when set to true will enable mobile device like "touch drag" scroll of scroll containers, including momentum scrolling after the touch has ended.
// - scrollDelta is the amount to scroll this frame on each axis in pixels.
// - deltaTime is the time in seconds since the last "frame" (scroll update)
func UpdateScrollContainers(enableDragScrolling bool, scrollDelta Vector2, deltaTime float32) {
	C.Clay_UpdateScrollContainers(C.bool(enableDragScrolling), scrollDelta.C(), C.float(deltaTime))
}

// Updates the layout dimensions in response to the window or outer container being resized.
func SetLayoutDimensions(dimensions Dimensions) {
	C.Clay_SetLayoutDimensions(dimensions.C())
}

// Called before starting any layout declarations.
func BeginLayout() {
	C.Clay_BeginLayout()
}

// Called when all layout declarations are finished.
// Computes the layout and generates and returns the array of render commands to draw.
func EndLayout() []RenderCommand {
	cmds := C.Clay_EndLayout()
	res := make([]RenderCommand, cmds.length)
	for i := int32(0); i < int32(cmds.length); i++ {
		res[i] = RenderCommand2Go(*C.Clay_RenderCommandArray_Get(&cmds, C.int32_t(i)))
	}
	return res
}

func GetElementData(id ElementID) (ElementData, bool) {
	res := ElementData2Go(C.Clay_GetElementData(id.C()))
	return res, res.Found
}

func Hovered() bool {
	return bool(C.Clay_Hovered())
}

type OnHoverFunc func(elementID ElementID, pointerData PointerData, userData any)
type registeredOnHover struct {
	Func     OnHoverFunc
	UserData any
}

func OnHover(f OnHoverFunc, userData any) {
	registeredFuncHandle := cgo.NewHandle(registeredOnHover{
		Func:     f,
		UserData: userData,
	})
	C.Clay_OnHover(C.Clay_OnHoverCallback(C.clayOnHoverCallback_cgo), C.intptr_t(registeredFuncHandle))
}

//export clayOnHoverCallback
func clayOnHoverCallback(elementId C.Clay_ElementId, pointerData C.Clay_PointerData, userData C.intptr_t) {
	registeredFunc := cgo.Handle(userData).Value().(registeredOnHover)
	registeredFunc.Func(ElementID2Go(elementId), PointerData2Go(pointerData), registeredFunc.UserData)
}

type MeasureTextFunction func(str string, config *TextElementConfig, userData any) Dimensions

func SetMeasureTextFunction(measureTextFunction MeasureTextFunction, userData any) {
	ctx := C.Clay_GetCurrentContext()
	measureTextFuncs[ctx] = measureTextFunction
	measureTextUserData[ctx] = userData
	C.Clay_SetMeasureTextFunction(C.Clay_MeasureTextFunc(C.clayMeasureTextCallback_cgo), nil)
}

func SetDebugModeEnabled(enabled bool) {
	C.Clay_SetDebugModeEnabled(C.bool(enabled))
}

func IsDebugModeEnabled() bool {
	return bool(C.Clay_IsDebugModeEnabled())
}

func SetMaxElementCount(maxElementCount int32) {
	C.Clay_SetMaxElementCount(C.int32_t(maxElementCount))
}

// ----------------------------
// Go-specific functions

var errorHandlers = make(map[*C.Clay_Context]ErrorHandler)
var measureTextUserData = make(map[*C.Clay_Context]any)
var measureTextFuncs = make(map[*C.Clay_Context]MeasureTextFunction)

//export clayErrorCallback
func clayErrorCallback(errorText C.Clay_ErrorData) {
	ctx := C.Clay_GetCurrentContext()
	handler := errorHandlers[ctx]
	data := ErrorData2Go(errorText)
	data.UserData = handler.UserData // this is what Clay does internally; we mimic it here
	handler.ErrorHandlerFunction(data)
}

//export clayMeasureTextCallback
func clayMeasureTextCallback(text C.Clay_StringSlice, config *C.Clay_TextElementConfig, _ unsafe.Pointer) C.Clay_Dimensions {
	ctx := C.Clay_GetCurrentContext()
	f := measureTextFuncs[ctx]
	userData := measureTextUserData[ctx]
	cfg := TextElementConfig2Go(*config)
	dimensions := f(C.GoStringN(text.chars, text.length), &cfg, userData)
	return dimensions.C()
}

// A magic sentinel ID recognized by these bindings as being intended to
// generate a new automatic ID. Since this seems to only be possible through
// macros like CLAY_AUTO_ID, which open a new element, we have back-doored it.
var AUTO_ID = ElementID{
	ID:     0xFFFFFFFF,
	Offset: 0xFFFFFFFF,
	BaseID: 0xFFFFFFFF,
}

func CLAY(id ElementID, decl ElementDeclaration, children ...func()) {
	if id == AUTO_ID {
		CLAY_AUTO_ID(decl, children...)
		return
	}

	C.Clay__OpenElementWithId(id.C())
	C.Clay__ConfigureOpenElement(decl.C())
	for _, f := range children {
		f()
	}
	C.Clay__CloseElement()
}

func CLAY_LATE(id ElementID, decl func() ElementDeclaration, children ...func()) {
	if id == AUTO_ID {
		CLAY_AUTO_ID_LATE(decl, children...)
		return
	}

	C.Clay__OpenElementWithId(id.C())
	C.Clay__ConfigureOpenElement(decl().C())
	for _, f := range children {
		f()
	}
	C.Clay__CloseElement()
}

func CLAY_AUTO_ID(decl ElementDeclaration, children ...func()) {
	C.Clay__OpenElement()
	C.Clay__ConfigureOpenElement(decl.C())
	for _, f := range children {
		f()
	}
	C.Clay__CloseElement()
}

func CLAY_AUTO_ID_LATE(decl func() ElementDeclaration, children ...func()) {
	C.Clay__OpenElement()
	C.Clay__ConfigureOpenElement(decl().C())
	for _, f := range children {
		f()
	}
	C.Clay__CloseElement()
}

func TEXT(text string, textConfig TextElementConfig) {
	clayStr := CacheString(text)
	C.Clay__OpenTextElement(clayStr.C(), C.Clay__StoreTextElementConfig(textConfig.C()))
}

func ID(label string) ElementID {
	return hashString(label, 0)
}

func IDI(label string, offset int) ElementID {
	return hashStringWithOffset(label, uint32(offset), 0)
}

func CacheString(str string) String {
	ptr := C.CBytes([]byte(str))
	return String{
		IsStaticallyAllocated: false,

		Length: int32(len(str)),
		Chars:  ptr,
	}
}

func ClearCachedStrings() {
	// TODO: uh, who knows?
}

func hashString(key string, seed uint32) ElementID {
	hash := seed

	for i := 0; i < len(key); i++ {
		hash += uint32(key[i])
		hash += (hash << 10)
		hash ^= (hash >> 6)
	}

	hash += (hash << 3)
	hash ^= (hash >> 11)
	hash += (hash << 15)
	return ElementID{
		ID:       hash + 1,
		Offset:   0,
		BaseID:   hash + 1,
		StringID: key, // Reserve the hash result of zero as "null id"
	}
}

func hashStringWithOffset(key string, offset uint32, seed uint32) ElementID {
	hash := uint32(0)
	base := seed

	for i := 0; i < len(key); i++ {
		base += uint32(key[i])
		base += (base << 10)
		base ^= (base >> 6)
	}
	hash = base
	hash += offset
	hash += (hash << 10)
	hash ^= (hash >> 6)

	hash += (hash << 3)
	base += (base << 3)
	hash ^= (hash >> 11)
	base ^= (base >> 11)
	hash += (hash << 15)
	base += (base << 15)
	return ElementID{
		ID:       hash + 1,
		Offset:   offset,
		BaseID:   base + 1,
		StringID: key, // Reserve the hash result of zero as "null id"
	}
}
