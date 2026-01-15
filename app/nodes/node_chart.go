package nodes

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type ChartType int

const (
	ChartTypeLine ChartType = iota
	ChartTypeBar
	ChartTypeScatter
)

type ChartRenderData struct {
	Type   ChartType
	Points []rl.Vector2
	MinX   float64
	MaxX   float64
	MinY   float64
	MaxY   float64
	Zoom   float32
	Error  string
}

// Helper to extract data
func ExtractChartData(n *core.Node, xCol, yCol string, chartType ChartType) *ChartRenderData {
	renderData := &ChartRenderData{
		Type: chartType,
		MinX: math.MaxFloat64,
		MaxX: -math.MaxFloat64,
		MinY: math.MaxFloat64,
		MaxY: -math.MaxFloat64,
	}

	res, _ := n.GetResult()
	if res.Err != nil {
		renderData.Error = fmt.Sprintf("Error: %v", res.Err)
		return renderData
	}

	if len(res.Outputs) == 0 {
		return renderData
	}

	val := res.Outputs[0]
	if val.Type.Kind != core.FSKindTable {
		return renderData
	}

	xIdx := -1
	yIdx := -1

	if val.Type.ContainedType != nil {
		for i, f := range val.Type.ContainedType.Fields {
			if xCol != "" && f.Name == xCol {
				xIdx = i
			}
			if yCol != "" && f.Name == yCol {
				yIdx = i
			}
		}
	}

	if yIdx == -1 {
		if val.Type.ContainedType != nil {
			// First pass: look for numeric
			for i, f := range val.Type.ContainedType.Fields {
				if f.Type.Kind == core.FSKindInt64 || f.Type.Kind == core.FSKindFloat64 {
					yIdx = i
					break
				}
			}
			// Fallback to 0 if no numeric column found
			if yIdx == -1 && len(val.Type.ContainedType.Fields) > 0 {
				yIdx = 0
			}
		}
	}

	maxPoints := 1000
	step := 1
	if len(val.TableValue) > maxPoints {
		step = len(val.TableValue) / maxPoints
	}

	for i := 0; i < len(val.TableValue); i += step {
		row := val.TableValue[i]
		var x, y float64
		var err error

		if xIdx != -1 && xIdx < len(row) {
			x, err = toFloat(row[xIdx].Value)
			if err != nil {
				// Fallback: use index as X if value is not numeric (e.g. categorical)
				x = float64(i)
				err = nil
			}
		} else {
			x = float64(i)
		}

		if yIdx != -1 && yIdx < len(row) {
			valY, errY := toFloat(row[yIdx].Value)
			if errY == nil {
				y = valY
			} else {
				continue
			}
		} else {
			continue
		}

		if err == nil {
			renderData.Points = append(renderData.Points, rl.Vector2{X: float32(x), Y: float32(y)})
			if x < renderData.MinX {
				renderData.MinX = x
			}
			if x > renderData.MaxX {
				renderData.MaxX = x
			}
			if y < renderData.MinY {
				renderData.MinY = y
			}
			if y > renderData.MaxY {
				renderData.MaxY = y
			}
		}
	}

	// Pad ranges
	xRange := renderData.MaxX - renderData.MinX
	yRange := renderData.MaxY - renderData.MinY
	if xRange == 0 {
		renderData.MinX -= 0.5
		renderData.MaxX += 0.5
	}
	if yRange == 0 {
		renderData.MinY -= 0.5
		renderData.MaxY += 0.5
	}

	// For Bar charts, Y baseline should often include 0
	if chartType == ChartTypeBar {
		if renderData.MinY > 0 {
			renderData.MinY = 0
		}
		if renderData.MaxY < 0 {
			renderData.MaxY = 0
		}
	}

	return renderData
}

func toFloat(v interface{}) (float64, error) {
	if fv, ok := v.(core.FlowValue); ok {
		switch fv.Type.Kind {
		case core.FSKindInt64:
			return float64(fv.Int64Value), nil
		case core.FSKindFloat64:
			return fv.Float64Value, nil
		case core.FSKindBytes:
			return strconv.ParseFloat(string(fv.BytesValue), 64)
		}
	}
	return 0, fmt.Errorf("not a number")
}

