package bloodlabnet

import (
	"time"
)

type TimingConfiguration struct {
	Timeout             time.Duration
	Deadline            time.Duration
	FlushBufferTimoutMs int
}

var DefaultTCPServerTimings = TimingConfiguration{
	Timeout:             time.Second * 3,
	Deadline:            time.Millisecond * 200,
	FlushBufferTimoutMs: 500,
}

type HighLevelProtocol int

const (
	PROTOCOL_RAW     HighLevelProtocol = 1
	PROTOCOL_STXETX  HighLevelProtocol = 2
	PROTOCLOL_LIS1A1 HighLevelProtocol = 3
)

type ProxyType int

const (
	NoLoadBalancer     ProxyType = 1
	HaProxySendProxyV2 ProxyType = 2
)

type FileNameGeneration int

const (
	Default   FileNameGeneration = 1
	TimeStamp FileNameGeneration = 1
)

type ReadFilePolicy int

const (
	Nothing ReadFilePolicy = 1
	Delete  ReadFilePolicy = 2
	Rename  ReadFilePolicy = 3
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
