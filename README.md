# Go BloodLab Net

A libarary to simplify communication with laboratory instruments.

###### Install
`go get github.com/blutspende/go-bloodlab-net`

###### Features
  - TCP/IP Server implementation
  - TCP/IP Client implementation
  - Low-level protocols : 
      - RAW `protocol.Raw()` 
	  - STX-ETX `protocol.STXETX()`  
	  - MLLP (for HL7) `protocol.MLLP()`
	  - Lis1A1  `protocol.Lis1A1()`

### TCP/IP Client 

Create a client to send data. 

``` go
 tcpClient := CreateNewTCPClient("127.0.0.1", 4001, protocol.Raw(), NoLoadBalancer)

 if err := tcpClient.Connect(); err != nil {  
   log.Panic(err)
 }

 n, err := tcpClient.Send([]byte("Hello TCP/IP"))

 receivedMsg, err := tcpClient.Receive()
```
### TCP/IP Server

``` go
package main

import (
	"fmt"
	"time"

	bloodlabnet "github.com/blutspende/go-bnet"
	"github.com/blutspende/go-bnet/protocol"
)

type MySessionHandler struct {
}

func (s *MySessionHandler) Connected(session bloodlabnet.Session) {
	fmt.Println("Disconnected Event")
}

func (s *MySessionHandler) Disconnected(session bloodlabnet.Session) {
	fmt.Println("Disconnected Event")
}

func (s *MySessionHandler) Error(session bloodlabnet.Session, errorType bloodlabnet.ErrorType, err error) {
	fmt.Println(err)
}

func (s *MySessionHandler) DataReceived(session bloodlabnet.Session, data []byte, receiveTimestamp time.Time) {
	
  rad, _ := session.RemoteAddress()
	fmt.Println("From %s received : ", rad, string(data))
	
  session.Send([]byte(fmt.Sprintf("You are sending from %s", rad)))
}

func main() {

	server := bloodlabnet.CreateNewTCPServerInstance(4009,
		protocol.STXETX(),
		bloodlabnet.HAProxySendProxyV2,
		100) // Max Connections

	server.Run(&MySessionHandler{})
}
```

## Protocols

### Raw Protocol (TCP/Client + TCP/Server)
Raw communication for tcp-ip. 

### STX-ETX Protocol (TCP/Client + TCP/Server)
Mesasge are embedded in <STX> (Ascii 0x02) and <ETX> (Ascii 0x03) to indicate start and end. At the end of each transmission the transmissions contents are passed further for higher level protocols.

```Transmission example
 .... <STX>Some data<ETX> This data here is ignored <STX>More data<ETX> ....
```

### MLLP Protocol (TCP/Client + TCP/Server)
Mesasge are embedded in <VT> (Ascii 11) and <FS> (Ascii 28) terminated with <CR> (Ascii 13) to indicate start and end. At the end of each transmission the transmissions contents are passed further for higher level protocols.

```Transmission example
 .... <VT>Some data<FS><CR> This data here is ignored <VT>More data<FS><CR> ....
```
For some vendors start/stop - bytes might be altered. In that case you can use DefaultMLLPProtocolSettings and use SetStartByte and SetStopByte to change them.
```
tcpServer := bloodlabnet.CreateNewTCPServerInstance(config.TCPListenerPort,
  bloodlabnetProtocol.MLLP(bloodlabnetProtocol.DefaultMLLPProtocolSettings().SetStartByte(0)),
  bloodlabnet.HAProxySendProxyV2, config.TCPServerMaxConnections)
```
### Lis1A1 Protocol (TCP/Client + TCP/Server)
Lis1A1 is a low level protocol for submitting data to laboratory instruments, typically via serial line.
Settings for Lis1A1 low level protocol (multiple settings can be chained):
```
// enables/disables verifying the checksum of received messages
EnableStrictChecksum() | DisableStrictChecksum() 

// enables/disables adding the frame number to the start of the sent message frame
EnableFrameNumber() | DisableFrameNumber()

// enables/disables verifying the frame number of received messages
EnableStrictFrameOrder() | DisableStrictFrameOrder()

// enables/disables sending <CR><ETX> as frame end, by default frame end is <ETX>
EnableAppendCarriageReturnToFrameEnd() | DisableAppendCarriageReturnToFrameEnd()

// sets line ending, by default line end is <CR><LF>
SetLineEnding(lineEnding []byte)
```
### au6xx Protocol (TCP/Client + TCP/Server)
The au6xx is the low-level protocol required for connecting to Beckman&Coulter AU6xx systems.
```
```


## TCP/IP Server Configuration

### Connection timeout
Portscanners or Loadbalancers do connect just to disconnect a second later. By default the session initiation
happens not on connect, rather when the first byte is sent. 

Notice that certain instruments require the connection to remain opened, even if there is no data.

#### Disable the connection-timeout
``` golang
config := DefaultTCPServerSettings
config.SessionAfterFirstByte = false // Default: true
```

#### Require the first Byte to be sent within a timelimit 
``` golang
config := DefaultTCPServerSettings
config.SessionInitationTimeout = time.Second * 3  // Default: 0
```

#### Configure blacklist
``` golang
tcpServerSettings := bnet.DefaultTCPServerSettings
	if len(config.BlackListedTCPClientIPAddresses) > 0 {
		tcpServerSettings.BlackListedIPAddresses = strings.Split(config.BlackListedTCPClientIPAddresses, ",")
	}
```

## Add low-level Logging : Protcol-Logger 

Logging can be added to any protocol by wrapping the Protocol into the logger. This does not affect the functionality.
In addition set the **environment-variable** PROTOLOG_ENABLE to true.

PROTOLOG_ENABLE = (traffic|extended)

  1. traffic = logs only the 
  2. extended = would log the traffic plus the states of the FSM
  
``` bash
set PROTOLOG_ENABLE=extended
```
``` golang
tcpServer := bloodlabnet.CreateNewTCPServerInstance(config.TCPListenerPort,
  bloodlabnetProtocol.Logger(
    bloodlabnetProtocol.MLLP(bloodlabnetProtocol.DefaultMLLPProtocolSettings().SetStartByte(0))
  ),
  bloodlabnet.HAProxySendProxyV2, config.TCPServerMaxConnections)

````
