package dao

import (
	"time"
)

import (
	"github.com/google/uuid"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/exec"
	"github.com/transaction-wg/seata-golang/pkg/client/context"
)

const (
	insertSoMaster = `INSERT INTO seata_order.so_master (sysno, so_id, buyer_user_sysno, seller_company_code, 
		receive_division_sysno, receive_address, receive_zip, receive_contact, receive_contact_phone, stock_sysno, 
        payment_type, so_amt, status, order_date, appid, memo) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,now(),?,?)`
	insertSoItem = `INSERT INTO seata_order.so_item(sysno, so_sysno, product_sysno, product_name, cost_price, 
		original_price, deal_price, quantity) VALUES (?,?,?,?,?,?,?,?)`
)

// postgres对应的sql预警
const (
	insertSoMasterByPostgres = `INSERT INTO so_master (sysno, so_id, buyer_user_sysno, seller_company_code, 
		receive_division_sysno, receive_address, receive_zip, receive_contact, receive_contact_phone, stock_sysno, 
        payment_type, so_amt, status, order_date, appid, memo) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now(),$14,$15)`
	insertSoItemByPostgres = `INSERT INTO so_item(sysno, so_sysno, product_sysno, product_name, cost_price, 
		original_price, deal_price, quantity) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
)

type Dao struct {
	*exec.DB
}

//现实中涉及金额可能使用长整形，这里使用 float64 仅作测试，不具有参考意义

type SoMaster struct {
	SysNo                int64   `json:"sysNo"`
	SoID                 string  `json:"soID"`
	BuyerUserSysNo       int64   `json:"buyerUserSysNo"`
	SellerCompanyCode    string  `json:"sellerCompanyCode"`
	ReceiveDivisionSysNo int64   `json:"receiveDivisionSysNo"`
	ReceiveAddress       string  `json:"receiveAddress"`
	ReceiveZip           string  `json:"receiveZip"`
	ReceiveContact       string  `json:"receiveContact"`
	ReceiveContactPhone  string  `json:"receiveContactPhone"`
	StockSysNo           int64   `json:"stockSysNo"`
	PaymentType          int32   `json:"paymentType"`
	SoAmt                float64 `json:"soAmt"`
	//10，创建成功，待支付；30；支付成功，待发货；50；发货成功，待收货；70，确认收货，已完成；90，下单失败；100已作废
	Status       int32     `json:"status"`
	OrderDate    time.Time `json:"orderDate"`
	PaymentDate  time.Time `json:"paymentDate"`
	DeliveryDate time.Time `json:"deliveryDate"`
	ReceiveDate  time.Time `json:"receiveDate"`
	AppID        string    `json:"appID"`
	Memo         string    `json:"memo"`
	CreateUser   string    `json:"createUser"`
	GmtCreate    time.Time `json:"gmtCreate"`
	ModifyUser   string    `json:"modifyUser"`
	GmtModified  time.Time `json:"gmtModified"`

	SoItems []*SoItem
}

type SoItem struct {
	SysNo         int64   `json:"sysNo"`
	SoSysNo       int64   `json:"soSysNo"`
	ProductSysNo  int64   `json:"productSysNo"`
	ProductName   string  `json:"productName"`
	CostPrice     float64 `json:"costPrice"`
	OriginalPrice float64 `json:"originalPrice"`
	DealPrice     float64 `json:"dealPrice"`
	Quantity      int32   `json:"quantity"`
}

func (dao *Dao) CreateSO(ctx *context.RootContext, soMasters []*SoMaster) ([]uint64, error) {
	result := make([]uint64, 0, len(soMasters))
	tx, err := dao.Begin(ctx)
	if err != nil {
		return nil, err
	}
	for _, soMaster := range soMasters {
		soid := NextID()
		_, err = tx.Exec(insertSoMasterByPostgres, soid, soid, soMaster.BuyerUserSysNo, soMaster.SellerCompanyCode, soMaster.ReceiveDivisionSysNo,
			soMaster.ReceiveAddress, soMaster.ReceiveZip, soMaster.ReceiveContact, soMaster.ReceiveContactPhone, soMaster.StockSysNo,
			soMaster.PaymentType, soMaster.SoAmt, soMaster.Status, soMaster.AppID, soMaster.Memo)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		soItems := soMaster.SoItems
		for _, soItem := range soItems {
			soItemID := NextID()
			_, err = tx.Exec(insertSoItemByPostgres, soItemID, soid, soItem.ProductSysNo, soItem.ProductName, soItem.CostPrice, soItem.OriginalPrice,
				soItem.DealPrice, soItem.Quantity)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
		}
		result = append(result, soid)
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NextID() uint64 {
	id, _ := uuid.NewUUID()
	return uint64(id.ID())
}
