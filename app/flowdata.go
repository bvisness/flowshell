package app

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
	Value *FlowValue
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

type FlowField struct {
	Name string
	Type *FlowType
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
