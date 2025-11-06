package config

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSessionConfig_RegisterFlagsWithPrefix_Defaults(t *testing.T) {
	cfg := &SessionConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("session", fs)

	// Test default values
	assert.False(t, cfg.CompressEncoding, "Default CompressEncoding should be false")
	assert.True(t, cfg.TCPNoDelay, "Default TCPNoDelay should be true")
	assert.True(t, cfg.TCPKeepAlive, "Default TCPKeepAlive should be true")
	assert.Equal(t, 3*time.Minute, cfg.KeepAlivePeriod, "Default KeepAlivePeriod should be 3 minutes")
	assert.Equal(t, 262144, cfg.TCPRBufSize, "Default TCPRBufSize should be 262144")
	assert.Equal(t, 65536, cfg.TCPWBufSize, "Default TCPWBufSize should be 65536")
	assert.Equal(t, time.Second, cfg.TCPReadTimeout, "Default TCPReadTimeout should be 1 second")
	assert.Equal(t, 5*time.Second, cfg.TCPWriteTimeout, "Default TCPWriteTimeout should be 5 seconds")
	assert.Equal(t, time.Second, cfg.WaitTimeout, "Default WaitTimeout should be 1 second")
	assert.Equal(t, 102400, cfg.MaxMsgLen, "Default MaxMsgLen should be 102400")
	assert.Equal(t, "client", cfg.SessionName, "Default SessionName should be client")
	assert.Equal(t, time.Second, cfg.CronPeriod, "Default CronPeriod should be 1 second")
}

func TestSessionConfig_RegisterFlagsWithPrefix_CustomValues(t *testing.T) {
	cfg := &SessionConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("session", fs)

	args := []string{
		"-session.compress-encoding=true",
		"-session.tcp-no-delay=false",
		"-session.tcp-keep-alive=false",
		"-session.keep-alive-period=5m",
		"-session.tcp-r-buf-size=524288",
		"-session.tcp-w-buf-size=131072",
		"-session.tcp-read-timeout=2s",
		"-session.tcp-write-timeout=10s",
		"-session.wait-timeout=3s",
		"-session.max-msg-len=204800",
		"-session.session-name=test-session",
		"-session.cron-period=2s",
	}
	err := fs.Parse(args)
	assert.NoError(t, err, "Flag parsing should not return an error")

	// Verify custom values
	assert.True(t, cfg.CompressEncoding, "CompressEncoding should be true")
	assert.False(t, cfg.TCPNoDelay, "TCPNoDelay should be false")
	assert.False(t, cfg.TCPKeepAlive, "TCPKeepAlive should be false")
	assert.Equal(t, 5*time.Minute, cfg.KeepAlivePeriod, "KeepAlivePeriod should be 5 minutes")
	assert.Equal(t, 524288, cfg.TCPRBufSize, "TCPRBufSize should be 524288")
	assert.Equal(t, 131072, cfg.TCPWBufSize, "TCPWBufSize should be 131072")
	assert.Equal(t, 2*time.Second, cfg.TCPReadTimeout, "TCPReadTimeout should be 2 seconds")
	assert.Equal(t, 10*time.Second, cfg.TCPWriteTimeout, "TCPWriteTimeout should be 10 seconds")
	assert.Equal(t, 3*time.Second, cfg.WaitTimeout, "WaitTimeout should be 3 seconds")
	assert.Equal(t, 204800, cfg.MaxMsgLen, "MaxMsgLen should be 204800")
	assert.Equal(t, "test-session", cfg.SessionName, "SessionName should be test-session")
	assert.Equal(t, 2*time.Second, cfg.CronPeriod, "CronPeriod should be 2 seconds")
}

