package protocol

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/blutspende/go-bloodlab-net/protocol/utilities"
)

type LogType string

const LogTypeInfo LogType = "info"
const LogTypeSend LogType = "send"
const LogTypeRecv LogType = "recv"
const LogTypeFail LogType = "fail"
const LogTypeClos LogType = "clos"

type LogAdapter interface {
	logMessage(timestamp time.Time, ltype LogType, msg string, payload []byte)
}
type protocolLogger struct {
	enableLog  bool
	protocol   Implementation
	logAdapter LogAdapter
}

func (pl *protocolLogger) Interrupt() {
	if pl.logAdapter != nil {
		pl.logAdapter.logMessage(time.Now(), LogTypeInfo, "interrupted connection", []byte{})
	}
	fmt.Printf("PL|%s| info - interrupt connection\n", time.Now().Format("20060102 150405.0"))
	pl.protocol.Interrupt()
}

func (pl *protocolLogger) logRead(n int, err error, datafull []byte) {

	if err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			// Dont log timouts
			return
		}

		if pl.logAdapter != nil {
			pl.logAdapter.logMessage(time.Now(), LogTypeFail, err.Error(), []byte{})
		}

		fmt.Printf("PL|%s| fail - '%s'\n", time.Now().Format("20060102 150405.0"), err.Error())
		return
	}

	peek := substr(makeBytesReadable(datafull[:n]), 0, 30)
	if len(peek) > 30 {
		peek = peek + "..."
	}

	if pl.logAdapter != nil {
		pl.logAdapter.logMessage(time.Now(), LogTypeRecv, peek, datafull)
	}

	fmt.Printf("PL|%s| recv - (%d bytes) '%s' \n", time.Now().Format("20060102 150405.0"), n, peek)
}

func (pl *protocolLogger) logWrite(n int, err error, datafull []byte) {

	if err != nil {

		if pl.logAdapter != nil {
			pl.logAdapter.logMessage(time.Now(), LogTypeFail, err.Error(), []byte{})
		}

		fmt.Printf("PL|%s| send - (error) '%s'\n", time.Now().Format("20060102 150405.0"), err.Error())
		return
	}

	peek := substr(makeBytesReadable(datafull), 0, 30)
	if len(datafull) > 30 {
		peek = peek + "..."
	}
	if pl.logAdapter != nil {
		pl.logAdapter.logMessage(time.Now(), LogTypeSend, peek, datafull)
	}
	fmt.Printf("PL|%s| send - (%d bytes) '%s'\n", time.Now().Format("20060102 150405.0"), n, peek)
}

func (pl *protocolLogger) logClose(peer string) {

	if pl.logAdapter != nil {
		pl.logAdapter.logMessage(time.Now(), LogTypeClos, peer, []byte{})
	}

	fmt.Printf("PL|%s| close|\n", time.Now().Format("20060102 150405.0"))
}

func (pl *protocolLogger) Receive(conn net.Conn) ([]byte, error) {
	if !pl.enableLog {
		return pl.protocol.Receive(conn)
	}
	return pl.protocol.Receive(wrapConnWithLogger(pl, conn))
}

func (pl *protocolLogger) Send(conn net.Conn, data [][]byte) (int, error) {
	if !pl.enableLog {
		return pl.protocol.Send(conn, data)
	}
	return pl.protocol.Send(wrapConnWithLogger(pl, conn), data)
}

func (pl *protocolLogger) NewInstance() Implementation {
	return &protocolLogger{
		protocol: pl.protocol.NewInstance(),
	}
}

func Logger(protocol Implementation) Implementation {
	protoLogger := &protocolLogger{
		enableLog: os.Getenv("PROTOLOG_ENABLE") != "",
		protocol:  protocol,
	}

	if protoLogger.enableLog {
		fmt.Printf("PL|%s| start - protocol logging is enabled\n", time.Now().Format("20060102 150405.0"))
	}

	return protoLogger
}

func LoggerWithAdapter(protocol Implementation, logAdapter LogAdapter) Implementation {
	protoLogger := &protocolLogger{
		enableLog:  os.Getenv("PROTOLOG_ENABLE") != "",
		protocol:   protocol,
		logAdapter: logAdapter,
	}

	if protoLogger.enableLog {
		fmt.Printf("PL|%s| start - protocol logging is enabled\n", time.Now().Format("20060102 150405.0"))
	}

	return protoLogger
}

func makeBytesReadable(in []byte) string {
	ret := ""
	for i := 0; i < len(in); i++ {
		if in[i] < 32 {
			ret = ret + utilities.ASCIIMapNotPrintable[in[i]]
		} else {
			ret = ret + string(in[i])
		}
	}
	return ret
}

func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

/* Implement net.Conn as a wrapper */
type netConnLoggerSpy struct {
	conn           net.Conn
	protocolLogger *protocolLogger
}

func (ls *netConnLoggerSpy) Read(b []byte) (n int, err error) {
	n, err = ls.conn.Read(b)
	if ls.protocolLogger.enableLog {
		ls.protocolLogger.logRead(n, err, b)
	}
	return n, err
}
func (ls *netConnLoggerSpy) Write(b []byte) (n int, err error) {
	n, err = ls.conn.Write(b)
	if ls.protocolLogger.enableLog {
		ls.protocolLogger.logWrite(n, err, b)
	}
	return n, err
}
func (ls *netConnLoggerSpy) Close() error {
	if ls.protocolLogger.enableLog {
		ls.protocolLogger.logClose(ls.conn.RemoteAddr().String())
	}
	return ls.conn.Close()
}
func (ls *netConnLoggerSpy) LocalAddr() net.Addr {
	return ls.conn.LocalAddr()
}
func (ls *netConnLoggerSpy) RemoteAddr() net.Addr {
	return ls.conn.RemoteAddr()
}
func (ls *netConnLoggerSpy) SetDeadline(t time.Time) error {
	return ls.conn.SetDeadline(t)
}
func (ls *netConnLoggerSpy) SetReadDeadline(t time.Time) error {
	return ls.conn.SetReadDeadline(t)
}
func (ls *netConnLoggerSpy) SetWriteDeadline(t time.Time) error {
	return ls.conn.SetWriteDeadline(t)
}

func wrapConnWithLogger(pl *protocolLogger, con net.Conn) net.Conn {
	return &netConnLoggerSpy{
		conn:           con,
		protocolLogger: pl,
	}
}
