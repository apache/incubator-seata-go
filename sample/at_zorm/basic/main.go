package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/tm"
)

func main() {
	client.InitPath("./sample/conf/seatago.yml")
	initService()
	tm.WithGlobalTx(context.Background(), &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}, insertData)

	<-make(chan struct{})
}

func queryData(ctx context.Context) error {
	var demo orderStruct
	finder := zorm.NewSelectFinder(demo.GetTableName())
	finder.Append("WHERE id=?", "230117")
	has, err := zorm.QueryRow(ctx, finder, &demo)
	if err != nil {
		fmt.Printf("err: %v", err)
		return err
	}
	data, _ := json.Marshal(demo)
	fmt.Println(string(data))
	fmt.Println(has)
	return nil
}

func insertData(ctx context.Context) error {
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		var order orderStruct
		order.Id = 230117
		order.UserId = "230117"
		order.CommodityCode = "2"
		order.Count = 2
		order.Money = 300
		order.Descs = "At the company's New Year's dinner, I got two Jingdong gift cards worth 600 yuan"
		_, err := zorm.Insert(ctx, &order)
		return nil, err
	})
	if err != nil {
		fmt.Printf("err: %v", err)
		return err
	}
	return nil
}

func deleteData(ctx context.Context) error {
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		demo := &orderStruct{
			Id: 230117,
		}
		_, err := zorm.Delete(ctx, demo)
		return nil, err
	})
	if err != nil {
		fmt.Printf("err: %v", err)
		return err
	}
	return nil
}

func updateData(ctx context.Context) error {
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		order := &orderStruct{}
		order.Id = 230117
		order.Descs = "The company issued two Jingdong gift cards, worth 200 yuan."
		_, err := zorm.UpdateNotZeroValue(ctx, order)
		return nil, err
	})
	if err != nil {
		fmt.Printf("err: %v", err)
		return err
	}
	return nil
}
