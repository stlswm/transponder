package main

import (
	"ConfigAdapter/JsonConfig"
	"strings"
	"time"
	"sync"
	"transponder/connection"
	"log"
)

// 外网服务器关系维护
type ServerInner struct {
	RegisterAddress string
	AuthKey         string
	connId          uint64
	connNum         int64
	ConnList        sync.Map //到外网服务的连接
	ProxyAddress    string
}

// 连接id生成
func (si *ServerInner) generateConnId() uint64 {
	si.connId++
	if si.connId > 4294967296 {
		si.connId = 1
	}
	_, ok := si.ConnList.Load(si.connId)
	if !ok {
		return si.connId
	}
	return si.generateConnId()
}

// 连接维持
func (si *ServerInner) batchPing() {
	t := time.NewTicker(time.Second * 10)
	for {
		<-t.C
		si.ConnList.Range(func(key, value interface{}) bool {
			innerConn := value.(*connection.InnerToOuterConnection)
			innerConn.Ping()
			return true
		})
	}
}

// 批量创建新连接到外网服务器
func (si *ServerInner) batchConnectToOuter(num int) {
	for i := 1; i <= num; i++ {
		c := &connection.InnerToOuterConnection{
			Id: si.generateConnId(),
			StatusMonitor: func(id uint64, status int) {
				switch status {
				case connection.StatusClose:
					si.ConnList.Delete(id)
					si.connNum--
					if si.connNum < 10 {
						si.batchConnectToOuter(10)
					}
				}
			},
			OutServerAddress:       si.RegisterAddress,
			OutServerAuthKey:       si.AuthKey,
			OutServerConnWriteLock: sync.Mutex{},
			ProxyAddress:           si.ProxyAddress,
		}
		c.Register()
		go c.Read()
		si.connNum++
		si.ConnList.Store(c.Id, c)
	}
}

func main() {
	type InnerConfig struct {
		RegisterAddress string
		ProxyAddress    string
		AuthKey         string
	}
	c := &InnerConfig{}
	JsonConfig.Load("inner.config.json", c)
	//注册地址
	addrSlice := strings.Split(c.RegisterAddress, "://")
	if len(addrSlice) < 2 {
		panic(c.RegisterAddress + " format error")
	}
	if addrSlice[0] != "tcp" {
		panic("register address only support tcp")
	}
	registerAddress := addrSlice[1]
	//转发地址
	addrSlice = strings.Split(c.ProxyAddress, "://")
	if len(addrSlice) < 2 {
		panic(c.ProxyAddress + " format error")
	}
	if addrSlice[0] != "tcp" {
		panic("proxy address only support tcp")
	}
	proxyAddress := addrSlice[1]
	si := &ServerInner{
		RegisterAddress: registerAddress,
		AuthKey:         c.AuthKey,
		ProxyAddress:    proxyAddress,
	}
	si.batchConnectToOuter(10)
	log.Println("start success")
	si.batchPing()
}
