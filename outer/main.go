package main

import (
	"transponder/outer/server"
	"ConfigAdapter/JsonConfig"
)

type Config struct {
	CommunicateServerAddress string
	InnerServerAddress       string
	OuterServerAddress       string
}

func main() {
	c := &Config{}
	JsonConfig.Load("config.json", c)
	//启动内部服务
	innerServer := server.NewInnerServer()
	innerServer.Address = c.InnerServerAddress
	//启动内部服务器通讯服务
	communicateServer := server.NewCommunicateServer(innerServer)
	communicateServer.Address = c.CommunicateServerAddress
	go communicateServer.StartServer()
	//启动外部服务
	outServer := server.NewOuterServer(communicateServer)
	outServer.Address = c.OuterServerAddress
	outServer.StartServer()
}
