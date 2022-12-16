package transport

import (
	"flag"
	"time"
)

type Shutdown struct {
	Wait time.Duration `yaml:"wait" json:"wait" konaf:"wait"`
}

type Config struct {
	ShutdownConfig                 ShutdownConfig `yaml:"shutdown" json:"shutdown" koanf:"shutdown"`
	Type                           string         `yaml:"type" json:"type" koanf:"type"`
	Server                         string         `yaml:"server" json:"server" koanf:"server"`
	Heartbeat                      bool           `yaml:"heartbeat" json:"heartbeat" koanf:"heartbeat"`
	Serialization                  string         `yaml:"serialization" json:"serialization" koanf:"serialization"`
	Compressor                     string         `yaml:"compressor" json:"compressor" koanf:"compressor"`
	EnableTmClientBatchSendRequest bool           `yaml:"enable-tm-client-batch-send-request" json:"enable-tm-client-batch-send-request" koanf:"enable-tm-client-batch-send-request"`
	EnableRmClientBatchSendRequest bool           `yaml:"enable-rm-client-batch-send-request" json:"enable-rm-client-batch-send-request" koanf:"enable-rm-client-batch-send-request"`
	RPCRmRequestTimeout            time.Duration  `yaml:"rpc-rm-request-timeout" json:"rpc-rm-request-timeout" koanf:"rpc-rm-request-timeout"`
	RPCTmRequestTimeout            time.Duration  `yaml:"rpc-tm-request-timeout" json:"rpc-tm-request-timeout" koanf:"rpc-tm-request-timeout"`
}

// RegisterFlagsWithPrefix for Config.
func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	cfg.ShutdownConfig.RegisterFlagsWithPrefix(prefix+".shutdown", f)
	f.StringVar(&cfg.Type, prefix+".type", "TCP", "Transport protocol type.")
	f.StringVar(&cfg.Server, prefix+".server", "NIO", "Server type.")
	f.BoolVar(&cfg.Heartbeat, prefix+".heartbeat", true, "Heartbeat.")
	f.StringVar(&cfg.Serialization, prefix+".serialization", "seata", "Encoding and decoding mode.")
	f.StringVar(&cfg.Compressor, prefix+".compressor", "none", "Message compression mode.")
	f.BoolVar(&cfg.EnableTmClientBatchSendRequest, prefix+".enable-tm-client-batch-send-request", false, "Allow batch sending of requests (TM).")
	f.BoolVar(&cfg.EnableRmClientBatchSendRequest, prefix+".enable-rm-client-batch-send-request", true, "Allow batch sending of requests (RM).")
	f.DurationVar(&cfg.RPCRmRequestTimeout, prefix+".rpc-rm-request-timeout", 3*time.Second, "RM send request timeout.")
	f.DurationVar(&cfg.RPCTmRequestTimeout, prefix+".rpc-tm-request-timeout", 3*time.Second, "TM send request timeout.")
}
