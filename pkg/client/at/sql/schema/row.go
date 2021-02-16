package schema

import "github.com/pkg/errors"

type Row struct {
	Fields []*Field
}

func (row *Row) PrimaryKeys() []*Field {
	fields := make([]*Field, 0)
	for _, field := range row.Fields {
		if field.KeyType == PRIMARY_KEY {
			fields = append(fields, field)
		}
	}
	if len(fields) > 1 {
		panic(errors.New("Multi-PK"))
	}
	return fields
}

func (row *Row) NonPrimaryKeys() []*Field {
	fields := make([]*Field, 0)
	for _, field := range row.Fields {
		if field.KeyType != PRIMARY_KEY {
			fields = append(fields, field)
		}
	}
	return fields
}