// ResizeDrag identifies a resize operation on a specific node
type ResizeDrag struct {
	NodeID int
}

func (d ResizeDrag) DragKey() string {
	return fmt.Sprintf("Resize-%d", d.NodeID)
}

// --- Line Chart ---

// GEN:NodeAction
type LineChartAction struct {
	XColumn string
	YColumn string
	Width   float32
	Height  float32
	Zoom    float32
}

func NewLineChartNode() *core.Node {
	return &core.Node{
		Name:        "Line Chart",
		InputPorts:  []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}},
		OutputPorts: []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}},
		Action:      &LineChartAction{Width: 400, Height: 200, Zoom: 1.0},
	}
}

var _ core.NodeAction = &LineChartAction{}

func (c *LineChartAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &c.XColumn)
	core.SStr(s, &c.YColumn)

	if s.Encode {
		core.SFloat(s, &c.Width)
		core.SFloat(s, &c.Height)
		core.SFloat(s, &c.Zoom)
	} else {
		// Heuristic to detect version:
		// V0: No extra data (next bytes are next Node's struct)
		// V1: Width, Height (8 bytes).
		// V2: Width, Height, Zoom (12 bytes).
		// Note: Next Node starts with ID (Varint).

		isV1 := false
		isV2 := false

		bytes := s.Buf.Bytes()
		if len(bytes) >= 8 {
			w := math.Float32frombits(binary.LittleEndian.Uint32(bytes[0:4]))
			h := math.Float32frombits(binary.LittleEndian.Uint32(bytes[4:8]))
			// Check if sane dimensions (e.g. 50 to 5000)
			if w >= 50 && w <= 5000 && h >= 50 && h <= 5000 {
				isV1 = true
				if len(bytes) >= 12 {
					z := math.Float32frombits(binary.LittleEndian.Uint32(bytes[8:12]))
					// Check if sane zoom (0.01 to 100)
					if z >= 0.01 && z <= 100.0 {
						isV2 = true
					}
				}
			}
		}

		if isV2 {
			core.SFloat(s, &c.Width)
			core.SFloat(s, &c.Height)
			core.SFloat(s, &c.Zoom)
		} else if isV1 {
			core.SFloat(s, &c.Width)
			core.SFloat(s, &c.Height)
			c.Zoom = 1.0
		} else {
			c.Width = 400
			c.Height = 200
			c.Zoom = 1.0
		}
	}
	return s.Ok()
}

func (c *LineChartAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
	if w, ok := n.GetInputWire(0); ok {
		n.OutputPorts[0].Type = w.Type()
	} else {
		n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindTable}
	}
}

