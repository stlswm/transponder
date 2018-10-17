package outer

import (
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// 外部服务对象
type Server struct {
	innerServer *InnerServer
	Address     string
}

// 启动服务
func (o *Server) StartServer() {
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
		_, err := os.Stat(addrSlice[1])
		if err == nil {
			os.Remove(addrSlice[1])
		}
		unixAddr, err := net.ResolveUnixAddr("unix", addrSlice[1])
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenUnix("unix", unixAddr)
		if err != nil {
			panic(err)
		}
		os.Chmod(addrSlice[1], 0777)
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
func (o *Server) IOExchange(outConn net.Conn) {
	outConn.SetReadDeadline(time.Now().Add(time.Second * 90))
	outConn.SetWriteDeadline(time.Now().Add(time.Second * 90))
	err, innerConn := o.innerServer.NewClientRequest()
	if err != nil {
		log.Println(err.Error())
		outConn.Close()
		return
	}
	innerConn.SetReadDeadline(time.Now().Add(time.Second * 90))
	innerConn.SetWriteDeadline(time.Now().Add(time.Second * 90))
	go func() {
		io.Copy(innerConn, outConn)
		outConn.Close()
	}()
	go func() {
		io.Copy(outConn, innerConn)
		innerConn.Close()
	}()
}

// 获取外部服务实例
func NewServer(i *InnerServer) *Server {
	o := &Server{
		innerServer: i,
	}
	return o
}
