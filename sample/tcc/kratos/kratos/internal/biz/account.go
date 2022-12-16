package biz

import (
	"context"

	"github.com/seata/seata-go/pkg/tm"

	v1 "kratos/api/account/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

// Account is an Account model.
type Account struct {
	Hello string
}

// AccountRepo is an Account repo.
type AccountRepo interface {
	Save(context.Context, *Account) (*Account, error)
	Update(context.Context, *Account) (*Account, error)
	FindByID(context.Context, int64) (*Account, error)
	ListByHello(context.Context, string) ([]*Account, error)
	ListAll(context.Context) ([]*Account, error)
}

// TransFrom transfer from an account
type TransFrom struct {
	repo AccountRepo
	log  *log.Helper
}

// NewTransFrom new an Account usecase.
func NewTransFrom(repo AccountRepo, logger log.Logger) *TransFrom {
	return &TransFrom{repo: repo, log: log.NewHelper(logger)}
}

// UpdateAccount update the account
func (uc *TransFrom) UpdateAccount(ctx context.Context, g *Account) (*Account, error) {
	uc.log.WithContext(ctx).Infof("UpdateAccount: %v", g.Hello)
	return uc.repo.Update(ctx, g)
}

func (uc *TransFrom) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("check if the account is enough to transfer(e.g ￥30)," +
		"if the answer is yes, then lock the ￥30" +
		"if the answer is no, just return false")
	return true, nil
}

// Commit execute the commit, please be sure the method is idempotent
func (uc *TransFrom) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("transfer from the account")
	return true, nil
}

// Rollback execute the rollback, please attention to Blank-rollback
// and also be sure to the method is idempotent
func (uc *TransFrom) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("unlock the ￥30, because we lock the this in Prepare phase")
	return true, nil
}

func (uc *TransFrom) GetActionName() string {
	return "TransFrom"
}

type TransTo struct {
	repo AccountRepo
	log  *log.Helper
}

func NewTransTo(repo AccountRepo, logger log.Logger) *TransTo {
	return &TransTo{repo: repo, log: log.NewHelper(logger)}
}

func (t *TransTo) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("check if the account is legal")
	return true, nil
}

func (t *TransTo) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("transfer money to the account")
	return true, nil
}

func (t *TransTo) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("do nothing")
	return true, nil
}

func (t *TransTo) GetActionName() string {
	return "TransTo"
}
