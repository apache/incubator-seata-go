package schema

import (
	"github.com/pkg/errors"
)

type TableMeta struct {
	TableName  string
	Columns    []string
	AllColumns map[string]ColumnMeta
	AllIndexes map[string]IndexMeta
}

func (meta TableMeta) GetPrimaryKeyMap() map[string]ColumnMeta {
	pk := make(map[string]ColumnMeta)
	for _, index := range meta.AllIndexes {
		if index.IndexType == IndexType_PRIMARY {
			for _, col := range index.Values {
				pk[col.ColumnName] = col
			}
		}
	}
	if len(pk) > 1 {
		panic(errors.Errorf("%s contains multi PK, but current not support.", meta.TableName))
	}
	if len(pk) < 1 {
		panic(errors.Errorf("%s needs to contain the primary key.", meta.TableName))
	}
	return pk
}

func (meta TableMeta) GetPrimaryKeyOnlyName() []string {
	list := make([]string, 0)
	pk := meta.GetPrimaryKeyMap()
	for key := range pk {
		list = append(list, key)
	}
	return list
}

func (meta TableMeta) GetPkName() string {
	return meta.GetPrimaryKeyOnlyName()[0]
}
