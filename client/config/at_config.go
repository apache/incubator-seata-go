package config

type ATConfig struct {
	DSN                 string `yaml:"dsn" json:"dsn,omitempty"`
	ReportRetryCount    int  `default:"5" yaml:"report_retry_count" json:"report_retry_count,omitempty"`
	ReportSuccessEnable bool `default:"false" yaml:"report_success_enable" json:"report_success_enable,omitempty"`
}