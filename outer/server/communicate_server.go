package server

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"time"
)

// 内部服务通讯对象
type CommunicateServer struct {
	Address     string
	innerServer *InnerServer
	innerConn   net.Conn //内部服务器连接
}

// 启动服务
func (c *CommunicateServer) StartServer() {
	log.Println("启动内部服务器通讯服务，" + c.Address)
	go c.Ping()
	tcpAddr, _ := net.ResolveTCPAddr("tcp", c.Address)
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		tcpConn, _ := tcpListener.AcceptTCP()
		log.Println("内部服务器通讯服务接收新连接：" + tcpConn.RemoteAddr().String())
		if c.innerConn != nil {
			c.innerConn.Close()
			c.innerConn = nil
		}
		c.innerConn = tcpConn
	}
}

// Ping
func (c *CommunicateServer) Ping() {
	t := time.NewTicker(time.Second * 30)
	for {
		<-t.C
		if c.innerConn != nil {
			c.sendToInnerServer(struct {
				E int
			}{
				E: 0,
			})
		}
	}
}

// 发送数据到内网服务器
func (c *CommunicateServer) sendToInnerServer(data interface{}) error {
	dataByte, err := json.Marshal(data)
	if err == nil {
		if c.innerConn != nil {
			_, err := c.innerConn.Write([]byte(string(dataByte) + "\r"))
			if err != nil {
				c.innerConn.Close()
				c.innerConn = nil
				return err
			}
			return nil
		}
	}
	return err
}

// 新连接请求
func (c *CommunicateServer) NewClient() (error, net.Conn) {
	log.Println("向内部服务器发送新连接请求")
	err := c.sendToInnerServer(struct {
		E int
	}{
		E: 1,
	})
	if err == nil {
		select {
		case conn := <-c.innerServer.innerQueue:
			return nil, conn
		case <-time.After(10 * time.Second): //超时10秒
			return errors.New("get connection timeout"), nil
		}
	}
	return err, nil
}

// 获取内部服务通讯实例
func NewCommunicateServer(i *InnerServer) *CommunicateServer {
	c := &CommunicateServer{
		innerServer: i,
		Address:     ":9091",
	}
	return c
}
