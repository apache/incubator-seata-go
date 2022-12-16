package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"server/internal/biz"
)

type accountRepo struct {
	data *Data
	log  *log.Helper
}

// NewAccountRepo .
func NewAccountRepo(data *Data, logger log.Logger) biz.AccountRepo {
	return &accountRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *accountRepo) Save(ctx context.Context, g *biz.Account) (*biz.Account, error) {
	return g, nil
}

func (r *accountRepo) Update(ctx context.Context, g *biz.Account) (*biz.Account, error) {
	return g, nil
}

func (r *accountRepo) FindByID(context.Context, int64) (*biz.Account, error) {
	return nil, nil
}

func (r *accountRepo) ListByHello(context.Context, string) ([]*biz.Account, error) {
	return nil, nil
}

func (r *accountRepo) ListAll(context.Context) ([]*biz.Account, error) {
	return nil, nil
}
