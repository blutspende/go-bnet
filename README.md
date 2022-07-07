# Go BloodLab Net

A libarary to simplify communication with laboratory instruments.

###### Install
`go get github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net`

###### Features
  - TCP/IP Server implementation
  - TCP/IP Client implementation
  - FTP Client implementation
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

	bloodlabnet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol"
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

### SFTP Client
Connect to a sftp server and keep polling. New files are presented through same API like TCP-Server and TCP-Client as if they were transmitted that way.

### Protocols

#### Raw Protocol (TCP/Client + TCP/Server)
Raw communication for tcp-ip. 

#### STX-ETX Protocol (TCP/Client + TCP/Server)
Mesasge are embedded in <STX> (Ascii 0x02) and <ETX> (Ascii 0x03) to indicate start and end. At the end of each transmission the transmissions contents are passed further for higher level protocols.

```Transmission example
 .... <STX>Some data<ETX> This data here is ignored <STX>More data<ETX> ....
```

#### MLLP Protocol (TCP/Client + TCP/Server)
Mesasge are embedded in <VT> (Ascii 11) and <FS> (Ascii 28) terminated with <CR> (Ascii 13) to indicate start and end. At the end of each transmission the transmissions contents are passed further for higher level protocols.

```Transmission example
 .... <VT>Some data<FS><CR> This data here is ignored <VT>More data<FS><CR> ....
```

#### Lis1A1 Protocol (TCP/Client + TCP/Server)
Lis1A1 is a low level protocol for submitting data to laboratory instruments, typically via serial line.


### TCP/IP Server Configuration

#### Connection timeout
Portscanners or Loadbalancers do connect just to disconnect a second later. By default the session initiation
happens not on connect, rather when the first byte is sent. 

Notice that certain instruments require the connection to remain opened, even if there is no data. 

##### Disable the connection-timeout
``` golang
config := DefaultTCPServerSettings
config.SessionAfterFirstByte = false // Default: true
```

##### Require the first Byte to be sent within a timelimit 
``` golang
config := DefaultTCPServerSettings
config.SessionInitationTimeout = time.Second * 3  // Default: 0
```