func (c *LineChartAction) UI(n *core.Node) {
	renderData := ExtractChartData(n, c.XColumn, c.YColumn, ChartTypeLine)
	renderData.Zoom = c.Zoom

	// Handle Drag
	dragKey := ResizeDrag{NodeID: n.ID}
	matches, _, _ := core.Drag.State(dragKey)
	if matches && core.Drag.Dragging {
		delta := core.Drag.Offset()
		c.Width = core.Drag.ObjStart.X + float32(delta.X)
		c.Height = core.Drag.ObjStart.Y + float32(delta.Y)
		if c.Width < 100 {
			c.Width = 100
		}
		if c.Height < 100 {
			c.Height = 100
		}
	}

	clay.CLAY(n.ClayID(), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(c.Width)}, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Settings", n.ID), clay.EL{Layout: clay.LAY{Sizing: core.GROWH, ChildGap: core.S2, ChildAlignment: core.YCENTER}}, func() {
			clay.TEXT("X:", clay.TextElementConfig{TextColor: core.LightGray, FontID: core.InterRegular})
			core.UITextBox(clay.IDI("XCol", n.ID), &c.XColumn, core.UITextBoxConfig{El: clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(80)}}}, OnSubmit: func(s string) { c.XColumn = s }})
			clay.TEXT("Y:", clay.TextElementConfig{TextColor: core.LightGray, FontID: core.InterRegular})
			core.UITextBox(clay.IDI("YCol", n.ID), &c.YColumn, core.UITextBoxConfig{El: clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(80)}}}, OnSubmit: func(s string) { c.YColumn = s }})
		})

		clay.CLAY(clay.IDI("ChartWrapper", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			clay.CLAY(clay.IDI("ChartArea", n.ID), clay.EL{
				Layout:          clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, c.Width), Height: clay.SizingFixed(c.Height)}},
				BackgroundColor: core.Black,
				Border:          clay.B{Color: core.Gray, Width: clay.BorderWidth{Left: 1, Right: 1, Top: 1, Bottom: 1}},
				Custom:          clay.CustomElementConfig{CustomData: renderData},
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					wheel := rl.GetMouseWheelMove()
					if wheel != 0 {
						core.UIInput.MouseWheelConsumed = true
						c.Zoom += wheel * 0.1
						if c.Zoom < 0.1 {
							c.Zoom = 0.1
						}
						if c.Zoom > 10.0 {
							c.Zoom = 10.0
						}
					}
				}, nil)
			})

			// Resize Handle
			clay.CLAY(clay.IDI("ResizeHandle", n.ID), clay.EL{
				Layout: clay.LAY{Sizing: core.WH(12, 12)},
				Floating: clay.FloatingElementConfig{
					AttachTo: clay.AttachToParent,
					AttachPoints: clay.FloatingAttachPoints{
						Parent:  clay.AttachPointRightBottom,
						Element: clay.AttachPointRightBottom,
					},
					ZIndex: core.Z_NODE_BUTTON,
				},
				BackgroundColor: core.Gray,
				CornerRadius:    core.RA(2),
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					core.UICursor = rl.MouseCursorResizeNWSE
					if data, ok := clay.GetElementData(elementID); ok {
						core.Drag.TryStartDrag(dragKey, rl.Rectangle(data.BoundingBox), rl.Vector2{X: float32(c.Width), Y: float32(c.Height)})
					}
				}, nil)
			})
		})

		clay.CLAY(clay.IDI("Ports", n.ID), clay.EL{
			Layout: clay.LAY{LayoutDirection: clay.LeftToRight, Sizing: core.GROWH, ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter}, ChildGap: core.S3},
		}, func() {
			core.UIInputPort(n, 0)
			clay.CLAY(clay.IDI("Spacer", n.ID), clay.EL{Layout: clay.LAY{Sizing: core.GROWH}})
			core.UIOutputPort(n, 0)
		})
	})
}

func (c *LineChartAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult)
	go func() {
		defer close(done)
		val, _, err := n.GetInputValue(0)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}
		done <- core.NodeActionResult{Outputs: []core.FlowValue{val}}
	}()
	return done
}

// --- Bar Chart ---

// GEN:NodeAction
type BarChartAction struct {
	XColumn string
	YColumn string
	Width   float32
	Height  float32
	Zoom    float32
}

func NewBarChartNode() *core.Node {
	return &core.Node{
		Name:        "Bar Chart",
		InputPorts:  []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}},
		OutputPorts: []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}},
		Action:      &BarChartAction{Width: 400, Height: 200, Zoom: 1.0},
	}
}

var _ core.NodeAction = &BarChartAction{}

