package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	mockholder "github.com/opentrx/seata-golang/v2/pkg/tc/holder/mock"
	mocklock "github.com/opentrx/seata-golang/v2/pkg/tc/lock/mock"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	mockserver "github.com/opentrx/seata-golang/v2/pkg/tc/server/mock"
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
			name: "test GetStatus success",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

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
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

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

func TestTransactionCoordinator_BranchReport(t *testing.T) {
	xid := "localhost:123"
	branchID := int64(1)
	updateBranchSessionErr := fmt.Errorf("test error")
	tests := []struct {
		name                   string
		transactionCoordinator func(ctrl *gomock.Controller) *TransactionCoordinator
		ctx                    context.Context
		request                *apis.BranchReportRequest
		expectedResult         *apis.BranchReportResponse
		expectedErr            error
	}{
		{
			name: "test BranchReport with empty XID",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedSessionHolder.EXPECT().FindGlobalTransaction("").Return(nil)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchReportRequest{
				XID: "",
			},
			expectedResult: &apis.BranchReportResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.GlobalTransactionNotExist,
				Message:       fmt.Sprintf("could not find global transaction xid = %s", ""),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchReport with empty Branch",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedBranchSession := &apis.BranchSession{
					BranchID: branchID + 1,
				}
				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID: xid,
					},
					BranchSessions: map[*apis.BranchSession]bool{
						mockedBranchSession: true,
					},
				}
				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchReportRequest{
				XID:      xid,
				BranchID: branchID,
			},
			expectedResult: &apis.BranchReportResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.BranchTransactionNotExist,
				Message:       fmt.Sprintf("could not find branch session xid = %s branchID = %d", xid, branchID),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchReport error update branch status ",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedBranchSession := &apis.BranchSession{
					BranchID: branchID,
				}
				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID: xid,
					},
					BranchSessions: map[*apis.BranchSession]bool{
						mockedBranchSession: true,
					},
				}

				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)
				mockedSessionHolder.EXPECT().UpdateBranchSessionStatus(mockedBranchSession, apis.Registered).Return(updateBranchSessionErr)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchReportRequest{
				XID:          xid,
				BranchID:     branchID,
				BranchStatus: apis.Registered,
			},
			expectedResult: &apis.BranchReportResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.BranchReportFailed,
				Message:       fmt.Sprintf("branch report failed, xid = %s, branchID = %d, err: %s", xid, branchID, updateBranchSessionErr.Error()),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchReport success",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedBranchSession := &apis.BranchSession{
					BranchID: branchID,
				}
				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID: xid,
					},
					BranchSessions: map[*apis.BranchSession]bool{
						mockedBranchSession: true,
					},
				}

				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)
				mockedSessionHolder.EXPECT().UpdateBranchSessionStatus(mockedBranchSession, apis.Registered).Return(nil)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchReportRequest{
				XID:          xid,
				BranchID:     branchID,
				BranchStatus: apis.Registered,
			},
			expectedResult: &apis.BranchReportResponse{
				ResultCode: apis.ResultCodeSuccess,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc := tt.transactionCoordinator(ctrl)

			actualResp, actualErr := tc.BranchReport(tt.ctx, tt.request)
			assert.Equal(t, tt.expectedResult, actualResp)
			assert.Equal(t, tt.expectedErr, actualErr)
		})
	}
}

func TestTransactionCoordinator_LockQuery(t *testing.T) {
	xid := "localhost:123"
	resourceID := "test"
	lockKey := "test_table:pk1,pk2"
	tests := []struct {
		name                   string
		transactionCoordinator func(ctrl *gomock.Controller) *TransactionCoordinator
		ctx                    context.Context
		request                *apis.GlobalLockQueryRequest
		expectedResult         *apis.GlobalLockQueryResponse
		expectedErr            error
	}{
		{
			name: "test LockQuery success",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedLockManager := mocklock.NewMockLockManagerInterface(ctrl)

				mockedLockManager.EXPECT().IsLockable(xid, resourceID, lockKey).Return(true)

				transactionCoordinator.resourceDataLocker = mockedLockManager

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.GlobalLockQueryRequest{
				XID:        xid,
				ResourceID: resourceID,
				LockKey:    lockKey,
			},
			expectedResult: &apis.GlobalLockQueryResponse{
				ResultCode: apis.ResultCodeSuccess,
				Lockable:   true,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc := tt.transactionCoordinator(ctrl)

			actualResp, actualErr := tc.LockQuery(tt.ctx, tt.request)
			assert.Equal(t, tt.expectedResult, actualResp)
			assert.Equal(t, tt.expectedErr, actualErr)
		})
	}
}

