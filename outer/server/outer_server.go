package server

import (
	"io"
	"log"
	"net"
	"time"
	"strings"
)

// 外部服务对象
type OuterServer struct {
	communicateServer *CommunicateServer
	Address           string
}

// 启动服务
func (o *OuterServer) StartServer() {
	log.Println("启动外部服务：" + o.Address)
	addrSlice := strings.Split(o.Address, "://")
	if len(addrSlice) < 2 {
		panic(o.Address + " format error.")
	}
	switch addrSlice[0] {
	case "tcp":
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
			conn, _ := listener.AcceptTCP()
			go o.IOExchange(conn)
		}
	case "unix":
		unixAddr, err := net.ResolveUnixAddr("unix", addrSlice[1])
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenUnix("unix", unixAddr)
		if err != nil {
			panic(err)
		}
		defer listener.Close()
		for {
			conn, _ := listener.Accept()
			go o.IOExchange(conn)
		}
	default:
		panic("net type " + addrSlice[0] + " is not allow.")
	}
}

// io转发
func (o *OuterServer) IOExchange(outConn net.Conn) {
	//log.Println("外部服务:" + o.Address + "，接收新连接启动转发...")
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
		//log.Println("关闭到客户端的连接")
	}()
	go func() {
		io.Copy(outConn, innerConn)
		innerConn.Close()
		//log.Println("关闭到内部服务器的连接")
	}()
}

// 获取外部服务实例
func NewOuterServer(c *CommunicateServer) *OuterServer {
	o := &OuterServer{
		communicateServer: c,
	}
	return o
}
