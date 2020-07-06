package holder

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

import (
	"vimagination.zapto.org/byteio"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/time"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

var FileTrxNum int64 = 0
var PerFileBlockSize int64 = 65535 * 8
var HisDataFilenamePostfix = ".1"
var MaxTrxTimeoutMills int64 = 30 * 60 * 1000

type ReloadableStore interface {
	/**
	 * Read write holder.
	 *
	 * @param readSize  the read size
	 * @param isHistory the is history
	 * @return the list
	 */
	ReadWriteStore(readSize int, isHistory bool) []*TransactionWriteStore

	/**
	 * Has remaining boolean.
	 *
	 * @param isHistory the is history
	 * @return the boolean
	 */
	HasRemaining(isHistory bool) bool
}

type FileTransactionStoreManager struct {
	SessionManager SessionManager

	currFullFileName  string
	hisFullFileName   string
	currFileChannel   *os.File
	LastModifiedTime  int64
	TrxStartTimeMills int64
	sync.Mutex

	recoverHisOffset  int64
	recoverCurrOffset int64
}

func (storeManager *FileTransactionStoreManager) InitFile(fullFileName string) {
	storeManager.currFullFileName = fullFileName
	storeManager.hisFullFileName = fullFileName + HisDataFilenamePostfix
	storeManager.currFileChannel, _ = os.OpenFile(fullFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	storeManager.TrxStartTimeMills = int64(time.CurrentTimeMillis())
}

func (storeManager *FileTransactionStoreManager) writeDataFrame(data []byte) {
	dataLength := uint32(len(data))
	dataLengthBytes := [4]byte{
		byte(dataLength >> 24),
		byte(dataLength >> 16),
		byte(dataLength >> 8),
		byte(dataLength),
	}
	storeManager.currFileChannel.Write(dataLengthBytes[:4])
	storeManager.currFileChannel.Write(data)
}

func (storeManager *FileTransactionStoreManager) WriteSession(logOperation LogOperation, session session.SessionStorable) bool {
	storeManager.Lock()
	defer storeManager.Unlock()
	var curFileTrxNum int64 = 0
	data, err := (&TransactionWriteStore{
		SessionRequest: session,
		LogOperation:   logOperation,
	}).Encode()
	if err != nil {
		logging.Logger.Info(err.Error())
		return false
	}
	storeManager.writeDataFrame(data)
	storeManager.LastModifiedTime = int64(time.CurrentTimeMillis())
	curFileTrxNum = atomic.AddInt64(&FileTrxNum, 1)
	if curFileTrxNum%PerFileBlockSize == 0 &&
		int64(time.CurrentTimeMillis())-storeManager.TrxStartTimeMills > MaxTrxTimeoutMills {
		storeManager.saveHistory()
	}
	return true
}

func (storeManager *FileTransactionStoreManager) ReadSession(xid string) *session.GlobalSession {
	return nil
}

func (storeManager *FileTransactionStoreManager) ReadSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession {
	return nil
}

func (storeManager *FileTransactionStoreManager) ReadSessionWithSessionCondition(sessionCondition model.SessionCondition) []*session.GlobalSession {
	return nil
}

func (storeManager *FileTransactionStoreManager) Shutdown() {
	storeManager.currFileChannel.Close()
}

func (storeManager *FileTransactionStoreManager) GetCurrentMaxSessionId() int64 {
	return int64(0)
}

func (storeManager *FileTransactionStoreManager) saveHistory() {
	storeManager.findTimeoutAndSave()
	os.Rename(storeManager.currFullFileName, storeManager.hisFullFileName)
	storeManager.InitFile(storeManager.currFullFileName)
}

func (storeManager *FileTransactionStoreManager) findTimeoutAndSave() (bool, error) {
	globalSessionsOverMaxTimeout := storeManager.SessionManager.FindGlobalSessions(model.SessionCondition{OverTimeAliveMills: MaxTrxTimeoutMills})

	if globalSessionsOverMaxTimeout == nil {
		return true, nil
	}

	for _, globalSession := range globalSessionsOverMaxTimeout {
		globalWriteStore := &TransactionWriteStore{
			SessionRequest: globalSession,
			LogOperation:   LogOperationGlobalAdd,
		}
		data, err := globalWriteStore.Encode()
		if err != nil {
			return false, err
		}
		storeManager.writeDataFrame(data)

		branchSessIonsOverMaXTimeout := globalSession.GetSortedBranches()
		if len(branchSessIonsOverMaXTimeout) > 0 {
			for _, branchSession := range branchSessIonsOverMaXTimeout {
				branchWriteStore := &TransactionWriteStore{
					SessionRequest: branchSession,
					LogOperation:   LogOperationBranchAdd,
				}
				data, err := branchWriteStore.Encode()
				if err != nil {
					return false, err
				}
				storeManager.writeDataFrame(data)
			}
		}
	}
	return true, nil
}

func (storeManager *FileTransactionStoreManager) ReadWriteStore(readSize int, isHistory bool) []*TransactionWriteStore {
	var (
		file          *os.File
		currentOffset int64
	)
	if isHistory {
		file, _ = os.OpenFile(storeManager.hisFullFileName, os.O_RDWR|os.O_CREATE, 0777)
		currentOffset = storeManager.recoverHisOffset
	} else {
		file, _ = os.OpenFile(storeManager.currFullFileName, os.O_RDWR|os.O_CREATE, 0777)
		currentOffset = storeManager.recoverCurrOffset
	}

	return storeManager.parseDataFile(file, readSize, currentOffset)
}

func (storeManager *FileTransactionStoreManager) HasRemaining(isHistory bool) bool {
	var (
		file          *os.File
		currentOffset int64
	)
	if isHistory {
		file, _ = os.OpenFile(storeManager.hisFullFileName, os.O_RDWR|os.O_CREATE, 0777)
		currentOffset = storeManager.recoverHisOffset
	} else {
		file, _ = os.OpenFile(storeManager.currFullFileName, os.O_RDWR|os.O_CREATE, 0777)
		currentOffset = storeManager.recoverCurrOffset
	}
	defer file.Close()

	fi, _ := file.Stat()
	return currentOffset < fi.Size()
}

func (storeManager *FileTransactionStoreManager) parseDataFile(file *os.File, readSize int, currentOffset int64) []*TransactionWriteStore {
	defer file.Close()
	var result = make([]*TransactionWriteStore, 0)
	fi, _ := file.Stat()
	fileSize := fi.Size()
	reader := byteio.BigEndianReader{Reader: file}
	offset := currentOffset
	for offset < fileSize {
		file.Seek(offset, 0)
		dataLength, n, _ := reader.ReadUint32()
		if n < 4 {
			break
		}

		data := make([]byte, int(dataLength))
		length, _ := reader.Read(data)
		offset += int64(length + 4)

		if length == int(dataLength) {
			st := &TransactionWriteStore{}
			st.Decode(data)
			result = append(result, st)
			if len(result) == readSize {
				break
			}
		} else if length == 0 {
			break
		}
	}
	if isHisFile(fi.Name()) {
		storeManager.recoverHisOffset = offset
	} else {
		storeManager.recoverCurrOffset = offset
	}
	return result
}

func isHisFile(path string) bool {
	return strings.HasSuffix(path, HisDataFilenamePostfix)
}
