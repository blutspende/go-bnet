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

 // Sending Data
 n, err := tcpClient.Send([]byte("Hello TCP/IP"))

 // Receiving data
 receivedMsg, err := tcpClient.Receive()
```
### TCP/IP Server

``` go
type MySessionData struct {
}

// Event: New conncetin est.
func (s *MySessionData) Connected(session Session) {
	fmt.Println("Connected Event")
}
// Event: Disconnected
func (s *MySessionData) Connected(session Session) {
	fmt.Println("Disconnected Event")
}
// Event: Some error occurred
func (s *MySessionData) Error(session Session, errorType ErrorType, err error) {
  fmt.Println(err)
}
// Event: Data received
func (s *MySessionData) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
  fmt.Println("Data received : ", string(fileData))
  
  // Sending data for the session
  session.Send([]byte("Some testdata"))
}

func main() {

  server := CreateNewTCPServerInstance(4009,
		protocol.STXETX(),  // use STX-ETX protocl
		HAProxySendProxyV2,  // Deal with the header and ignore empty requests
		100) // Max Connections	 
  
  go server.Run(MySessionData) // this starts the asynchroneous process 
  ...
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
