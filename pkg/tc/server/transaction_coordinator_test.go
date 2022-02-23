package server

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	hmock "github.com/opentrx/seata-golang/v2/pkg/tc/holder/mock"
	"github.com/stretchr/testify/assert"
)

func TestTransactionCoordinator_GetStatus(t *testing.T) {
	xid := "localhost:123"
	tests := []struct {
		name                   string
		transactionCoordinator func(ctrl *gomock.Controller) *TransactionCoordinator
		ctx                    context.Context
		request                *apis.GlobalStatusRequest
		expectedResult         *apis.GlobalStatusResponse
		expectedErr            error
	}{
		{
			name: "test GetStatus with existing XID",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := hmock.NewMockSessionHolderInterface(ctrl)

				mockedSessionHolder.
					EXPECT().
					FindGlobalSession(xid).
					Return(&apis.GlobalSession{
						Status: apis.Begin,
					})

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.GlobalStatusRequest{
				XID: "localhost:123",
			},
			expectedResult: &apis.GlobalStatusResponse{
				ResultCode:   apis.ResultCodeSuccess,
				GlobalStatus: apis.Begin,
			},
			expectedErr: nil,
		},
		{
			name: "test GetStatus with empty XID",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := hmock.NewMockSessionHolderInterface(ctrl)

				mockedSessionHolder.
					EXPECT().
					FindGlobalSession("").
					Return(&apis.GlobalSession{
						Status: apis.Finished,
					})

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.GlobalStatusRequest{
				XID: "",
			},
			expectedResult: &apis.GlobalStatusResponse{
				ResultCode:   apis.ResultCodeSuccess,
				GlobalStatus: apis.Finished,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc := tt.transactionCoordinator(ctrl)

			actualResp, actualErr := tc.GetStatus(tt.ctx, tt.request)
			assert.Equal(t, tt.expectedResult, actualResp)
			assert.Equal(t, tt.expectedErr, actualErr)
		})
	}
}
