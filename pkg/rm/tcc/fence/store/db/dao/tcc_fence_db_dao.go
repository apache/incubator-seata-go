package dao

import (
	"database/sql"

	"google.golang.org/genproto/googleapis/type/date"
)

type TCCFenceStore interface {
	Query(conn sql.Conn, xid string, branchId int64) TCCFenceDO
	Insert(conn sql.Conn, xid string, branchId int64) bool
	UpdateTCCFenceDO(conn sql.Conn, xid string, branchId int64, newStatus int, oldStatus int) bool
	DeleteTCCFenceDO(conn sql.Conn, xid string, branchId int64) bool
	DeleteTCCFenceDOByDate(conn sql.Conn, datetime date.Date) bool
	SetLogTableName(logTable string)
}

type TCCFenceDO struct {
	xid         string
	branchId    int64
	actionName  string
	status      int
	gmtCreate   date.Date
	gmtModified date.Date
}