func TestTransactionCoordinator_Begin(t *testing.T) {
	addGlobalSessionHolderErr := fmt.Errorf("error add global session")
	tests := []struct {
		name                   string
		transactionCoordinator func(ctrl *gomock.Controller) *TransactionCoordinator
		ctx                    context.Context
		request                *apis.GlobalBeginRequest
		expectedResult         *apis.GlobalBeginResponse
		expectedErr            error
	}{
		{
			name: "test Begin error add global session",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedSessionHolder.EXPECT().AddGlobalSession(gomock.Any()).Return(addGlobalSessionHolderErr)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.GlobalBeginRequest{
				Addressing:      "localhost",
				TransactionName: "TestTransaction",
				Timeout:         int32(300),
			},
			expectedResult: &apis.GlobalBeginResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.BeginFailed,
				Message:       addGlobalSessionHolderErr.Error(),
			},
			expectedErr: nil,
		},
		{
			name: "test Begin success",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedSessionHolder.EXPECT().AddGlobalSession(gomock.Any()).Return(nil)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.GlobalBeginRequest{
				Addressing:      "localhost",
				TransactionName: "TestTransaction",
				Timeout:         int32(300),
			},
			expectedResult: &apis.GlobalBeginResponse{
				ResultCode: apis.ResultCodeSuccess,
				//XID:        xid,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc := tt.transactionCoordinator(ctrl)

			actualResp, actualErr := tc.Begin(tt.ctx, tt.request)
			if tt.name == "test Begin success" {
				// in case XID unable to mock
				assert.Equal(t, tt.expectedResult.ResultCode, actualResp.ResultCode)
				assert.Equal(t, tt.expectedErr, actualErr)
			} else {
				assert.Equal(t, tt.expectedResult, actualResp)
				assert.Equal(t, tt.expectedErr, actualErr)
			}
		})
	}
}

