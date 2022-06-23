package examples

//
//func (session *SessionHandlerTCP) handleDataReived(source string, filedata []byte, receivetimestamp time.Time) err {
//	message := unmarshal(filedata....)
//	repo = repostiory.persist_val(message)
//	trigger_transmissioN_to_cia(repo)
//	return nil
//}
//
//// for servermode: this method is called on connect.ö the conneciton remains active for as long as the funciton does not exit
//func (session *SessionHandlerTCP) serverSession(OurConnection netHandle, source string) {
//	if !isListed(con) {
//		return
//	}
//	// alternative. non-blocking
//	// logic: check for new data... send new data
//	for netHandle.isAlive() {
//		analysis_request := <-events_from_elswehere_in_our_application // blocking!
//		netHandle.Send(anayls_request.data)
//
//		if schlechte_laune {
//			con.stop() // close connection
//		}
//	}
//	// return = disconnect this connection
//}
//
//func main() {
//serve = CreateTCPServer(4001, ProtocolASTMWrappedSTX:, NO_PROXY, 100 maxconnections, DefaultTimings)
////sftp = CreateSFTPClient('host.hostus.net', '/wodiedateniegen', 'user', 'pass', '1938123083key', DefaultTimings)
//client := CreateTCPClient("meingeraet.netzwerk", 4001, NoProxy, ProtocolASTMWrappedSTX, DefaultTimings)
//var handler_tcp SessionHandlerTCP
//
////var handler_ftp SessionHandlerFTP
//go serve.run(handler_tcp)
////go sftp.run(handler_sftp)
////go client.run(handler_tcpclient)
//...
//func timer() {
//	for bestllung: range bestellungen{
//		daten, := marshal(bestellung)
//		client.Send(daten)
//	}
//}
//myprogram.api.run()
//}
