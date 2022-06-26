package main

import (
	"time"
)

type TimingConfiguration struct {
	Timeout  time.Duration
	Deadline time.Duration
}

type HighLevelProtocol int

const (
	PROTOCOL_RAW     HighLevelProtocol = 1
	PROTOCOL_STXETX  HighLevelProtocol = 2
	PROTOCLOL_LIS1A1 HighLevelProtocol = 3
)

type ProxyType int

const (
	NoLoadbalancer     ProxyType = 1
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

var DefaultTCPTiming = TimingConfiguration{
	Timeout:  time.Second * 3,
	Deadline: time.Millisecond * 200,
}

type ErrorType int

const (
	ErrorConnect    ErrorType = 1
	ErrorSend       ErrorType = 2
	ErrorReceive    ErrorType = 3
	ErrorDisconnect ErrorType = 4
	ErrorInternal   ErrorType = 5
)
