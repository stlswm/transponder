package server

import (
	"io"
	"log"
	"net"
	"time"
)

// 外部服务对象
type OuterServer struct {
	communicateServer *CommunicateServer
	Address           string
}

// 启动服务
func (o *OuterServer) StartServer() {
	log.Println("启动外部服务：" + o.Address)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", o.Address)
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		tcpConn, _ := tcpListener.AcceptTCP()
		go o.IOExchange(tcpConn)
	}
}

// io转发
func (o *OuterServer) IOExchange(outConn net.Conn) {
	log.Println("外部服务:" + o.Address + "，接收新连接启动转发...")
	outConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	outConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	err, innerConn := o.communicateServer.NewClient()
	if err != nil {
		log.Println(err.Error())
		outConn.Close()
		return
	}
	//log.Println("获取到连接，开始转发")
	innerConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	innerConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	go func() {
		io.Copy(innerConn, outConn)
		outConn.Close()
		log.Println("关闭到客户端的连接")
	}()
	go func() {
		io.Copy(outConn, innerConn)
		innerConn.Close()
		log.Println("关闭到内部服务器的连接")
	}()
}

// 获取外部服务实例
func NewOuterServer(c *CommunicateServer) *OuterServer {
	o := &OuterServer{
		communicateServer: c,
		Address:           ":9090",
	}
	return o
}