func (c *BarChartAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &c.XColumn)
	core.SStr(s, &c.YColumn)

	if s.Encode {
		core.SFloat(s, &c.Width)
		core.SFloat(s, &c.Height)
		core.SFloat(s, &c.Zoom)
	} else {
		isV1 := false
		isV2 := false

		bytes := s.Buf.Bytes()
		if len(bytes) >= 8 {
			w := math.Float32frombits(binary.LittleEndian.Uint32(bytes[0:4]))
			h := math.Float32frombits(binary.LittleEndian.Uint32(bytes[4:8]))
			// Check if sane dimensions
			if w >= 50 && w <= 5000 && h >= 50 && h <= 5000 {
				isV1 = true
				if len(bytes) >= 12 {
					z := math.Float32frombits(binary.LittleEndian.Uint32(bytes[8:12]))
					if z >= 0.01 && z <= 100.0 {
						isV2 = true
					}
				}
			}
		}

		if isV2 {
			core.SFloat(s, &c.Width)
			core.SFloat(s, &c.Height)
			core.SFloat(s, &c.Zoom)
		} else if isV1 {
			core.SFloat(s, &c.Width)
			core.SFloat(s, &c.Height)
			c.Zoom = 1.0
		} else {
			c.Width = 400
			c.Height = 200
			c.Zoom = 1.0
		}
	}
	return s.Ok()
}

func (c *BarChartAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
	if w, ok := n.GetInputWire(0); ok {
		n.OutputPorts[0].Type = w.Type()
	} else {
		n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindTable}
	}
}

func (c *BarChartAction) UI(n *core.Node) {
	renderData := ExtractChartData(n, c.XColumn, c.YColumn, ChartTypeBar)
	renderData.Zoom = c.Zoom

	// Handle Drag
	dragKey := ResizeDrag{NodeID: n.ID}
	matches, _, _ := core.Drag.State(dragKey)
	if matches && core.Drag.Dragging {
		delta := core.Drag.Offset()
		c.Width = core.Drag.ObjStart.X + float32(delta.X)
		c.Height = core.Drag.ObjStart.Y + float32(delta.Y)
		if c.Width < 100 {
			c.Width = 100
		}
		if c.Height < 100 {
			c.Height = 100
		}
	}

	clay.CLAY(n.ClayID(), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(c.Width)}, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Settings", n.ID), clay.EL{Layout: clay.LAY{Sizing: core.GROWH, ChildGap: core.S2, ChildAlignment: core.YCENTER}}, func() {
			clay.TEXT("X:", clay.TextElementConfig{TextColor: core.LightGray, FontID: core.InterRegular})
			core.UITextBox(clay.IDI("XCol", n.ID), &c.XColumn, core.UITextBoxConfig{El: clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(80)}}}, OnSubmit: func(s string) { c.XColumn = s }})
			clay.TEXT("Y:", clay.TextElementConfig{TextColor: core.LightGray, FontID: core.InterRegular})
			core.UITextBox(clay.IDI("YCol", n.ID), &c.YColumn, core.UITextBoxConfig{El: clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(80)}}}, OnSubmit: func(s string) { c.YColumn = s }})
		})

		clay.CLAY(clay.IDI("ChartWrapper", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			clay.CLAY(clay.IDI("ChartArea", n.ID), clay.EL{
				Layout:          clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, c.Width), Height: clay.SizingFixed(c.Height)}},
				BackgroundColor: core.Black,
				Border:          clay.B{Color: core.Gray, Width: clay.BorderWidth{Left: 1, Right: 1, Top: 1, Bottom: 1}},
				Custom:          clay.CustomElementConfig{CustomData: renderData},
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					wheel := rl.GetMouseWheelMove()
					if wheel != 0 {
						core.UIInput.MouseWheelConsumed = true
						c.Zoom += wheel * 0.1
						if c.Zoom < 0.1 {
							c.Zoom = 0.1
						}
						if c.Zoom > 10.0 {
							c.Zoom = 10.0
						}
					}
				}, nil)
			})

			// Resize Handle
			clay.CLAY(clay.IDI("ResizeHandle", n.ID), clay.EL{
				Layout: clay.LAY{Sizing: core.WH(12, 12)},
				Floating: clay.FloatingElementConfig{
					AttachTo: clay.AttachToParent,
					AttachPoints: clay.FloatingAttachPoints{
						Parent:  clay.AttachPointRightBottom,
						Element: clay.AttachPointRightBottom,
					},
					ZIndex: core.Z_NODE_BUTTON,
				},
				BackgroundColor: core.Gray,
				CornerRadius:    core.RA(2),
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					core.UICursor = rl.MouseCursorResizeNWSE
					if data, ok := clay.GetElementData(elementID); ok {
						core.Drag.TryStartDrag(dragKey, rl.Rectangle(data.BoundingBox), rl.Vector2{X: float32(c.Width), Y: float32(c.Height)})
					}
				}, nil)
			})
		})

		clay.CLAY(clay.IDI("Ports", n.ID), clay.EL{
			Layout: clay.LAY{LayoutDirection: clay.LeftToRight, Sizing: core.GROWH, ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter}, ChildGap: core.S3},
		}, func() {
			core.UIInputPort(n, 0)
			clay.CLAY(clay.IDI("Spacer", n.ID), clay.EL{Layout: clay.LAY{Sizing: core.GROWH}})
			core.UIOutputPort(n, 0)
		})
	})
}

