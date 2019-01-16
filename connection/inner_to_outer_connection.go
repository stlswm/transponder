/**
 * @Time: 2019/1/15 14:01
 * @Author: wangmin
 * @File: inner_to_outer_connection.go
 * @Software: GoLand
 */
package connection

import (
	"errors"
	"transponder/event"
	"encoding/json"
	"log"
	"net"
	"time"
	"io"
	"strconv"
	"sync"
)

// 内网到外网服务的连接
type InnerToOuterConnection struct {
	Id                     uint64
	Status                 int
	StatusMonitor          func(id uint64, status int)
	OutServerAddress       string
	OutServerAuthKey       string
	outServerConn          net.Conn
	OutServerConnWriteLock sync.Mutex
	ProxyAddress           string
}

// 与外网服务器通讯
func (itoc *InnerToOuterConnection) communicate(single int) error {
	if itoc.outServerConn == nil {
		return errors.New("please register first")
	}
	var sb []byte
	switch single {
	case event.Ping:
		//ping
		sb = event.GenerateSignal(event.Ping, "")
	case event.Auth:
		//授权
		sb = event.GenerateSignal(event.Auth, itoc.OutServerAuthKey)
	case event.StartProxy:
		//准备转发
		sb = event.GenerateSignal(event.StartProxy, "")
	default:
		return errors.New("不支持的信号类型")
	}
	itoc.OutServerConnWriteLock.Lock()
	_, err := itoc.outServerConn.Write(sb)
	itoc.OutServerConnWriteLock.Unlock()
	if err == nil {
		return nil
	}
	itoc.Close()
	return err
}

// 读取服务器数据
func (itoc *InnerToOuterConnection) Read() {
	for {
		if itoc.Status == StatusProxy || itoc.Status == StatusClose {
			return
		}
		buf := make([]byte, event.PackageLength)
		n, err := itoc.outServerConn.Read(buf)
		if err != nil {
			log.Println("read from server connection error:" + err.Error())
			itoc.Close()
			return
		}
		if n != event.PackageLength {
			log.Println("read from server connection error data length error")
			itoc.Close()
			return
		}
		signal := &event.Signal{}
		err = json.Unmarshal(buf, signal)
		if err != nil {
			log.Println("cant not parse outer server signal data:" + string(buf) + " error：" + err.Error())
			itoc.Close()
			return
		}
		switch signal.T {
		case event.StartProxy:
			itoc.Status = StatusProxy
			itoc.Proxy()
			return
		default:
			log.Println("unknown event:" + strconv.Itoa(signal.T))
			itoc.Close()
			return
		}
	}
}

// 连接外网服务器并注册
func (itoc *InnerToOuterConnection) Register() {
	sc, err := net.Dial("tcp", itoc.OutServerAddress)
	if err != nil {
		log.Println("connect to error:" + err.Error())
		itoc.Close()
		return
	}
	itoc.outServerConn = sc
	err = itoc.communicate(event.Auth)
	if err != nil {
		//注册失败
		log.Println("register fail:" + err.Error())
		return
	}
	itoc.Status = StatusOk
	itoc.StatusMonitor(itoc.Id, itoc.Status)
}

// 维持与服务器的连接
func (itoc *InnerToOuterConnection) Ping() {
	if itoc.Status == StatusInit || itoc.Status == StatusProxy || itoc.outServerConn == nil {
		return
	}
	err := itoc.communicate(event.Ping)
	if err != nil {
		log.Println("ping fail:" + err.Error())
		return
	}
}

// 开始数据转发
func (itoc *InnerToOuterConnection) Proxy() {
	//log.Println("内部服务开始转发")
	itoc.Status = StatusProxy
	itoc.StatusMonitor(itoc.Id, itoc.Status)
	//发送转发信号
	err := itoc.communicate(event.StartProxy)
	if err != nil {
		log.Println("send proxy signal fail:" + err.Error())
		return
	}
	//开始转发
	itoc.outServerConn.SetReadDeadline(time.Now().Add(time.Second * 90))
	itoc.outServerConn.SetWriteDeadline(time.Now().Add(time.Second * 90))
	proxyConn, err := net.Dial("tcp", itoc.ProxyAddress)
	if err != nil {
		itoc.Close()
		log.Println("connect to proxy server error:" + err.Error())
		return
	}
	proxyConn.SetReadDeadline(time.Now().Add(time.Second * 90))
	proxyConn.SetWriteDeadline(time.Now().Add(time.Second * 90))
	go func() {
		io.Copy(itoc.outServerConn, proxyConn)
		itoc.Close()
		if proxyConn != nil {
			proxyConn.Close()
		}
	}()
	go func() {
		io.Copy(proxyConn, itoc.outServerConn)
		itoc.Close()
		if proxyConn != nil {
			proxyConn.Close()
		}
	}()
}

// 关闭连接
func (itoc *InnerToOuterConnection) Close() {
	itoc.Status = StatusClose
	if itoc.outServerConn != nil {
		itoc.outServerConn.Close()
		itoc.outServerConn = nil
	}
	itoc.StatusMonitor(itoc.Id, itoc.Status)
}
