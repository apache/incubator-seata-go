package config

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
)

const (
	DefaultFileDir                      = "root.data"
	DefaultMaxBranchSessionSize         = 1024 * 16
	DefaultMaxGlobalSessionSize         = 512
	DefaultWriteBufferSize              = 1024 * 16
	DefualtServiceSessionReloadReadSize = 100
)

type FlushDiskMode int

const (
	/**
	 * sync flush disk
	 */
	FlushdiskModeSyncModel FlushDiskMode = iota

	/**
	 * async flush disk
	 */
	FlushdiskModeAsyncModel
)

type StoreConfig struct {
	MaxBranchSessionSize int             `default:"16384" yaml:"max_branch_session_size" json:"max_branch_session_size,omitempty"`
	MaxGlobalSessionSize int             `default:"512" yaml:"max_global_session_size" json:"max_global_session_size,omitempty"`
	StoreMode            string          `default:"file" yaml:"mode" json:"mode,omitempty"`
	FileStoreConfig      FileStoreConfig `yaml:"file" json:"file,omitempty"`
	DBStoreConfig        DBStoreConfig   `yaml:"db" json:"db,omitempty"`
}

type FileStoreConfig struct {
	FileDir                  string        `default:"root.data" yaml:"file_dir" json:"file_dir,omitempty"`
	FileWriteBufferCacheSize int           `default:"16384" yaml:"file_write_buffer_cache_size" json:"file_write_buffer_cache_size,omitempty"`
	FlushDiskMode            FlushDiskMode `default:"1" yaml:"flush_disk_mode" json:"flush_disk_mode,omitempty"`
	SessionReloadReadSize    int           `default:"100" yaml:"session_reload_read_size" json:"session_reload_read_size,omitempty"`
}

type DBStoreConfig struct {
	LogQueryLimit int    `default:"100" yaml:"log_query_limit" json:"log_query_limit"`
	DSN           string `yaml:"dsn" json:"dsn"`
	Engine        *xorm.Engine
}

func GetDefaultFileStoreConfig() FileStoreConfig {
	return FileStoreConfig{
		FileDir:                  DefaultFileDir,
		FileWriteBufferCacheSize: DefaultWriteBufferSize,
		FlushDiskMode:            0,
		SessionReloadReadSize:    DefualtServiceSessionReloadReadSize,
	}
}
