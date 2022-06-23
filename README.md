# Go BloodLab Net

A libarary to communicate over various networking-protocols through one API. go-bloodlab-net works for systems that require blocked transfer and hides the implementation details, but is not suited for protocols that require control over the connection



# Quick start


``` go
type MySessionData struct {
}

func (session *MySessionData) handleDataReived(source string, filedata []byte, receivetimestamp time.Time) err {	
  fmt.Println(string(filedata), " received from ", source)  
	return nil
}

func (session *MySessionData) serverSession(session go_bloodlab_net.Session, source string) {
}

func main() {
  h := go_bloodlab_net.CreateTCPServer(4001, sth)
  v := go_bloodlab_net.CreateFTPServer(...)
    
  go h.Run(MySessionData)
  go v.Run(MySessionData)
  
  
}

```