func TestSessionConfig_RegisterFlagsWithPrefix_BooleanFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "All boolean flags true",
			args: []string{
				"-session.compress-encoding=true",
				"-session.tcp-no-delay=true",
				"-session.tcp-keep-alive=true",
			},
			expected: SessionConfig{
				CompressEncoding: true,
				TCPNoDelay:       true,
				TCPKeepAlive:     true,
			},
		},
		{
			name: "All boolean flags false",
			args: []string{
				"-session.compress-encoding=false",
				"-session.tcp-no-delay=false",
				"-session.tcp-keep-alive=false",
			},
			expected: SessionConfig{
				CompressEncoding: false,
				TCPNoDelay:       false,
				TCPKeepAlive:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)

			err := fs.Parse(tt.args)
			assert.NoError(t, err)

			assert.Equal(t, tt.expected.CompressEncoding, cfg.CompressEncoding)
			assert.Equal(t, tt.expected.TCPNoDelay, cfg.TCPNoDelay)
			assert.Equal(t, tt.expected.TCPKeepAlive, cfg.TCPKeepAlive)
		})
	}
}

func TestSessionConfig_RegisterFlagsWithPrefix_DurationValues(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "Millisecond durations",
			args: []string{
				"-session.keep-alive-period=500ms",
				"-session.tcp-read-timeout=100ms",
				"-session.tcp-write-timeout=200ms",
				"-session.wait-timeout=50ms",
				"-session.cron-period=300ms",
			},
			expected: SessionConfig{
				KeepAlivePeriod: 500 * time.Millisecond,
				TCPReadTimeout:  100 * time.Millisecond,
				TCPWriteTimeout: 200 * time.Millisecond,
				WaitTimeout:     50 * time.Millisecond,
				CronPeriod:      300 * time.Millisecond,
			},
		},
		{
			name: "Minute durations",
			args: []string{
				"-session.keep-alive-period=10m",
				"-session.tcp-read-timeout=1m",
				"-session.tcp-write-timeout=2m",
				"-session.wait-timeout=30s",
				"-session.cron-period=45s",
			},
			expected: SessionConfig{
				KeepAlivePeriod: 10 * time.Minute,
				TCPReadTimeout:  time.Minute,
				TCPWriteTimeout: 2 * time.Minute,
				WaitTimeout:     30 * time.Second,
				CronPeriod:      45 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)

			err := fs.Parse(tt.args)
			assert.NoError(t, err)

			assert.Equal(t, tt.expected.KeepAlivePeriod, cfg.KeepAlivePeriod)
			assert.Equal(t, tt.expected.TCPReadTimeout, cfg.TCPReadTimeout)
			assert.Equal(t, tt.expected.TCPWriteTimeout, cfg.TCPWriteTimeout)
			assert.Equal(t, tt.expected.WaitTimeout, cfg.WaitTimeout)
			assert.Equal(t, tt.expected.CronPeriod, cfg.CronPeriod)
		})
	}
}

func TestSessionConfig_RegisterFlagsWithPrefix_IntegerValues(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "Small buffer sizes",
			args: []string{
				"-session.tcp-r-buf-size=1024",
				"-session.tcp-w-buf-size=512",
				"-session.max-msg-len=2048",
			},
			expected: SessionConfig{
				TCPRBufSize: 1024,
				TCPWBufSize: 512,
				MaxMsgLen:   2048,
			},
		},
		{
			name: "Large buffer sizes",
			args: []string{
				"-session.tcp-r-buf-size=1048576",
				"-session.tcp-w-buf-size=524288",
				"-session.max-msg-len=1024000",
			},
			expected: SessionConfig{
				TCPRBufSize: 1048576,
				TCPWBufSize: 524288,
				MaxMsgLen:   1024000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)

			err := fs.Parse(tt.args)
			assert.NoError(t, err)

			assert.Equal(t, tt.expected.TCPRBufSize, cfg.TCPRBufSize)
			assert.Equal(t, tt.expected.TCPWBufSize, cfg.TCPWBufSize)
			assert.Equal(t, tt.expected.MaxMsgLen, cfg.MaxMsgLen)
		})
	}
}
