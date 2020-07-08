package parser

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"time"
)

import (
	"github.com/golang/protobuf/proto"
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema"
	"github.com/dk-lockdown/seata-golang/client/at/sqlparser"
	"github.com/dk-lockdown/seata-golang/client/at/undo"
)

type MysqlFieldValueType byte

const (
	MysqlFieldValueType_Int64 MysqlFieldValueType = iota
	MysqlFieldValueType_Float32
	MysqlFieldValueType_Float64
	MysqlFieldValueType_Uint8Slice
	MysqlFieldValueType_Time
)

const timeFormat = "2006-01-02 15:04:05.999999"

type ProtoBufUndoLogParser struct {
}

func (parser ProtoBufUndoLogParser) GetName() string {
	return "protobuf"
}

func (parser ProtoBufUndoLogParser) GetDefaultContent() []byte {
	return []byte("[]")
}

func (parser ProtoBufUndoLogParser) Encode(branchUndoLog *undo.BranchUndoLog) []byte {
	pbBranchUndoLog := convertBranchSqlUndoLog(branchUndoLog)
	data, err := proto.Marshal(pbBranchUndoLog)
	if err != nil {
		panic(err)
	}
	return data
}

func (parser ProtoBufUndoLogParser) Decode(data []byte) *undo.BranchUndoLog {
	var pbBranchUndoLog = &PbBranchUndoLog{}
	err := proto.Unmarshal(data, pbBranchUndoLog)
	if err != nil {
		panic(err)
	}

	return convertPbBranchSqlUndoLog(pbBranchUndoLog)
}

func convertField(field *schema.Field) *PbField {
	pbField := &PbField{
		Name:    field.Name,
		KeyType: int32(field.KeyType),
		Type:    field.Type,
	}
	if field.Value == nil {
		return pbField
	}
	var buf bytes.Buffer
	w := byteio.BigEndianWriter{Writer: &buf}

	switch v := field.Value.(type) {
	case int64:
		w.WriteByte(byte(MysqlFieldValueType_Int64))
		w.WriteInt64(v)
		break
	case float32:
		w.WriteByte(byte(MysqlFieldValueType_Float32))
		w.WriteFloat32(v)
		break
	case float64:
		w.WriteByte(byte(MysqlFieldValueType_Float64))
		w.WriteFloat64(v)
		break
	case []uint8:
		w.WriteByte(byte(MysqlFieldValueType_Uint8Slice))
		w.Write(v)
		break
	case time.Time:
		var a [64]byte
		var b = a[:0]

		if v.IsZero() {
			b = append(b, "0000-00-00"...)
		} else {
			loc, _ := time.LoadLocation("Local")
			b = v.In(loc).AppendFormat(b, timeFormat)
		}
		w.WriteByte(byte(MysqlFieldValueType_Time))
		w.Write(b)
	default:
		panic(errors.Errorf("unsupport types:%s,%v", reflect.TypeOf(field.Value).String(), field.Value))
	}
	pbField.Value = buf.Bytes()
	return pbField
}

func convertPbField(pbField *PbField) *schema.Field {
	field := &schema.Field{
		Name:    pbField.Name,
		KeyType: schema.KeyType(pbField.KeyType),
		Type:    pbField.Type,
	}
	if pbField.Value == nil {
		return field
	}
	r := byteio.BigEndianReader{Reader: bytes.NewReader(pbField.Value)}
	valueType, _ := r.ReadByte()

	switch MysqlFieldValueType(valueType) {
	case MysqlFieldValueType_Int64:
		value, _, _ := r.ReadInt64()
		field.Value = value
	case MysqlFieldValueType_Float32:
		value, _, _ := r.ReadFloat32()
		field.Value = value
	case MysqlFieldValueType_Float64:
		value, _, _ := r.ReadFloat64()
		field.Value = value
	case MysqlFieldValueType_Uint8Slice:
		field.Value = pbField.Value[1:]
	case MysqlFieldValueType_Time:
		loc, _ := time.LoadLocation("Local")
		t, err := parseDateTime(
			string(pbField.Value[1:]),
			loc,
		)
		if err != nil {
			panic(err)
		}
		field.Value = t
		break
	default:
		fmt.Printf("unsupport types:%v", valueType)
		break
	}
	return field
}

func convertRow(row *schema.Row) *PbRow {
	pbFields := make([]*PbField, 0)
	for _, field := range row.Fields {
		pbField := convertField(field)
		pbFields = append(pbFields, pbField)
	}
	pbRow := &PbRow{
		Fields: pbFields,
	}
	return pbRow
}