func (c *BarChartAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult)
	go func() {
		defer close(done)
		val, _, err := n.GetInputValue(0)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}
		done <- core.NodeActionResult{Outputs: []core.FlowValue{val}}
	}()
	return done
}

// --- Scatter Plot ---

// GEN:NodeAction
type ScatterPlotAction struct {
	XColumn string
	YColumn string
	Width   float32
	Height  float32
	Zoom    float32
}

func NewScatterPlotNode() *core.Node {
	return &core.Node{
		Name:        "Scatter Plot",
		InputPorts:  []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}},
		OutputPorts: []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}},
		Action:      &ScatterPlotAction{Width: 400, Height: 200, Zoom: 1.0},
	}
}

var _ core.NodeAction = &ScatterPlotAction{}

func (c *ScatterPlotAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &c.XColumn)
	core.SStr(s, &c.YColumn)

	if s.Encode {
		core.SFloat(s, &c.Width)
		core.SFloat(s, &c.Height)
		core.SFloat(s, &c.Zoom)
	} else {
		isV1 := false
		isV2 := false

		bytes := s.Buf.Bytes()
		if len(bytes) >= 8 {
			w := math.Float32frombits(binary.LittleEndian.Uint32(bytes[0:4]))
			h := math.Float32frombits(binary.LittleEndian.Uint32(bytes[4:8]))
			// Check if sane dimensions
			if w >= 50 && w <= 5000 && h >= 50 && h <= 5000 {
				isV1 = true
				if len(bytes) >= 12 {
					z := math.Float32frombits(binary.LittleEndian.Uint32(bytes[8:12]))
					if z >= 0.01 && z <= 100.0 {
						isV2 = true
					}
				}
			}
		}

		if isV2 {
			core.SFloat(s, &c.Width)
			core.SFloat(s, &c.Height)
			core.SFloat(s, &c.Zoom)
		} else if isV1 {
			core.SFloat(s, &c.Width)
			core.SFloat(s, &c.Height)
			c.Zoom = 1.0
		} else {
			c.Width = 400
			c.Height = 200
			c.Zoom = 1.0
		}
	}
	return s.Ok()
}

func (c *ScatterPlotAction) UpdateAndValidate(n *core.Node) {
	n.Valid = true
	if w, ok := n.GetInputWire(0); ok {
		n.OutputPorts[0].Type = w.Type()
	} else {
		n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindTable}
	}
}