func TestTransactionCoordinator_BranchRegister(t *testing.T) {
	xid := "localhost:123"
	resourceID := "test_DB"
	lockKey := "test_table:pk1,pk2"
	addBranchSessionErr := fmt.Errorf("error add branch session")
	tests := []struct {
		name                   string
		transactionCoordinator func(ctrl *gomock.Controller) *TransactionCoordinator
		ctx                    context.Context
		request                *apis.BranchRegisterRequest
		expectedResult         *apis.BranchRegisterResponse
		expectedErr            error
	}{
		{
			name: "test BranchRegister with empty XID",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedSessionHolder.
					EXPECT().
					FindGlobalTransaction("").
					Return(nil)

				transactionCoordinator.holder = mockedSessionHolder

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID: "",
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.GlobalTransactionNotExist,
				Message:       fmt.Sprintf("could not find global transaction xid = %s", ""),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister error try lock",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
					},
					BranchSessions: nil,
				}
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedSessionHolder.
					EXPECT().
					FindGlobalTransaction(xid).
					Return(mockedGlobalTransaction)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(false, fmt.Errorf("error try lock"))

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID: xid,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.FailedLockGlobalTransaction,
				Message:       fmt.Sprintf("could not lock global transaction xid = %s", xid),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister error global transaction is not active",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
						Status:  apis.Begin,
						Active:  false,
					},
					BranchSessions: nil,
				}
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedSessionHolder.
					EXPECT().
					FindGlobalTransaction(xid).
					Return(mockedGlobalTransaction)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(true, nil)
				mockedGlobalSessionLock.EXPECT().Unlock(mockedGlobalTransaction.GlobalSession).Return()

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID: xid,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.GlobalTransactionNotActive,
				Message:       fmt.Sprintf("could not register branch into global session xid = %s status = %d", xid, apis.Begin),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister error global transaction is not begin status",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
						Status:  apis.Committed,
						Active:  true,
					},
					BranchSessions: nil,
				}
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedSessionHolder.
					EXPECT().
					FindGlobalTransaction(xid).
					Return(mockedGlobalTransaction)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(true, nil)

				mockedGlobalSessionLock.EXPECT().Unlock(mockedGlobalTransaction.GlobalSession).Return()

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID: xid,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.GlobalTransactionStatusInvalid,
				Message: fmt.Sprintf("could not register branch into global session xid = %s status = %d while expecting %d",
					xid, apis.Committed, apis.Begin),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister error acquire resource data lock",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
						Status:  apis.Begin,
						Active:  true,
					},
					BranchSessions: nil,
				}
				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(true, nil)
				mockedGlobalSessionLock.EXPECT().Unlock(mockedGlobalTransaction.GlobalSession).Return()

				mockedResourceDataLock := mocklock.NewMockLockManagerInterface(ctrl)
				mockedResourceDataLock.EXPECT().AcquireLock(gomock.Any()).Return(false)

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock
				transactionCoordinator.resourceDataLocker = mockedResourceDataLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID:        xid,
				ResourceID: resourceID,
				LockKey:    lockKey,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.LockKeyConflict,
				Message: fmt.Sprintf("branch lock acquire failed xid = %s resourceId = %s, lockKey = %s",
					xid, resourceID, lockKey),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister error acquire resource data lock",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
						Status:  apis.Begin,
						Active:  true,
					},
					BranchSessions: nil,
				}
				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(true, nil)
				mockedGlobalSessionLock.EXPECT().Unlock(mockedGlobalTransaction.GlobalSession).Return()

				mockedResourceDataLock := mocklock.NewMockLockManagerInterface(ctrl)
				mockedResourceDataLock.EXPECT().AcquireLock(gomock.Any()).Return(false)

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock
				transactionCoordinator.resourceDataLocker = mockedResourceDataLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID:        xid,
				ResourceID: resourceID,
				LockKey:    lockKey,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.LockKeyConflict,
				Message: fmt.Sprintf("branch lock acquire failed xid = %s resourceId = %s, lockKey = %s",
					xid, resourceID, lockKey),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister error add branch session",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
						Status:  apis.Begin,
						Active:  true,
					},
					BranchSessions: nil,
				}
				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)
				mockedSessionHolder.EXPECT().AddBranchSession(mockedGlobalTransaction.GlobalSession, gomock.Any()).Return(addBranchSessionErr)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(true, nil)
				mockedGlobalSessionLock.EXPECT().Unlock(mockedGlobalTransaction.GlobalSession).Return()

				mockedResourceDataLock := mocklock.NewMockLockManagerInterface(ctrl)
				mockedResourceDataLock.EXPECT().AcquireLock(gomock.Any()).Return(true)

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock
				transactionCoordinator.resourceDataLocker = mockedResourceDataLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID:        xid,
				ResourceID: resourceID,
				LockKey:    lockKey,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode:    apis.ResultCodeFailed,
				ExceptionCode: apis.BranchRegisterFailed,
				//Message:       fmt.Sprintf("branch register failed, xid = %s, branchID = %d, err: %s",xid, branchID, addBranchSessionErr.Error()),
			},
			expectedErr: nil,
		},
		{
			name: "test BranchRegister success",
			transactionCoordinator: func(ctrl *gomock.Controller) *TransactionCoordinator {
				transactionCoordinator := &TransactionCoordinator{}
				mockedSessionHolder := mockholder.NewMockSessionHolderInterface(ctrl)
				mockedGlobalSessionLock := mockserver.NewMockGlobalSessionLocker(ctrl)

				mockedGlobalTransaction := &model.GlobalTransaction{
					GlobalSession: &apis.GlobalSession{
						XID:     xid,
						Timeout: int32(300),
						Status:  apis.Begin,
						Active:  true,
					},
					BranchSessions: nil,
				}
				mockedSessionHolder.EXPECT().FindGlobalTransaction(xid).Return(mockedGlobalTransaction)
				mockedSessionHolder.EXPECT().AddBranchSession(mockedGlobalTransaction.GlobalSession, gomock.Any()).Return(nil)

				mockedGlobalSessionLock.EXPECT().TryLock(
					mockedGlobalTransaction.GlobalSession,
					time.Duration(mockedGlobalTransaction.Timeout)*time.Millisecond,
				).Return(true, nil)
				mockedGlobalSessionLock.EXPECT().Unlock(mockedGlobalTransaction.GlobalSession).Return()

				mockedResourceDataLock := mocklock.NewMockLockManagerInterface(ctrl)
				mockedResourceDataLock.EXPECT().AcquireLock(gomock.Any()).Return(true)

				transactionCoordinator.holder = mockedSessionHolder
				transactionCoordinator.locker = mockedGlobalSessionLock
				transactionCoordinator.resourceDataLocker = mockedResourceDataLock

				return transactionCoordinator
			},
			ctx: nil,
			request: &apis.BranchRegisterRequest{
				XID:        xid,
				ResourceID: resourceID,
				LockKey:    lockKey,
			},
			expectedResult: &apis.BranchRegisterResponse{
				ResultCode: apis.ResultCodeSuccess,
				//BranchID:   branchID,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc := tt.transactionCoordinator(ctrl)

			actualResp, actualErr := tc.BranchRegister(tt.ctx, tt.request)

			switch tt.name {
			case "test BranchRegister error add branch session":
				// in case branchID unable to mock
				assert.Equal(t, tt.expectedResult.ResultCode, actualResp.ResultCode)
				assert.Equal(t, tt.expectedResult.ExceptionCode, actualResp.ExceptionCode)
				assert.Equal(t, tt.expectedErr, actualErr)
			case "test BranchRegister success":
				// in case branchID unable to mock
				assert.Equal(t, tt.expectedResult.ResultCode, actualResp.ResultCode)
				assert.Equal(t, tt.expectedErr, actualErr)
			default:
				assert.Equal(t, tt.expectedResult, actualResp)
				assert.Equal(t, tt.expectedErr, actualErr)
			}
		})
	}
}

func TestTransactionCoordinator_Commit(t *testing.T) {
	// todo
}

func TestTransactionCoordinator_Rollback(t *testing.T) {
	// todo
}
