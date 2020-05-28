package main

import (
	"ConfigAdapter/JsonConfig"
	"log"
	"strings"
	"sync"
	"time"
	"transponder/connection"
)

// 外网服务器关系维护
type ServerInner struct {
	RegisterAddress string
	AuthKey         string
	connId          uint64    // 当前连接Id
	connFree        sync.Map  //空闲连接
	connMaxFree     int       //最大空闲连接数
	connChangeSign  chan bool //连接增加或减少
	ConnList        sync.Map  //到外网服务的连接
	ProxyAddress    string    //转发地址
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

// 检查连接数量
func (si *ServerInner) connectionNumCheck() {
	for {
		select {
		case <-si.connChangeSign:
			freeNum := 0
			si.connFree.Range(func(key, value interface{}) bool {
				freeNum++
				return true
			})
			newNum := si.connMaxFree - freeNum
			if newNum > 0 {
				go si.batchConnectToOuter(newNum)
			}
		}
	}
}

// 批量创建新连接到外网服务器
func (si *ServerInner) batchConnectToOuter(num int) {
	cidArr := make([]uint64, num)
	for i := 1; i <= num; i++ {
		cid := si.generateConnId()
		cidArr = append(cidArr, cid)
		c := &connection.InnerToOuterConnection{
			Id: cid,
			StatusMonitor: func(id uint64, status int) {
				switch status {
				case connection.StatusProxy:
					si.connFree.Delete(id)
				case connection.StatusClose:
					si.connFree.Delete(id)
					si.ConnList.Delete(id)
				}
				si.connChangeSign <- true
			},
			OutServerAddress:       si.RegisterAddress,
			OutServerAuthKey:       si.AuthKey,
			OutServerConnWriteLock: sync.Mutex{},
			ProxyAddress:           si.ProxyAddress,
		}
		si.ConnList.Store(cid, c)
		si.connFree.Store(cid, true)
	}
	for _, cid := range cidArr {
		c, ok := si.ConnList.Load(cid)
		if ok {
			c.(*connection.InnerToOuterConnection).Register()
			go c.(*connection.InnerToOuterConnection).Read()
		}
	}
}

func main() {
	type InnerConfig struct {
		RegisterAddress string
		ProxyAddress    string
		AuthKey         string
	}
	c := &InnerConfig{}
	err := JsonConfig.Load("inner.config.json", c)
	if err != nil {
		panic("can not parse config file:inner.config.json")
	}
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
		connMaxFree:     10,
		connChangeSign:  make(chan bool, 90),
	}
	log.Println("start success")
	si.connChangeSign <- true
	go si.connectionNumCheck()
	si.batchPing()
}
