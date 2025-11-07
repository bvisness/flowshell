package app

import (
	"fmt"
	"strings"
	"time"
)

type FlowValue struct {
	Type *FlowType

	BytesValue   []byte
	Int64Value   int64
	Float64Value float64
	ListValue    []FlowValue
	RecordValue  []FlowValueField
	TableValue   [][]FlowValueField
}

type FlowValueField struct {
	Name  string
	Value FlowValue
}

func (v *FlowValue) ColumnValues(col int) []FlowValue {
	if v.Type.Kind != FSKindTable {
		panic(fmt.Errorf("value %s was not a table", v))
	}

	var res []FlowValue
	for _, row := range v.TableValue {
		res = append(res, row[col].Value)
	}
	return res
}

func (v FlowValue) String() string {
	return "???(" + v.Type.String() + ")"
}

type FlowTypeKind int

const (
	FSKindAny FlowTypeKind = iota // not valid for use on a FlowValue
	FSKindBytes
	FSKindInt64
	FSKindFloat64
	FSKindList
	FSKindRecord
	FSKindTable
)

type FlowType struct {
	Kind FlowTypeKind

	ContainedType *FlowType   // for lists and tables
	Fields        []FlowField // for records

	// For primitive values, an optional unit to use for presentation or
	// contextual operations.
	Unit FlowUnit

	// If set, this type has been annotated as "well-known", meaning some other
	// operations may be conveniently available on it.
	WellKnownType FlowWellKnownType
}

func (t FlowType) String() string {
	switch t.WellKnownType {
	case FSWKTFile:
		return "File"
	case FSWKTTimestamp:
		return "Timestamp"
	}

	joinFields := func(fields []FlowField) string {
		var bits []string
		for _, f := range fields {
			bits = append(bits, fmt.Sprintf("%s:%s", f.Name, f.Type.String()))
		}
		return strings.Join(bits, ", ")
	}

	switch t.Kind {
	case FSKindAny:
		return "Any"
	case FSKindBytes:
		return "Bytes"
	case FSKindInt64:
		return "Int64"
	case FSKindFloat64:
		return "Float64"
	case FSKindList:
		return fmt.Sprintf("List[%s]", t.ContainedType.String())
	case FSKindRecord:
		return fmt.Sprintf("Record[%s]", joinFields(t.Fields))
	case FSKindTable:
		return fmt.Sprintf("Table[%s]", joinFields(t.ContainedType.Fields))
	default:
		return "<UNKNOWN TYPE>"
	}
}

func (t *FlowType) Serialize(s *Serializer) bool {
	SInt(s, &t.Kind)
	SMaybeThing(s, &t.ContainedType)
	SSlice(s, &t.Fields)
	SInt(s, &t.Unit)
	return s.Ok()
}

func Typecheck(a, b FlowType) error {
	// Every type matches Any.
	if b.Kind == FSKindAny {
		return nil
	}

	// Kinds must always match.
	if a.Kind != b.Kind {
		return fmt.Errorf("expected type %s, but got %s", b.String(), a.String())
	}

	switch b.Kind {
	case FSKindBytes, FSKindInt64, FSKindFloat64:
		// These are primitives, so if their kinds are the same, there is nothing else to check.
	case FSKindList, FSKindTable:
		if err := Typecheck(*a.ContainedType, *b.ContainedType); err != nil {
			return fmt.Errorf("expected type %s, but got %s: %v", b.String(), a.String(), err)
		}
	case FSKindRecord:
		if len(a.Fields) != len(b.Fields) {
			return fmt.Errorf("record types have different number of fields")
		}
		for i := range a.Fields {
			if a.Fields[i].Name != b.Fields[i].Name {
				return fmt.Errorf("fields have different names: expected %s, got %s", b.Fields[i].Name, a.Fields[i].Name)
			}
			if err := Typecheck(*a.Fields[i].Type, *b.Fields[i].Type); err != nil {
				return fmt.Errorf("bad type for field %s: %v", a.Fields[i].Name, err)
			}
		}
	}

	return nil
}

type FlowField struct {
	Name string
	Type *FlowType
}

func (f *FlowField) Serialize(s *Serializer) bool {
	SStr(s, &f.Name)
	SMaybeThing(s, &f.Type)
	return s.Ok()
}

type FlowUnit int

const (
	FSUnitBytes FlowUnit = iota + 1
	FSUnitSeconds
)

type FlowWellKnownType int

const (
	FSWKTFile FlowWellKnownType = iota + 1
	FSWKTTimestamp
)

var FSFile = &FlowType{
	Kind: FSKindRecord,
	Fields: []FlowField{
		{Name: "name", Type: &FlowType{Kind: FSKindBytes}},
		{Name: "type", Type: &FlowType{Kind: FSKindBytes}},
		{Name: "size", Type: &FlowType{Kind: FSKindInt64, Unit: FSUnitBytes}},
		{Name: "modified", Type: FSTimestamp},
	},
	WellKnownType: FSWKTFile,
}

var FSTimestamp = &FlowType{
	Kind:          FSKindInt64,
	Unit:          FSUnitSeconds,
	WellKnownType: FSWKTTimestamp,
}

// ---------------------------
// Constructors

func NewListType(contained FlowType) FlowType {
	return FlowType{
		Kind:          FSKindList,
		ContainedType: &contained,
	}
}

func NewRecordType(fields []FlowField) FlowType {
	return FlowType{
		Kind:   FSKindRecord,
		Fields: fields,
	}
}

func NewTableType(fields []FlowField) FlowType {
	return FlowType{
		Kind: FSKindTable,
		ContainedType: &FlowType{
			Kind:   FSKindRecord,
			Fields: fields,
		},
	}
}

func NewAnyTableType() FlowType {
	return FlowType{
		Kind:          FSKindTable,
		ContainedType: &FlowType{Kind: FSKindAny},
	}
}

func NewBytesValue(bytes []byte) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindBytes}, BytesValue: bytes}
}

func NewStringValue(str string) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindBytes}, BytesValue: []byte(str)}
}

func NewInt64Value(v int64, unit FlowUnit) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindInt64, Unit: unit}, Int64Value: v}
}

func NewFloat64Value(v float64, unit FlowUnit) FlowValue {
	return FlowValue{Type: &FlowType{Kind: FSKindFloat64, Unit: unit}, Float64Value: v}
}

func NewTimestampValue(t time.Time) FlowValue {
	return FlowValue{Type: FSTimestamp, Int64Value: t.Unix()}
}

func NewListValue(contained FlowType, items []FlowValue) FlowValue {
	t := NewListType(contained)
	for _, item := range items {
		if err := Typecheck(*item.Type, contained); err != nil {
			panic(err)
		}
	}
	return FlowValue{
		Type:      &t,
		ListValue: items,
	}
}
