package main

import (
	"context"
	"fmt"
	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/tm"
	"time"
)

type Foo struct {
	id   int
	name string
}

func main() {
	client.Init()
	initService()
	tm.WithGlobalTx(context.Background(), &tm.TransactionInfo{
		Name: "ATSampleLocalGlobalTx",
	}, updateData)
	<-make(chan struct{})
}

func selectData() {
	var foo Foo
	row := db.QueryRow("select * from  foo where id = ? ", 1)
	err := row.Scan(&foo.id, &foo.name)
	if err != nil {
		panic(err)
	}
	fmt.Println(foo)
}

func updateData(ctx context.Context) error {
	sql := "update foo set name=? where id=?"
	ret, err := db.ExecContext(ctx, sql, fmt.Sprintf("张三-%d", time.Now().UnixMilli()), 1)
	if err != nil {
		fmt.Printf("更新失败, err:%v\n", err)
		return nil
	}
	rows, err := ret.RowsAffected()
	if err != nil {
		fmt.Printf("更新行失败, err:%v\n", err)
		return nil
	}
	fmt.Printf("更新成功, 更新的行数： %d.\n", rows)
	return nil
}

func insertData(ctx context.Context) error {
	sqlStr := "insert into foo(name) values (?)"
	ret, err := db.ExecContext(ctx, sqlStr, fmt.Sprintf("张三-%d", time.Now().UnixMilli()))
	if err != nil {
		fmt.Printf("insert failed, err:%v\n", err)
		return err
	}
	theID, err := ret.LastInsertId() // 新插入数据的id
	if err != nil {
		fmt.Printf("get lastinsert ID failed, err:%v\n", err)
		return err
	}
	fmt.Printf("insert success, the id is %d.\n", theID)
	return nil
}
