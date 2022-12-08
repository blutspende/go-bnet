package protocol

import (
	"fmt"
	"net"
	"os"
	"time"
)

type protocolLogger struct {
	enableLog bool
	protocol  Implementation
}

func (pl *protocolLogger) Interrupt() {
	fmt.Printf("PL|%s interrupt connection\n", time.Now().Format("20060102 150405.0"))
	pl.protocol.Interrupt()
}

func (pl *protocolLogger) logRead(bytes int, err error, datafull string) {
	fmt.Printf("PL|%s recv - (%d bytes) %s\n", time.Now().Format("20060102 150405.0"), "???")
}

func (pl *protocolLogger) logWrite(n int, err error, datafull string) {
	peek := "<empty>"
	fmt.Printf("PL|%s send - %s|\n", time.Now().Format("20060102 150405.0"), peek)
}

func (pl *protocolLogger) logClose(peer string) {
	fmt.Printf("PL|%s close|\n", time.Now().Format("20060102 150405.0"))
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
	return &protocolLogger{
		enableLog: (os.Getenv("PROTOLOG_ENABLE") == "true"),
		protocol:  protocol,
	}
}

func substr(input string, start int, length int) string {
	// TODO
	return input
}

/* Implement net.Conn as a wrapper */
type netConnLoggerSpy struct {
	conn           net.Conn
	protocolLogger *protocolLogger
}

func (ls *netConnLoggerSpy) Read(b []byte) (n int, err error) {
	n, err = ls.conn.Read(b)
	if ls.protocolLogger.enableLog {
		ls.protocolLogger.logRead(n, err, string(b))
	}
	return n, err
}
func (ls *netConnLoggerSpy) Write(b []byte) (n int, err error) {
	n, err = ls.conn.Write(b)
	if ls.protocolLogger.enableLog {
		ls.protocolLogger.logWrite(n, err, string(b))
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
