package schema

import (
	"database/sql"
	"reflect"
	"strings"
)

type TableRecords struct {
	TableMeta TableMeta `json:"-"`
	TableName string
	Rows      []*Row
}

func NewTableRecords(meta TableMeta) *TableRecords {
	return &TableRecords{
		TableMeta: meta,
		TableName: meta.TableName,
		Rows:      make([]*Row, 0),
	}
}

func (records *TableRecords) PkFields() []*Field {
	pkRows := make([]*Field, 0)
	pk := records.TableMeta.GetPkName()
	for _, row := range records.Rows {
		for _, field := range row.Fields {
			if strings.ToLower(field.Name) == strings.ToLower(pk) {
				pkRows = append(pkRows, field)
				break
			}
		}
	}
	return pkRows
}

func BuildRecords(meta TableMeta, resultSet *sql.Rows) *TableRecords {
	records := NewTableRecords(meta)
	columns, _ := resultSet.Columns()
	rows := make([]*Row, 0)
	for resultSet.Next() {
		values := make([]interface{}, 0)
		count := len(columns)
		for i := 0; i < count; i++ {
			values = append(values, new(interface{}))
		}
		resultSet.Scan(values...)
		fields := make([]*Field, 0, len(columns))
		for i, col := range columns {
			val := reflect.ValueOf(values[i])
			realVal := val.Elem().Interface()
			filed := &Field{
				Name:  col,
				Type:  meta.AllColumns[col].DataType,
				Value: realVal,
			}
			if strings.ToLower(col) == strings.ToLower(meta.GetPkName()) {
				filed.KeyType = PRIMARY_KEY
			}
			fields = append(fields, filed)
		}
		row := &Row{Fields: fields}
		rows = append(rows, row)
	}
	records.Rows = rows
	return records
}
