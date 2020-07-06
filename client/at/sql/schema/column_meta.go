package schema

type ColumnMeta struct {
	TableCat        string
	TableSchemeName string
	TableName       string
	ColumnName      string
	DataType        int32
	DataTypeName    string
	ColumnSize      int32
	DecimalDigits   int32
	NumPrecRadix    int32
	Nullable        int32
	Remarks         string
	ColumnDef       string
	SqlDataType     int32
	SqlDatetimeSub  int32
	CharOctetLength int32
	OrdinalPosition int32
	IsNullable      string
	IsAutoIncrement string
}
