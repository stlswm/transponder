package main

import (
	"ConfigAdapter/JsonConfig"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"transponder/connection"
)

// 内部服务对象
type ServerForInner struct {
	Address      string // 监听地址
	AuthKey      string // 连接授权码
	connId       uint64 // 连接id
	connLock     sync.Mutex
	tempConnList sync.Map // 内网服务连接列表
}

// 连接id生成
func (sfi *ServerForInner) generateConnId() uint64 {
	sfi.connId++
	if sfi.connId > 4294967296 {
		sfi.connId = 1
	}
	_, ok := sfi.tempConnList.Load(sfi.connId)
	if !ok {
		return sfi.connId
	}
	sfi.generateConnId()
	return sfi.connId
}

// 启动服务
func (sfi *ServerForInner) StartServer() {
	log.Println("start server for inner service at address " + sfi.Address)
	addrSlice := strings.Split(sfi.Address, "://")
	if len(addrSlice) < 2 {
		panic(sfi.Address + " format error.")
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
	go sfi.authOverdueCheck()
	for {
		id := sfi.generateConnId()
		tcpConn, _ := listener.AcceptTCP()
		innerConn := &connection.InnerConnection{
			Id:      id,
			Created: time.Now().Unix(),
			AuthKey: sfi.AuthKey,
			Conn:    tcpConn,
			StatusMonitor: func(id uint64, status int) {
				switch status {
				case connection.StatusOk:
					log.Println("new connection register")
				case connection.StatusClose:
					// 回收资源
					sfi.tempConnList.Delete(id)
				}
			},
		}
		sfi.tempConnList.Store(id, innerConn)
		go innerConn.Read()
	}
}

// 状态检查关闭授权过期的连接
func (sfi *ServerForInner) authOverdueCheck() {
	t := time.NewTicker(time.Second * 5)
	for {
		<-t.C
		timeNow := time.Now().Unix()
		sfi.tempConnList.Range(func(key, value interface{}) bool {
			innerConn := value.(*connection.InnerConnection)
			if timeNow-innerConn.Created > 10 && innerConn.Status == connection.StatusInit {
				// 超时没有认证的连接关闭并释放资源
				log.Println(innerConn.Conn.RemoteAddr().String() + " auth timeout closed by server")
				innerConn.Close()
				sfi.tempConnList.Delete(key)
			}
			return true
		})
	}
}

// 转发
func (sfi *ServerForInner) IOExchange(conn net.Conn) {
	var innerConn *connection.InnerConnection
	sfi.connLock.Lock()
	sfi.tempConnList.Range(func(key, value interface{}) bool {
		tmpConn := value.(*connection.InnerConnection)
		if tmpConn.Status == connection.StatusOk && tmpConn.Conn != nil {
			sfi.tempConnList.Delete(key)
			innerConn = tmpConn
			return false
		}
		return true
	})
	sfi.connLock.Unlock()
	if innerConn == nil {
		// 无可用连接
		log.Println("no usable connection")
		_ = conn.Close()
		return
	}
	innerConn.ProxyRequest(conn)
}

func main() {
	type OutConfig struct {
		InnerServerAddress string
		OuterServerAddress string
		AuthKey            string
	}
	c := &OutConfig{}
	err := JsonConfig.Load("outer.config.json", c)
	if err != nil {
		panic("can not parse config file:outer.config.json")
	}
	// 启动内部服务
	serverForInner := &ServerForInner{
		Address:  c.InnerServerAddress,
		AuthKey:  c.AuthKey,
		connLock: sync.Mutex{},
	}
	go serverForInner.StartServer()
	// 启动外部服务
	log.Println("start out service server at " + c.OuterServerAddress)
	addrSlice := strings.Split(c.OuterServerAddress, "://")
	if len(addrSlice) < 2 {
		panic(c.OuterServerAddress + " format error")
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
			go serverForInner.IOExchange(conn)
		}
	case "unix":
		_, err := os.Stat(addrSlice[1])
		if err == nil {
			_ = os.Remove(addrSlice[1])
		}
		unixAddr, err := net.ResolveUnixAddr("unix", addrSlice[1])
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenUnix("unix", unixAddr)
		if err != nil {
			panic(err)
		}
		_ = os.Chmod(addrSlice[1], 0777)
		defer listener.Close()
		for {
			conn, _ := listener.Accept()
			go serverForInner.IOExchange(conn)
		}
	default:
		panic("net type " + addrSlice[0] + " is not allow")
	}
}
