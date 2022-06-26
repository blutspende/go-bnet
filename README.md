# Go BloodLab Net

A libarary to communicate over various networking-protocols through one API. go-bloodlab-net works for systems that require blocked transfer and hides the implementation details, but is not suited for protocols that require control over the connection

###### Install
`go get github.com/DRK-Blutspende-BaWueHe/go-astm`

### TCP/IP Client  with synchroneous reception
Use this when you implement a client do not need asynchronous receiving.

``` go
tcpClient := CreateNewTCPClient("127.0.0.1", 4001, 
 PROTOCOL_RAW, // encoding of incoming data
 PROTOCOL_RAW, // encoding of sent data
 NoLoadBalancer, 
 DefaultTCPTiming)

if err := tcpClient.Connect(); err != nil {
  log.Panic(err)
}

n, err := tcpClient.Send([]byte("Hello TCP/IP"))
...
message, err := tcpClient.Receive()
...
```
### TCP/IP Client with asynchroneous reception
Use this when you implement a client and transmissions may occur asynchroneous.

``` go
import "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"

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
	fmt.Println("Data received")
}

func main() {

  tcpClient := CreateNewTCPClient("127.0.0.1", 4001, 
  PROTOCOL_RAW, // protocol for incoming
  PROTOCOL_RAW, // protocol for sending
  NoLoadbalancer, 
  DefaultTCPTiming)
    
  go v.Run(MySessionData) // this starts the asynchroneous process 
  
  v.Send([]byte{ENQ}) // sync. sending is still possible 
  ...
```