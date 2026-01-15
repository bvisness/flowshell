package nodes

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bvisness/flowshell/app/core"
	"github.com/bvisness/flowshell/clay"
)

// GEN:NodeAction
type LoadFileAction struct {
	Path string

	Format core.UIDropdown

	// InferTypes controls whether CSV columns are automatically converted to
	// Int64/Float64 based on their content. If false, all columns are loaded
	// as strings (Bytes). Users can then use the "Convert Type" node to
	// manually convert specific columns if needed.
	InferTypes bool
}

const maxLoadFileBytes int64 = 256 << 20

func readAllWithLimit(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	r := io.Reader(f)
	if limit > 0 {
		r = io.LimitReader(f, limit+1)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if limit > 0 && int64(len(b)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return b, nil
}

var loadFileFormatOptions = []core.UIDropdownOption{
	{Name: "Raw bytes", Value: "raw"},
	{Name: "Stream", Value: "stream"},
	{Name: "CSV", Value: "csv"},
	{Name: "JSON", Value: "json"},
}

func NewLoadFileNode(path string) *core.Node {
	formatDropdown := core.UIDropdown{
		Options: loadFileFormatOptions,
	}

	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		formatDropdown.SelectByValue(ext[1:])
	}

	return &core.Node{
		Name: "Load File",

		InputPorts: []core.NodePort{{
			Name: "Path",
			Type: core.FlowType{Kind: core.FSKindBytes},
		}},
		OutputPorts: []core.NodePort{{
			Name: "Data",
			Type: core.FlowType{Kind: core.FSKindBytes},
		}},

		Action: &LoadFileAction{
			Path:       path,
			Format:     formatDropdown,
			InferTypes: true,
		},
	}
}

var _ core.NodeAction = &LoadFileAction{}

func (c *LoadFileAction) UpdateAndValidate(n *core.Node) {
	isListInput := false
	if wire, ok := n.GetInputWire(0); ok {
		switch wire.Type().Kind {
		case core.FSKindBytes:
			n.InputPorts[0].Type = core.FlowType{Kind: core.FSKindBytes}
		case core.FSKindList:
			isListInput = true
			n.InputPorts[0].Type = core.NewListType(core.FlowType{Kind: core.FSKindBytes})
		default:
			n.InputPorts[0].Type = core.FlowType{Kind: core.FSKindBytes}
		}
	} else {
		n.InputPorts[0].Type = core.FlowType{Kind: core.FSKindBytes}
	}

	switch c.Format.GetSelectedOption().Value {
	case "raw":
		if len(n.OutputPorts) == 0 {
			n.OutputPorts = []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindBytes}}}
		}
		if isListInput {
			n.OutputPorts[0].Type = core.NewListType(core.FlowType{Kind: core.FSKindBytes})
		} else {
			n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindBytes}
		}
	case "stream":
		if len(n.OutputPorts) == 0 {
			n.OutputPorts = []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindStream}}}
		}
		if isListInput {
			n.OutputPorts[0].Type = core.NewListType(core.FlowType{Kind: core.FSKindStream})
		} else {
			n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindStream}
		}
	case "csv":
		if res, ok := n.GetResult(); ok && len(res.Outputs) > 0 && res.Outputs[0].Type != nil && res.Outputs[0].Type.Kind == core.FSKindTable {
			if len(n.OutputPorts) == 0 {
				n.OutputPorts = []core.NodePort{{Name: "Data", Type: *res.Outputs[0].Type}}
			} else {
				n.OutputPorts[0].Type = *res.Outputs[0].Type
			}
		} else {
			if len(n.OutputPorts) == 0 {
				n.OutputPorts = []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindTable}}}
			} else {
				n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindTable}
			}
		}
	case "json":
		if len(n.OutputPorts) == 0 {
			n.OutputPorts = []core.NodePort{{Name: "Data", Type: core.FlowType{Kind: core.FSKindAny}}}
		}
		if isListInput {
			n.OutputPorts[0].Type = core.NewListType(core.FlowType{Kind: core.FSKindAny})
		} else {
			n.OutputPorts[0].Type = core.FlowType{Kind: core.FSKindAny}
		}
	}

	n.Valid = true
}

