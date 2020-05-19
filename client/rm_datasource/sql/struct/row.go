package _struct

import "github.com/pkg/errors"

type Row struct {
	Fields []Field
}

func (row Row) PrimaryKeys() []Field {
	fileds := make([]Field,0)
	for _,field := range fileds {
		if field.KeyType == PRIMARY_KEY {
			fileds = append(fileds,field)
		}
	}
	if len(fileds) > 1 {
		panic(errors.New("Multi-PK"))
	}
	return fileds
}

func (row Row) NonPrimaryKeys() []Field {
	fileds := make([]Field,0)
	for _,field := range fileds {
		if field.KeyType != PRIMARY_KEY {
			fileds = append(fileds,field)
		}
	}
	return fileds
}

