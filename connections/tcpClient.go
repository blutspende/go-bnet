package connections

type tcpClient struct {
}

func CreateNewTCPClient() Connections {
	return &tcpClient{}
}

func (s *tcpClient) Run() {}

func (s *tcpClient) Send(data interface{}) {

}