func (c *ScatterPlotAction) UI(n *core.Node) {
	renderData := ExtractChartData(n, c.XColumn, c.YColumn, ChartTypeScatter)
	renderData.Zoom = c.Zoom

	// Handle Drag
	dragKey := ResizeDrag{NodeID: n.ID}
	matches, _, _ := core.Drag.State(dragKey)
	if matches && core.Drag.Dragging {
		delta := core.Drag.Offset()
		c.Width = core.Drag.ObjStart.X + float32(delta.X)
		c.Height = core.Drag.ObjStart.Y + float32(delta.Y)
		if c.Width < 100 {
			c.Width = 100
		}
		if c.Height < 100 {
			c.Height = 100
		}
	}

	clay.CLAY(n.ClayID(), clay.EL{
		Layout: clay.LAY{LayoutDirection: clay.TopToBottom, Sizing: clay.Sizing{Width: clay.SizingFixed(c.Width)}, ChildGap: core.S2},
	}, func() {
		clay.CLAY(clay.IDI("Settings", n.ID), clay.EL{Layout: clay.LAY{Sizing: core.GROWH, ChildGap: core.S2, ChildAlignment: core.YCENTER}}, func() {
			clay.TEXT("X:", clay.TextElementConfig{TextColor: core.LightGray, FontID: core.InterRegular})
			core.UITextBox(clay.IDI("XCol", n.ID), &c.XColumn, core.UITextBoxConfig{El: clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(80)}}}, OnSubmit: func(s string) { c.XColumn = s }})
			clay.TEXT("Y:", clay.TextElementConfig{TextColor: core.LightGray, FontID: core.InterRegular})
			core.UITextBox(clay.IDI("YCol", n.ID), &c.YColumn, core.UITextBoxConfig{El: clay.EL{Layout: clay.LAY{Sizing: clay.Sizing{Width: clay.SizingFixed(80)}}}, OnSubmit: func(s string) { c.YColumn = s }})
		})
		clay.CLAY(clay.IDI("ChartWrapper", n.ID), clay.EL{
			Layout: clay.LAY{Sizing: core.GROWH},
		}, func() {
			clay.CLAY(clay.IDI("ChartArea", n.ID), clay.EL{
				Layout:          clay.LAY{Sizing: clay.Sizing{Width: clay.SizingGrow(0, c.Width), Height: clay.SizingFixed(c.Height)}},
				BackgroundColor: core.Black,
				Border:          clay.B{Color: core.Gray, Width: clay.BorderWidth{Left: 1, Right: 1, Top: 1, Bottom: 1}},
				Custom:          clay.CustomElementConfig{CustomData: renderData},
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					wheel := rl.GetMouseWheelMove()
					if wheel != 0 {
						core.UIInput.MouseWheelConsumed = true
						c.Zoom += wheel * 0.1
						if c.Zoom < 0.1 {
							c.Zoom = 0.1
						}
						if c.Zoom > 10.0 {
							c.Zoom = 10.0
						}
					}
				}, nil)
			})

			// Resize Handle
			clay.CLAY(clay.IDI("ResizeHandle", n.ID), clay.EL{
				Layout: clay.LAY{Sizing: core.WH(12, 12)},
				Floating: clay.FloatingElementConfig{
					AttachTo: clay.AttachToParent,
					AttachPoints: clay.FloatingAttachPoints{
						Parent:  clay.AttachPointRightBottom,
						Element: clay.AttachPointRightBottom,
					},
					ZIndex: core.Z_NODE_BUTTON,
				},
				BackgroundColor: core.Gray,
				CornerRadius:    core.RA(2),
			}, func() {
				clay.OnHover(func(elementID clay.ElementID, pointerData clay.PointerData, _ any) {
					core.UICursor = rl.MouseCursorResizeNWSE
					if data, ok := clay.GetElementData(elementID); ok {
						core.Drag.TryStartDrag(dragKey, rl.Rectangle(data.BoundingBox), rl.Vector2{X: float32(c.Width), Y: float32(c.Height)})
					}
				}, nil)
			})
		})

		// Move ports inside (at bottom or top as appropriate, usually simple stacking)
		clay.CLAY(clay.IDI("Ports", n.ID), clay.EL{
			Layout: clay.LAY{LayoutDirection: clay.LeftToRight, Sizing: core.GROWH, ChildAlignment: clay.ChildAlignment{Y: clay.AlignYCenter}, ChildGap: core.S3},
		}, func() {
			core.UIInputPort(n, 0)
			clay.CLAY(clay.IDI("Spacer", n.ID), clay.EL{Layout: clay.LAY{Sizing: core.GROWH}}) // Push output to right
			core.UIOutputPort(n, 0)
		})
	})
}

