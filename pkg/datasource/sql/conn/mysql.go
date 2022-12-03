package conn

import (
	"context"
	"database/sql/driver"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlXAConn struct {
	driver.Conn
}

func (c *MysqlXAConn) Commit(xid string, onePhase bool) error {
	var sb strings.Builder
	sb.WriteString("XA COMMIT ")
	sb.WriteString(xid)
	if onePhase {
		sb.WriteString(" ONE PHASE")
	}

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}

func (c *MysqlXAConn) End(xid string, flags int) error {
	var sb strings.Builder
	sb.WriteString("XA END ")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}

func (c *MysqlXAConn) Forget(xid string) error {
	//TODO implement me
	panic("implement me")
}

func (c *MysqlXAConn) GetTransactionTimeout() time.Duration {
	//TODO implement me
	panic("implement me")
}

func (c *MysqlXAConn) IsSameRM(resource XAResource) bool {
	//TODO implement me
	panic("implement me")
}

func (c *MysqlXAConn) XAPrepare(xid string) (int, error) {
	var sb strings.Builder
	sb.WriteString("XA PREPARE ")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	if _, err := conn.ExecContext(context.TODO(), sb.String(), nil); err != nil {
		return -1, err
	}
	return 0, nil
}

func (c *MysqlXAConn) Recover(flag int) []string {
	//TODO implement me
	panic("implement me")
}

func (c *MysqlXAConn) Rollback(xid string) error {
	var sb strings.Builder
	sb.WriteString("XA ROLLBACK ")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}

func (c *MysqlXAConn) SetTransactionTimeout(duration time.Duration) bool {
	//TODO implement me
	panic("implement me")
}

func (c *MysqlXAConn) Start(xid string, flags int) error {
	var sb strings.Builder
	sb.WriteString("XA START")
	sb.WriteString(xid)

	conn, _ := c.Conn.(driver.ExecerContext)
	_, err := conn.ExecContext(context.TODO(), sb.String(), nil)
	return err
}