func convertPbRow(pbRow *PbRow) *schema.Row {
	fields := make([]*schema.Field, 0)
	for _, pbField := range pbRow.Fields {
		field := convertPbField(pbField)
		fields = append(fields, field)
	}
	row := &schema.Row{Fields: fields}
	return row
}

func convertTableRecords(records *schema.TableRecords) *PbTableRecords {
	pbRows := make([]*PbRow, 0)
	for _, row := range records.Rows {
		pbRow := convertRow(row)
		pbRows = append(pbRows, pbRow)
	}
	pbRecords := &PbTableRecords{
		TableName: records.TableName,
		Rows:      pbRows,
	}
	return pbRecords
}

func convertPbTableRecords(pbRecords *PbTableRecords) *schema.TableRecords {
	rows := make([]*schema.Row, 0)
	for _, pbRow := range pbRecords.Rows {
		row := convertPbRow(pbRow)
		rows = append(rows, row)
	}
	records := &schema.TableRecords{
		TableName: pbRecords.TableName,
		Rows:      rows,
	}
	return records
}

func convertSqlUndoLog(undoLog *undo.SqlUndoLog) *PbSqlUndoLog {
	pbSqlUndoLog := &PbSqlUndoLog{
		SqlType:   int32(undoLog.SqlType),
		TableName: undoLog.TableName,
	}
	if undoLog.BeforeImage != nil {
		beforeImage := convertTableRecords(undoLog.BeforeImage)
		pbSqlUndoLog.BeforeImage = beforeImage
	}
	if undoLog.AfterImage != nil {
		afterImage := convertTableRecords(undoLog.AfterImage)
		pbSqlUndoLog.AfterImage = afterImage
	}

	return pbSqlUndoLog
}

func convertPbSqlUndoLog(pbSqlUndoLog *PbSqlUndoLog) *undo.SqlUndoLog {
	sqlUndoLog := &undo.SqlUndoLog{
		SqlType:   sqlparser.SQLType(pbSqlUndoLog.SqlType),
		TableName: pbSqlUndoLog.TableName,
	}
	if pbSqlUndoLog.BeforeImage != nil {
		beforeImage := convertPbTableRecords(pbSqlUndoLog.BeforeImage)
		sqlUndoLog.BeforeImage = beforeImage
	}
	if pbSqlUndoLog.AfterImage != nil {
		afterImage := convertPbTableRecords(pbSqlUndoLog.AfterImage)
		sqlUndoLog.AfterImage = afterImage
	}
	return sqlUndoLog
}

func convertBranchSqlUndoLog(branchUndoLog *undo.BranchUndoLog) *PbBranchUndoLog {
	sqlUndoLogs := make([]*PbSqlUndoLog, 0)
	for _, sqlUndoLog := range branchUndoLog.SqlUndoLogs {
		pbSqlUndoLog := convertSqlUndoLog(sqlUndoLog)
		sqlUndoLogs = append(sqlUndoLogs, pbSqlUndoLog)
	}
	pbBranchUndoLog := &PbBranchUndoLog{
		Xid:         branchUndoLog.Xid,
		BranchId:    branchUndoLog.BranchId,
		SqlUndoLogs: sqlUndoLogs,
	}
	return pbBranchUndoLog
}

func convertPbBranchSqlUndoLog(pbBranchUndoLog *PbBranchUndoLog) *undo.BranchUndoLog {
	sqlUndoLogs := make([]*undo.SqlUndoLog, 0)
	for _, sqlUndoLog := range pbBranchUndoLog.SqlUndoLogs {
		sqlUndoLog := convertPbSqlUndoLog(sqlUndoLog)
		sqlUndoLogs = append(sqlUndoLogs, sqlUndoLog)
	}
	branchUndoLog := &undo.BranchUndoLog{
		Xid:         pbBranchUndoLog.Xid,
		BranchId:    pbBranchUndoLog.BranchId,
		SqlUndoLogs: sqlUndoLogs,
	}
	return branchUndoLog
}

func parseDateTime(str string, loc *time.Location) (t time.Time, err error) {
	base := "0000-00-00 00:00:00.0000000"
	switch len(str) {
	case 10, 19, 21, 22, 23, 24, 25, 26: // up to "YYYY-MM-DD HH:MM:SS.MMMMMM"
		if str == base[:len(str)] {
			return
		}
		t, err = time.Parse(timeFormat[:len(str)], str)
	default:
		err = fmt.Errorf("invalid time string: %s", str)
		return
	}

	// Adjust location
	if err == nil && loc != time.UTC {
		y, mo, d := t.Date()
		h, mi, s := t.Clock()
		t, err = time.Date(y, mo, d, h, mi, s, t.Nanosecond(), loc), nil
	}

	return
}
