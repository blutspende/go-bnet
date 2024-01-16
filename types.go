package bloodlabnet

import (
	"time"
)

type TCPClientConfiguration struct {
	Timeout                 time.Duration
	Deadline                time.Duration
	FlushBufferTimoutMs     int
	PollInterval            time.Duration
	SessionAfterFirstByte   bool
	SessionInitationTimeout time.Duration
	SourceIP                string
}

func (s TCPClientConfiguration) SetSourceIP(sourceIP string) TCPClientConfiguration {
	s.SourceIP = sourceIP
	return s
}

var DefaultTCPClientSettings = TCPClientConfiguration{
	Timeout:                 time.Second * 3,
	Deadline:                time.Millisecond * 200,
	FlushBufferTimoutMs:     500,
	PollInterval:            time.Second * 60,
	SessionAfterFirstByte:   true,            // Sessions are initiated after reading the first bytes (avoids disconnects)
	SessionInitationTimeout: time.Second * 0, // Waiting forever by default
	SourceIP:                "",
}

type TCPServerConfiguration struct {
	Timeout                 time.Duration
	Deadline                time.Duration
	FlushBufferTimoutMs     int
	PollInterval            time.Duration
	SessionAfterFirstByte   bool
	SessionInitationTimeout time.Duration
}

type SecureConnectionOptions struct {
	PublicKey string
}

var DefaultTCPServerSettings = TCPServerConfiguration{
	Timeout:                 time.Second * 3,
	Deadline:                time.Millisecond * 200,
	FlushBufferTimoutMs:     500,
	PollInterval:            time.Second * 60,
	SessionAfterFirstByte:   true,            // Sessions are initiated after reading the first bytes (avoids disconnects)
	SessionInitationTimeout: time.Second * 0, // Waiting forever by default
}

type ConnectionType int

const (
	NoLoadBalancer     ConnectionType = 1
	HAProxySendProxyV2 ConnectionType = 2
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
	ErrorCreateSession   ErrorType = 9  // server only
	ErrorConfiguration   ErrorType = 10 // Error in configuration
	ErrorLogin           ErrorType = 11
)
