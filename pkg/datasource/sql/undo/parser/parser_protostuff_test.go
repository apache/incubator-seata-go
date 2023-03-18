package parser

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func TestConvertPbBranchUndoLogToIntreeBranchUndoLog(t *testing.T) {
	str, err := structpb.NewStruct(map[string]interface{}{
		"KeyType":    int32(IndexType_IndexTypePrimaryKey),
		"ColumnName": "id",
		"ColumnType": int32(JDBCType_JDBCTypeBigInt),
		"Value":      uint64(time.Now().Second()),
	})
	assert.Nil(t, err)

	pbBranchUndoLog := BranchUndoLog{
		Xid:      "123456",
		BranchID: 123456,
		Logs: []*SQLUndoLog{
			&SQLUndoLog{
				SQLType:   SQLType_SQLTypeAlter,
				TableName: "MockTable",
				AfterImage: &RecordImage{
					TableName: "MockTable",
					SQLType:   SQLType_SQLTypeAlter,
					Rows: []*RowImage{
						&RowImage{
							Columns: []*structpb.Struct{
								str,
							},
						},
					},
				},
			},
		},
	}

	marsaller := jsonpb.Marshaler{}
	bytes, err := marsaller.MarshalToString(&pbBranchUndoLog)

	assert.Nil(t, err)
	fmt.Println(string(bytes))

	fmt.Println(ConvertPbBranchUndoLogToIntreeBranchUndoLog(&pbBranchUndoLog))
}
