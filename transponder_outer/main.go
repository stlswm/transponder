package main

import "transponder_outer/server"

func main() {
	innerServer := server.NewInnerServer()
	communicateServer := server.NewCommunicateServer(innerServer)
	go communicateServer.StartServer()
	server.NewOuterServer(communicateServer).StartServer()
}