func (c *LoadFileAction) UI(n *core.Node) {
	clay.CLAY(clay.IDI("LoadFileContainer", n.ID), clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          core.GROWH,
			ChildGap:        core.S2,
		},
	}, func() {
		clay.CLAY(clay.IDI("LoadFileRow1", n.ID), clay.EL{
			Layout: clay.LAY{
				Sizing:         core.GROWH,
				ChildAlignment: core.YCENTER,
			},
		}, func() {
			core.PortAnchor(n, false, 0)
			core.UITextBox(clay.IDI("LoadFilePath", n.ID), &c.Path, core.UITextBoxConfig{
				El: clay.EL{
					Layout: clay.LAY{Sizing: core.GROWH},
				},
				Disabled: n.InputIsWired(0),
			})
			core.UISpacer(clay.IDI("LoadFileSpacer", n.ID), core.W2)

			core.UIButton(clay.IDI("LoadFileBrowse", n.ID), core.UIButtonConfig{
				OnClick: func(_ clay.ElementID, _ clay.PointerData, _ any) {
					cwd, _ := os.Getwd()
					path, ok, err := core.OpenFileDialog("Load File", cwd, nil)
					if err == nil && ok {
						c.Path = path
					}
				},
				Disabled: n.InputIsWired(0),
				ZIndex:   core.Z_NODE_BUTTON,
			}, func() {
				clay.TEXT("Browse...", clay.TextElementConfig{TextColor: core.White})
			})

			core.UISpacer(clay.IDI("LoadFileSpacer2", n.ID), core.W2)
			if len(n.OutputPorts) > 0 {
				core.UIOutputPort(n, 0)
			}
		})

		c.Format.Do(clay.IDI("LoadFileFormat", n.ID), core.UIDropdownConfig{
			El: clay.EL{
				Layout: clay.LAY{Sizing: core.GROWH},
			},
			OnChange: func(before, after any) {
				n.ClearResult()
			},
		})

		if c.Format.GetSelectedOption().Value == "csv" {
			core.UICheckbox(clay.IDI("LoadFileInferTypes", n.ID), &c.InferTypes, "Infer Types")
		}
	})
}

