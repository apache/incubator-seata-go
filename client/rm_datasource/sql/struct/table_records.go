package _struct

import (
	"database/sql"
	"strings"
)

type TableRecords struct {
	TableMeta TableMeta
	TableName string
	Rows []Row
}

func NewTableRecords(meta TableMeta) TableRecords {
	return TableRecords{
		TableMeta: meta,
		TableName: meta.TableName,
		Rows:      make([]Row,0),
	}
}

func (records TableRecords) PkRows() []Field {
	pkRows := make([]Field,0)
	pk := records.TableMeta.GetPkName()
	for _,row := range records.Rows {
		for _,field := range row.Fields {
			if strings.ToLower(field.Name) == strings.ToLower(pk) {
				pkRows = append(pkRows,field)
				break
			}
		}
	}
	return pkRows
}

func BuildRecords(meta TableMeta,resultSet *sql.Rows) TableRecords {
	records := NewTableRecords(meta)
	columns,_ := resultSet.Columns()
	rows := make([]Row,0)
	for resultSet.Next() {
		values := make([]interface{},0, len(columns))
		resultSet.Scan(values...)
		fileds := make([]Field,0,len(columns))
		for i,col := range columns {
			filed := Field{
				Name:    col,
				Type:	 meta.AllColumns[col].DataType,
				Value:   values[i],
			}
			if strings.ToLower(col) == strings.ToLower(meta.GetPkName()) {
				filed.KeyType = PRIMARY_KEY
			}
			fileds = append(fileds,filed)
		}
		row := Row{Fields:fileds}
		rows = append(rows, row)
	}
	records.Rows = rows
	return records
}