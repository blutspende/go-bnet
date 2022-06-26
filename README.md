# Go BloodLab Net

A libarary to communicate over various networking-protocols through one API. go-bloodlab-net works for systems that require blocked transfer and hides the implementation details, but is not suited for protocols that require control over the connection

###### Install
`go get github.com/DRK-Blutspende-BaWueHe/go-astm`

# TCP/IP Client  with synchroneous reception
Use this when you implement a client do not need asynchronous receiving.

``` go
	tcpClient := CreateNewTCPClient("127.0.0.1", 4001, PROTOCOL_RAW, PROTOCOL_RAW, NoProxy, DefaultTCPTiming)

  if err := tcpClient.Connect(); err != nil {
    log.Panic(err)
  }

  n, err := tcpClient.Send([]byte("Hello TCP/IP"))
  ...
  message, err := tcpClient.Receive()
  ...
```
# TCP/IP Client with asynchroneous reception
Use this when you implement a client and transmissions may occur asynchroneous.

``` go
import "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"

type MySessionData struct {
}

// Event: New conncetin est.
func (s *ClientTestSession) Connected(session Session) {
	fmt.Println("Connected Event")
}
// Event: Disconnected
func (s *ClientTestSession) Connected(session Session) {
	fmt.Println("Disconnected Event")
}
// Event: Some error occurred
func (s *ClientTestSession) Error(session Session, errorType ErrorType, err error) {
  fmt.Println(err)
}
// Event: Data received
func (s *ClientTestSession) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	fmt.Println("Data received")
}

func main() {

  tcpClient := CreateNewTCPClient("127.0.0.1", 4001, 
  PROTOCOL_RAW, // protocol for incoming
  PROTOCOL_RAW, // protocol for sending
  NoLoadbalancer, 
  DefaultTCPTiming)
    
  go v.Run(MySessionData)
  
  v.Send([]byte{ENQ})
  ...
```