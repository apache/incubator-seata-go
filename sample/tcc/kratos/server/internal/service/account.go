package service

import (
	"context"

	"github.com/seata/seata-go/pkg/rm/tcc"

	pb "server/api/account/v1"
	"server/internal/biz"
)

type AccountService struct {
	pb.UnimplementedAccountServer
	transTo   *biz.TransTo
	transFrom *biz.TransFrom
}

func NewAccountService(tt *biz.TransTo, tf *biz.TransFrom) *AccountService {
	return &AccountService{
		transTo:   tt,
		transFrom: tf,
	}
}

func (s *AccountService) TransFrom(ctx context.Context, req *pb.AccountRequest) (*pb.AccountReply, error) {
	proxy, err := tcc.NewTCCServiceProxy(s.transFrom)
	if err != nil {
		return nil, err
	}
	if _, err := proxy.Prepare(ctx, nil); err != nil {
		return nil, err
	}
	return &pb.AccountReply{Message: "success"}, nil
}

func (s *AccountService) TransTo(ctx context.Context, req *pb.AccountRequest) (*pb.AccountReply, error) {
	proxy, err := tcc.NewTCCServiceProxy(s.transTo)
	if err != nil {
		return nil, err
	}
	if _, err := proxy.Prepare(ctx, nil); err != nil {
		return nil, err
	}
	return &pb.AccountReply{Message: "success"}, nil
}
