package config

type UndoConfig struct {
	LogSaveDays int16 `default:"7" yaml:"log_save_days" json:"log_save_days,omitempty"`
}
