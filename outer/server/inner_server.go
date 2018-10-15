package server

import (
	"log"
	"net"
)

// 内部服务对象
type InnerServer struct {
	Address    string
	innerQueue chan net.Conn
}

// 启动服务
func (i *InnerServer) StartServer() {
	log.Println("启动内部服务器服务，" + i.Address)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", i.Address)
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		tcpConn, _ := tcpListener.AcceptTCP()
		//log.Println("内部服务器接收新连接：" + tcpConn.RemoteAddr().String())
		i.innerQueue <- tcpConn
	}
}

// 获取内部服务实例
func NewInnerServer() *InnerServer {
	i := &InnerServer{
		Address:    ":9092",
		innerQueue: make(chan net.Conn, 10240),
	}
	go i.StartServer()
	return i
}
