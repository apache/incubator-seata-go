package model

import "time"

type TCCFenceDO struct {
	/**
	 * the global transaction id
	 */
	Xid string

	/**
	 * the branch transaction id
	 */
	BranchId int64

	/**
	 * the action name
	 */
	ActionName string

	/**
	 * the tcc fence status
	 * tried: 1; committed: 2; rollbacked: 3; suspended: 4
	 */
	Status int32

	/**
	 * create time
	 */
	GmtCreate time.Time

	/**
	 * update time
	 */
	GmtModified time.Time
}
