package bloodlabnet

import (
	"time"
)

type TimingConfiguration struct {
	Timeout             time.Duration
	Deadline            time.Duration
	FlushBufferTimoutMs int
	PollInterval        time.Duration
}

type SecureConnectionOptions struct {
	PublicKey string
}

var DefaultTCPServerSettings = TimingConfiguration{
	Timeout:             time.Second * 3,
	Deadline:            time.Millisecond * 200,
	FlushBufferTimoutMs: 500,
	PollInterval:        time.Second * 60,
}

var DefaultFTPClientTimings = TimingConfiguration{
	Timeout:      time.Second * 5,
	PollInterval: time.Second * 60,
}

type ProxyType int

const (
	NoLoadBalancer     ProxyType = 1
	HAProxySendProxyV2 ProxyType = 2
)

type FileNameGeneration int

const (
	Default   FileNameGeneration = 1
	TimeStamp FileNameGeneration = 1
)

type ReadFilePolicy int

const (
	ReadAndLeaveFile ReadFilePolicy = 1
	DeleteWhenRead   ReadFilePolicy = 2
	RenameWhenRead   ReadFilePolicy = 3
)

type ErrorType int

const (
	ErrorConnect         ErrorType = 1
	ErrorSend            ErrorType = 2
	ErrorReceive         ErrorType = 3
	ErrorDisconnect      ErrorType = 4
	ErrorInternal        ErrorType = 5
	ErrorConnectionLimit ErrorType = 6
	ErrorAccept          ErrorType = 7 // error for Server
	ErrorMaxConnections  ErrorType = 8
	ErrorCreateSession   ErrorType = 9 // server only
)
