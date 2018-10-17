package main

import (
	"ConfigAdapter/JsonConfig"
	"transponder/outer"
)

type OutConfig struct {
	InnerServerAddress string
	OuterServerAddress string
	AuthKey            string
}

func main() {
	c := &OutConfig{}
	JsonConfig.Load("outer.config.json", c)
	//启动内部服务
	innerServer := outer.NewInnerServer()
	innerServer.AuthKey = c.AuthKey
	innerServer.Address = c.InnerServerAddress
	go innerServer.StartServer()
	//启动外部服务
	server := outer.NewServer(innerServer)
	server.Address = c.OuterServerAddress
	server.StartServer()
}