func (c *LoadFileAction) RunContext(ctx context.Context, n *core.Node) <-chan core.NodeActionResult {
	done := make(chan core.NodeActionResult)
	go func() {
		var res core.NodeActionResult
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				res = core.NodeActionResult{Err: fmt.Errorf("panic in node %s: %v", n.Name, r)}
			}
			done <- res
		}()

		select {
		case <-ctx.Done():
			res.Err = ctx.Err()
			return
		default:
		}

		var paths []string
		fmt.Printf("LoadFile: Starting execution\n")
		wireVal, hasWire, err := n.GetInputValue(0)
		if err != nil {
			res.Err = err
			return
		}

		if hasWire {
			switch wireVal.Type.Kind {
			case core.FSKindList:
				for _, v := range wireVal.ListValue {
					if v.Type.Kind == core.FSKindBytes {
						paths = append(paths, string(v.BytesValue))
					} else {
						// Try to convert to string or just skip/error?
						// For now, let's assume strict typing or best effort.
						// The UpdateAndValidate checks for core.FSKindList, but doesn't check contained type strictly yet?
						// Actually UpdateAndValidate doesn't check contained type.
						// Let's best-effort convert.
						paths = append(paths, fmt.Sprintf("%v", v)) // TODO: Better string conversion
					}
				}
			case core.FSKindBytes:
				paths = append(paths, string(wireVal.BytesValue))
			default:
				res.Err = fmt.Errorf("unsupported input type %v", wireVal.Type.Kind)
				return
			}
		} else {
			paths = []string{c.Path}
		}

		switch format := c.Format.GetSelectedOption().Value; format {
		case "raw":
			fmt.Printf("LoadFile: Mode RAW\n")
			var outputs []core.FlowValue
			for _, path := range paths {
				// Check context
				if ctx.Err() != nil {
					res.Err = ctx.Err()
					return
				}

				content, err := readAllWithLimit(path, maxLoadFileBytes)
				if err != nil {
					res.Err = fmt.Errorf("failed to read %s: %w", path, err)
					return
				}
				outputs = append(outputs, core.NewBytesValue(content))
			}

			if hasWire && wireVal.Type.Kind == core.FSKindList {
				res = core.NodeActionResult{
					Outputs: []core.FlowValue{core.NewListValue(core.FlowType{Kind: core.FSKindBytes}, outputs)},
				}
			} else {
				// Single output if single input
				if len(outputs) == 1 {
					res = core.NodeActionResult{Outputs: []core.FlowValue{outputs[0]}}
				} else {
					// Should not happen given logic above
					res = core.NodeActionResult{Outputs: []core.FlowValue{outputs[0]}}
				}
			}

		case "stream":
			var outputs []core.FlowValue
			for _, path := range paths {
				if ctx.Err() != nil {
					// Close already opened files
					for _, out := range outputs {
						_ = out.StreamValue.Close()
					}
					res.Err = ctx.Err()
					return
				}

				f, err := os.Open(path)
				if err != nil {
					// Close already opened files
					for _, out := range outputs {
						_ = out.StreamValue.Close()
					}
					res.Err = fmt.Errorf("failed to open %s: %w", path, err)
					return
				}
				outputs = append(outputs, core.FlowValue{
					Type:        &core.FlowType{Kind: core.FSKindStream},
					StreamValue: f,
				})
			}

			if hasWire && wireVal.Type.Kind == core.FSKindList {
				res = core.NodeActionResult{
					Outputs: []core.FlowValue{core.NewListValue(core.FlowType{Kind: core.FSKindStream}, outputs)},
				}
			} else {
				if len(outputs) == 1 {
					res = core.NodeActionResult{Outputs: []core.FlowValue{outputs[0]}}
				} else {
					res = core.NodeActionResult{Outputs: []core.FlowValue{outputs[0]}}
				}
			}

		case "csv":
			fmt.Printf("LoadFile: Mode CSV, paths: %v\n", paths)
			var allHeader []string
			var allDataRows [][]string
			var colIsInt []bool
			var colIsFloat []bool
			var colSeenNonEmpty []bool

			for i, path := range paths {
				// Check context
				if ctx.Err() != nil {
					res.Err = ctx.Err()
					return
				}

				f, err := os.Open(path)
				if err != nil {
					fmt.Printf("LoadFile: Failed to open %s: %v\n", path, err)
					res.Err = fmt.Errorf("failed to open %s: %w", path, err)
					return
				}
				fmt.Printf("LoadFile: Opened %s\n", path)
				r := csv.NewReader(f)
				r.FieldsPerRecord = -1

				header, err := r.Read()
				if err != nil {
					_ = f.Close()
					if err == io.EOF {
						continue
					}
					res.Err = fmt.Errorf("failed to read CSV header %s: %w", path, err)
					return
				}
				// Trim spaces from headers
				for i := range header {
					header[i] = strings.TrimSpace(header[i])
				}
				header = append([]string(nil), header...)

				if i == 0 {
					allHeader = header
					colIsInt = make([]bool, len(allHeader))
					colIsFloat = make([]bool, len(allHeader))
					colSeenNonEmpty = make([]bool, len(allHeader))
					for j := range allHeader {
						colIsInt[j] = true
						colIsFloat[j] = true
					}
				} else {
					if len(header) != len(allHeader) {
						_ = f.Close()
						res.Err = fmt.Errorf("CSV header mismatch in %s: expected %d columns, got %d", path, len(allHeader), len(header))
						return
					}
				}

				for {
					select {
					case <-ctx.Done():
						_ = f.Close()
						res.Err = ctx.Err()
						return
					default:
					}

					record, err := r.Read()
					if err != nil {
						_ = f.Close()
						if err == io.EOF {
							break
						}
						res.Err = fmt.Errorf("failed to read CSV %s: %w", path, err)
						return
					}

					row := append([]string(nil), record...)
					allDataRows = append(allDataRows, row)

					if c.InferTypes {
						for col := 0; col < len(allHeader); col++ {
							if col >= len(row) {
								continue
							}
							val := row[col]
							if val == "" {
								continue
							}
							colSeenNonEmpty[col] = true
							if colIsInt[col] {
								if _, err := strconv.ParseInt(val, 10, 64); err != nil {
									colIsInt[col] = false
								}
							}
							if colIsFloat[col] {
								if _, err := strconv.ParseFloat(val, 64); err != nil {
									colIsFloat[col] = false
								}
							}
						}
					}
				}

				_ = f.Close()
			}

			if len(allHeader) == 0 {
				// Empty table
				res = core.NodeActionResult{
					Outputs: []core.FlowValue{{
						Type: &core.FlowType{
							Kind: core.FSKindTable,
							ContainedType: &core.FlowType{
								Kind:   core.FSKindRecord,
								Fields: nil,
							},
						},
					}},
				}
				return
			}

			// Process all collected rows
			numCols := len(allHeader)
			colTypes := make([]core.FlowTypeKind, numCols)
			for i := range colTypes {
				colTypes[i] = core.FSKindBytes
			}

			if c.InferTypes {
				for col := 0; col < numCols; col++ {
					if !colSeenNonEmpty[col] {
						continue
					}
					if colIsInt[col] {
						colTypes[col] = core.FSKindInt64
					} else if colIsFloat[col] {
						colTypes[col] = core.FSKindFloat64
					}
				}
			}

			// Build schema
			tableRecordType := core.FlowType{Kind: core.FSKindRecord}
			for i, headerField := range allHeader {
				tableRecordType.Fields = append(tableRecordType.Fields, core.FlowField{
					Name: headerField,
					Type: &core.FlowType{Kind: colTypes[i]},
				})
			}

			// Build rows
			var tableRows [][]core.FlowValueField
			for _, row := range allDataRows {
				select {
				case <-ctx.Done():
					res.Err = ctx.Err()
					return
				default:
				}

				var flowRow []core.FlowValueField
				for col, value := range row {
					if col >= numCols {
						continue
					}

					var flowValue core.FlowValue
					switch colTypes[col] {
					case core.FSKindInt64:
						if value == "" {
							flowValue = core.NewInt64Value(0, 0)
						} else {
							val, _ := strconv.ParseInt(value, 10, 64)
							flowValue = core.NewInt64Value(val, 0)
						}
					case core.FSKindFloat64:
						if value == "" {
							flowValue = core.NewFloat64Value(0, 0)
						} else {
							val, _ := strconv.ParseFloat(value, 64)
							flowValue = core.NewFloat64Value(val, 0)
						}
					default:
						flowValue = core.NewStringValue(value)
					}

					flowRow = append(flowRow, core.FlowValueField{
						Name:  allHeader[col],
						Value: flowValue,
					})
				}

				// Fill missing
				if len(row) < numCols {
					for col := len(row); col < numCols; col++ {
						var flowValue core.FlowValue
						switch colTypes[col] {
						case core.FSKindInt64:
							flowValue = core.NewInt64Value(0, 0)
						case core.FSKindFloat64:
							flowValue = core.NewFloat64Value(0, 0)
						default:
							flowValue = core.NewStringValue("")
						}
						flowRow = append(flowRow, core.FlowValueField{
							Name:  allHeader[col],
							Value: flowValue,
						})
					}
				}

				tableRows = append(tableRows, flowRow)
			}

			res = core.NodeActionResult{
				Outputs: []core.FlowValue{{
					Type: &core.FlowType{
						Kind:          core.FSKindTable,
						ContainedType: &tableRecordType,
					},
					TableValue: tableRows,
				}},
			}
			fmt.Printf("LoadFile: CSV done, rows: %d\n", len(tableRows))
		case "json":
			var outputs []core.FlowValue
			for _, path := range paths {
				if ctx.Err() != nil {
					res.Err = ctx.Err()
					return
				}

				f, err := os.Open(path)
				if err != nil {
					res.Err = fmt.Errorf("failed to open %s: %w", path, err)
					return
				}

				var v any
				decoder := json.NewDecoder(f)
				if err := decoder.Decode(&v); err != nil {
					_ = f.Close()
					res.Err = fmt.Errorf("failed to decode JSON %s: %w", path, err)
					return
				}
				_ = f.Close()

				fv, err := core.NativeToFlowValue(v)
				if err != nil {
					res.Err = fmt.Errorf("failed to convert JSON to core.FlowValue in %s: %w", path, err)
					return
				}
				outputs = append(outputs, fv)
			}

			if hasWire && wireVal.Type.Kind == core.FSKindList {
				res = core.NodeActionResult{
					Outputs: []core.FlowValue{core.NewListValue(core.FlowType{Kind: core.FSKindAny}, outputs)},
				}
			} else {
				if len(outputs) == 1 {
					res = core.NodeActionResult{Outputs: []core.FlowValue{outputs[0]}}
				} else {
					// Should not happen
					res = core.NodeActionResult{Outputs: []core.FlowValue{outputs[0]}}
				}
			}

		default:
			res.Err = fmt.Errorf("unknown format \"%v\"", format)
		}
	}()
	return done
}

func (c *LoadFileAction) Run(n *core.Node) <-chan core.NodeActionResult {
	return c.RunContext(context.Background(), n)
}

func (c *LoadFileAction) Serialize(s *core.Serializer) bool {
	core.SStr(s, &c.Path)
	core.SBool(s, &c.InferTypes)

	if s.Encode {
		val := ""
		opt := c.Format.GetSelectedOption()
		if v, ok := opt.Value.(string); ok {
			val = v
		}
		s.WriteStr(val)
	} else {
		val, ok := s.ReadStr()
		if !ok {
			return false
		}
		c.Format = core.UIDropdown{Options: loadFileFormatOptions}
		c.Format.SelectByValue(val)
	}

	return s.Ok()
}
