package server

import (
	"log"
	"net"
	"strings"
)

// 内部服务对象
type InnerServer struct {
	Address    string
	innerQueue chan net.Conn
}

// 启动服务
func (i *InnerServer) StartServer() {
	log.Println("启动内部服务器服务，" + i.Address)
	addrSlice := strings.Split(i.Address, "://")
	if len(addrSlice) < 2 {
		panic(i.Address + " format error.")
	}
	if addrSlice[0] != "tcp" {
		panic("inner server only support tcp.")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addrSlice[1])
	if err != nil {
		panic(err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	for {
		tcpConn, _ := listener.AcceptTCP()
		//log.Println("内部服务器接收新连接：" + tcpConn.RemoteAddr().String())
		i.innerQueue <- tcpConn
	}
}

// 获取内部服务实例
func NewInnerServer() *InnerServer {
	i := &InnerServer{
		innerQueue: make(chan net.Conn, 10240),
	}
	go i.StartServer()
	return i
}