func (c *ScatterPlotAction) Run(n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult)
	go func() {
		defer close(done)
		val, _, err := n.GetInputValue(0)
		if err != nil {
			done <- core.NodeActionResult{Err: err}
			return
		}
		done <- core.NodeActionResult{Outputs: []core.FlowValue{val}}
	}()
	return done
}

// GEN:CustomRender
func RenderChart(bbox clay.BoundingBox, data *ChartRenderData) {
	width := bbox.Width
	height := bbox.Height

	if data.Error != "" {
		rl.DrawText(data.Error, int32(bbox.X)+10, int32(bbox.Y)+10, 20, rl.Red)
		return
	}

	if len(data.Points) < 1 {
		rl.DrawText("No Data", int32(bbox.X)+10, int32(bbox.Y)+10, 20, rl.Gray)
		return
	}

	xRange := data.MaxX - data.MinX
	yRange := data.MaxY - data.MinY

	midX := data.MinX + xRange/2
	midY := data.MinY + yRange/2

	zoom := float64(data.Zoom)
	if zoom <= 0 {
		zoom = 1.0
	}

	effXRange := xRange / zoom
	effYRange := yRange / zoom

	effMinX := midX - effXRange/2
	effMinY := midY - effYRange/2

	mapPt := func(pt rl.Vector2) rl.Vector2 {
		x := (float64(pt.X) - effMinX) / effXRange
		y := (float64(pt.Y) - effMinY) / effYRange
		return rl.Vector2{
			X: bbox.X + float32(x)*width,
			Y: bbox.Y + height - float32(y)*height,
		}
	}

	// Clip to bounding box
	rl.BeginScissorMode(int32(bbox.X), int32(bbox.Y), int32(width), int32(height))
	defer rl.EndScissorMode()

	// Draw Grid/Axes
	rl.DrawRectangleLinesEx(rl.Rectangle{X: bbox.X, Y: bbox.Y, Width: width, Height: height}, 1, rl.DarkGray)

	// Draw Zero Line Y if visible
	if data.MinY < 0 && data.MaxY > 0 {
		zeroY := bbox.Y + height - float32((0-data.MinY)/yRange)*height
		rl.DrawLineEx(rl.Vector2{X: bbox.X, Y: zeroY}, rl.Vector2{X: bbox.X + width, Y: zeroY}, 1, rl.DarkGray)
	}

	switch data.Type {
	case ChartTypeLine:
		if len(data.Points) == 1 {
			p := mapPt(data.Points[0])
			rl.DrawCircleV(p, 3, rl.Green)
		} else {
			for i := 0; i < len(data.Points)-1; i++ {
				p1 := mapPt(data.Points[i])
				p2 := mapPt(data.Points[i+1])
				rl.DrawLineEx(p1, p2, 2, rl.Green)
			}
		}
	case ChartTypeBar:
		barWidth := width / float32(len(data.Points)) * 0.8
		for _, pt := range data.Points {
			p := mapPt(pt)
			base := mapPt(rl.Vector2{X: pt.X, Y: 0})
			// Draw bar from base to p
			// We need to handle rect

			// Actually mapPt X is center? No, X is value.
			// Bar charts usually treat X as category.
			// Here we just plot X/Y.

			// Let's draw a vertical line or thin rect
			h := base.Y - p.Y
			// If h is negative (value < 0), p.Y > base.Y

			r := rl.Rectangle{
				X:      p.X - barWidth/2,
				Y:      util.Tern(h > 0, p.Y, base.Y),
				Width:  barWidth,
				Height: float32(math.Abs(float64(h))),
			}
			rl.DrawRectangleRec(r, rl.SkyBlue)
		}
	case ChartTypeScatter:
		for _, pt := range data.Points {
			p := mapPt(pt)
			rl.DrawCircleV(p, 3, rl.Yellow)
		}
	}

	rl.DrawText(fmt.Sprintf("%.2f", data.MaxY), int32(bbox.X)+5, int32(bbox.Y)+5, 10, rl.Gray)
	rl.DrawText(fmt.Sprintf("%.2f", data.MinY), int32(bbox.X)+5, int32(bbox.Y+height)-15, 10, rl.Gray)
}
