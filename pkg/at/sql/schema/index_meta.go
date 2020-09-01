package schema

type IndexType byte

const (
	IndexType_PRIMARY IndexType = iota
	IndexType_NORMAL
	IndexType_UNIQUE
	IndexType_FULL_TEXT
)

type IndexMeta struct {
	Values          []ColumnMeta
	NonUnique       bool
	IndexQualifier  string
	IndexName       string
	ColumnName      string
	Type            int16
	IndexType       IndexType
	AscOrDesc       string
	Cardinality     int32
	OrdinalPosition int32
}
