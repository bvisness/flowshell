package app

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/bvisness/flowshell/clay"
	"github.com/bvisness/flowshell/util"
)

type SaveFileAction struct {
	path   string
	format UIDropdown
}

func NewSaveFileNode(path string) *Node {
	formatDropdown := UIDropdown{
		Options: saveFileFormatOptions,
	}

	return &Node{
		ID:   NewNodeID(),
		Name: "Save File",

		InputPorts: []NodePort{
			{
				Name: "Path",
				Type: FlowType{Kind: FSKindBytes},
			},
			{
				Name: "Data",
				Type: FlowType{Kind: FSKindAny},
			},
		},
		OutputPorts: []NodePort{{
			Name: "Data",
			Type: FlowType{Kind: FSKindAny},
		}},

		Action: &SaveFileAction{
			path:   path,
			format: formatDropdown,
		},
	}
}

var saveFileFormatOptions = []UIDropdownOption{
	{Name: "Raw bytes", Value: "raw"},
	{Name: "CSV", Value: "csv"},
	{Name: "JSON", Value: "json"},
}

var _ NodeAction = &SaveFileAction{}

func (c *SaveFileAction) UpdateAndValidate(n *Node) {
	n.Valid = true

	data, dataWired := n.GetInputWire(1)
	if dataWired {
		n.OutputPorts[0].Type = data.Type()
	} else {
		n.OutputPorts[0].Type = FlowType{Kind: FSKindAny}
		n.Valid = false
	}
}

func (c *SaveFileAction) UI(n *Node) {
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

		UIInputPort(n, 1)

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

func (c *SaveFileAction) Run(n *Node) <-chan NodeActionResult {
	done := make(chan NodeActionResult)

	go func() {
		var res NodeActionResult
		defer func() { done <- res }()

		data, ok, err := n.GetInputValue(1)
		if !ok {
			panic("should have had input data due to validation")
		}
		if err != nil {
			res.Err = err
			return
		}

		primitiveValueToBytes := func(v FlowValue) ([]byte, error) {
			switch v.Type.Kind {
			case FSKindBytes:
				return v.BytesValue, nil
			case FSKindInt64:
				return []byte(fmt.Sprintf("%v", v.Int64Value)), nil
			case FSKindFloat64:
				return []byte(fmt.Sprintf("%v", v.Float64Value)), nil
			default:
				return nil, fmt.Errorf("cannot write type %s as raw bytes - use another format like CSV instead", v.Type)
			}
		}

		var outputBytes []byte
		switch format := c.format.GetSelectedOption().Value; format {
		case "raw":
			var err error
			outputBytes, err = primitiveValueToBytes(data)
			if err != nil {
				res.Err = err
				return
			}
		case "csv":
			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			switch data.Type.Kind {
			case FSKindBytes, FSKindInt64, FSKindFloat64:
				prim, err := primitiveValueToBytes(data)
				if err != nil {
					res.Err = err
					return
				}
				w.Write([]string{string(prim)})
			case FSKindList:
				// one line per value
				for _, v := range data.ListValue {
					prim, err := primitiveValueToBytes(v)
					if err != nil {
						res.Err = err
						return
					}
					w.Write([]string{string(prim)})
				}
			case FSKindRecord:
				var headers []string
				var values []string
				for _, f := range data.RecordValue {
					prim, err := primitiveValueToBytes(f.Value)
					if err != nil {
						res.Err = err
						return
					}

					headers = append(headers, f.Name)
					values = append(values, string(prim))
				}
				w.Write(headers)
				w.Write(values)
			case FSKindTable:
				var headers []string
				for _, f := range data.Type.ContainedType.Fields {
					headers = append(headers, f.Name)
				}
				w.Write(headers)

				for _, row := range data.TableValue {
					var values []string
					for _, v := range row {
						prim, err := primitiveValueToBytes(v.Value)
						if err != nil {
							res.Err = err
							return
						}
						values = append(values, string(prim))
					}
					w.Write(values)
				}
			default:
				res.Err = fmt.Errorf("can't convert type %s to CSV", data.Type)
				return
			}

			w.Flush()
			outputBytes = buf.Bytes()
		default:
			res.Err = fmt.Errorf("unknown format \"%v\"", format)
			return
		}

		err = os.WriteFile(c.path, outputBytes, 0666) // TODO: get path from port
		if err != nil {
			res.Err = err
			return
		}

		res = NodeActionResult{
			Outputs: []FlowValue{data},
		}
	}()

	return done
}

var _ Serializable[SaveFileAction] = SaveFileAction{}

func (SaveFileAction) Serialize(s *Serializer, n *SaveFileAction) error {
	SStr(s, &n.path)

	if s.Encode {
		s.WriteStr(n.format.GetSelectedOption().Name)
	} else {
		selected, _ := s.ReadStr()
		n.format = UIDropdown{Options: saveFileFormatOptions}
		n.format.SelectByName(selected)
		util.Assert(n.format.GetSelectedOption().Name == selected, "format %s should have been selected, but %s was instead", selected, n.format.GetSelectedOption().Name)
	}
	return s.Err
}
