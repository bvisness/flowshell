package app

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

// GEN:NodeAction
type LoadFileAction struct {
	path string

	format UIDropdown

	// TODO: In reality this should be a more complex thing. For now we will just
	// always parse them as floats. (The "right way" to do it would be to have
	// CSV always parse as strings, but make it clear in the UI that they are
	// strings, and then have the user convert them to numbers. Perhaps this
	// could be done with a "Convert to Number" node that works on single
	// strings, lists, records, and tables. But perhaps you'd want to be able to
	// easily apply it to specific columns of a table? Maybe implicit conversion
	// to number would be ok within the Aggregate node and other nodes that do
	// math? Who knows. Very large design space. For now we just demo by always
	// parsing as float.
	csvNumbers bool
}

var loadFileFormatOptions = []UIDropdownOption{
	{Name: "Raw bytes", Value: "raw"},
	{Name: "CSV", Value: "csv"},
	{Name: "JSON", Value: "json"},
}

// TODO: Make this node polymorphic on lists of strings
// (rename to "Load Files" dynamically)
func NewLoadFileNode(path string) *Node {
	formatDropdown := UIDropdown{
		Options: loadFileFormatOptions,
	}

	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		formatDropdown.SelectByValue(ext[1:])
	}

	return &Node{
		ID:   NewNodeID(),
		Name: "Load File",

		InputPorts: []NodePort{{
			Name: "Path",
			Type: FlowType{Kind: FSKindBytes},
		}},
		OutputPorts: []NodePort{{
			Name: "Data",
			Type: FlowType{Kind: FSKindBytes},
		}},

		Action: &LoadFileAction{
			path:       path,
			format:     formatDropdown,
			csvNumbers: true,
		},
	}
}

var _ NodeAction = &LoadFileAction{}

func (c *LoadFileAction) UpdateAndValidate(n *Node) {
	switch c.format.GetSelectedOption().Value {
	case "raw":
		n.OutputPorts[0].Type = FlowType{Kind: FSKindBytes}
	case "csv":
		n.OutputPorts[0].Type = FlowType{Kind: FSKindTable, ContainedType: &FlowType{Kind: FSKindAny}}
	case "json":
		n.OutputPorts[0].Type = FlowType{Kind: FSKindAny}
	}

	n.Valid = true
}

func (c *LoadFileAction) UI(n *Node) {
	clay.CLAY_AUTO_ID(clay.EL{
		Layout: clay.LAY{
			LayoutDirection: clay.TopToBottom,
			Sizing:          GROWH,
			ChildGap:        S2,
		},
	}, func() {
		clay.CLAY_AUTO_ID(clay.EL{
			Layout: clay.LAY{
				Sizing:         GROWH,
				ChildAlignment: YCENTER,
			},
		}, func() {
			PortAnchor(n, false, 0)
			UITextBox(clay.IDI("LoadFilePath", n.ID), &c.path, UITextBoxConfig{
				El: clay.EL{
					Layout: clay.LAY{Sizing: GROWH},
				},
				Disabled: n.InputIsWired(0),
			})
			UISpacer(clay.AUTO_ID, W2)
			UIOutputPort(n, 0)
		})

		c.format.Do(clay.AUTO_ID, UIDropdownConfig{
			El: clay.EL{
				Layout: clay.LAY{Sizing: GROWH},
			},
			OnChange: func(before, after any) {
				n.ClearResult()
			},
		})
	})
}

func (c *LoadFileAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		content, err := os.ReadFile(c.path) // TODO: Get path from port
		if err != nil {
			res.Err = err
			return
		}

		switch format := c.format.GetSelectedOption().Value; format {
		case "raw":
			res = NodeActionResult{
				Outputs: []FlowValue{NewBytesValue(content)},
			}
		case "csv":
			r := csv.NewReader(bytes.NewReader(content))
			rows, err := r.ReadAll()
			if err != nil {
				res.Err = err
				return
			}

			// Special case: if we don't even get a row, synthesize an empty table with no columns.
			if len(rows) == 0 {
				res = NodeActionResult{
					Outputs: []FlowValue{{
						Type: &FlowType{
							Kind: FSKindTable,
							ContainedType: &FlowType{
								Kind:   FSKindRecord,
								Fields: nil,
							},
						},
					}},
				}
				return
			}

			tableRecordType := FlowType{Kind: FSKindRecord}
			for _, headerField := range rows[0] {
				tableRecordType.Fields = append(tableRecordType.Fields, FlowField{
					Name: headerField,
					Type: &FlowType{Kind: util.Tern(c.csvNumbers, FSKindFloat64, FSKindBytes)},
				})
			}

			// TODO(low): Be resilient against variable numbers of fields per row, potentially
			var tableRows [][]FlowValueField
			for _, row := range rows[1:] {
				var flowRow []FlowValueField
				for col, value := range row {
					floatVal, err := strconv.ParseFloat(value, 64)
					if err != nil {
						res.Err = err
						return
					}
					flowRow = append(flowRow, FlowValueField{
						Name:  rows[0][col],
						Value: util.Tern(c.csvNumbers, NewFloat64Value(floatVal, 0), NewStringValue(value)),
					})
				}
				tableRows = append(tableRows, flowRow)
			}

			res = NodeActionResult{
				Outputs: []FlowValue{{
					Type: &FlowType{
						Kind:          FSKindTable,
						ContainedType: &tableRecordType,
					},
					TableValue: tableRows,
				}},
			}
		default:
			res.Err = fmt.Errorf("unknown format \"%v\"", format)
		}
	}()

	return done
}

func (n *LoadFileAction) Serialize(s *Serializer) bool {
	SStr(s, &n.path)
	SBool(s, &n.csvNumbers)

	if s.Encode {
		s.WriteStr(n.format.GetSelectedOption().Name)
	} else {
		selected, ok := s.ReadStr()
		if !ok {
			return false
		}
		n.format = UIDropdown{Options: loadFileFormatOptions}
		n.format.SelectByName(selected)
		util.Assert(n.format.GetSelectedOption().Name == selected, "format %s should have been selected, but %s was instead", selected, n.format.GetSelectedOption().Name)
	}

	return s.Ok()
}
