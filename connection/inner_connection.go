/**
 * @Time: 2019/1/15 14:50
 * @Author: wangmin
 * @File: inner_connection.go
 * @Software: GoLand
 */
package connection

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	"transponder/event"
)

const StatusInit = 0
const StatusOk = 2
const StatusProxy = 3
const StatusClose = 4

// 内部连接对象
type InnerConnection struct {
	Id            uint64   //连接id
	Status        int      //连接状态
	AuthKey       string   //授权码
	Created       int64    //创建时间
	Conn          net.Conn //内网服务连接对象
	proxyConn     net.Conn //外网连接
	StatusMonitor func(id uint64, status int)
}

// 读取内网服务器上行数据
func (ic *InnerConnection) Read() {
	for {
		if ic.Status == StatusProxy || ic.Status == StatusClose {
			return
		}
		buf := make([]byte, event.PackageLength)
		n, err := ic.Conn.Read(buf)
		if err != nil {
			log.Println("read from inner server connection error:" + err.Error())
			ic.Close()
			return
		}
		if n != event.PackageLength {
			log.Println("read from inner server connection error data length error")
			ic.Close()
			return
		}
		signal := &event.Signal{}
		err = json.Unmarshal(buf, signal)
		if err != nil {
			log.Println("cant not parse inner server signal data:" + string(buf) + " error：" + err.Error())
			ic.Close()
			return
		}
		switch signal.T {
		case event.Ping:
			// do nothing
		case event.Auth:
			if strings.TrimRight(signal.Ext, " ") != ic.AuthKey {
				log.Println("auth key " + ic.AuthKey + " != " + signal.Ext)
				ic.Close()
				return
			}
			ic.Status = StatusOk
			ic.StatusMonitor(ic.Id, StatusOk)
		case event.StartProxy:
			ic.Status = StatusProxy
			ic.StatusMonitor(ic.Id, StatusProxy)
			ic.startProxy()
			return
		default:
			log.Println("unknown inner server event:" + strconv.Itoa(signal.T))
			ic.Close()
			return
		}
	}
}

// 开始转发
func (ic *InnerConnection) ProxyRequest(conn net.Conn) {
	ic.proxyConn = conn
	_, err := ic.Conn.Write(event.GenerateSignal(event.StartProxy, ""))
	if err != nil {
		log.Println("send request fail:" + err.Error())
		ic.Close()
		return
	}
	time.AfterFunc(time.Second*5, func() {
		if ic.Status != StatusProxy && ic.Status != StatusClose {
			log.Println("wait for inner service timeout")
			ic.Close()
		}
	})
}

// 开始转发
func (ic *InnerConnection) startProxy() {
	//log.Println("外部服务开始转发")
	ic.Conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	ic.Conn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	ic.proxyConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	ic.proxyConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	go func() {
		if ic.proxyConn != nil && ic.Conn != nil {
			_, err := io.Copy(ic.proxyConn, ic.Conn)
			if err != nil {
				log.Println(err.Error())
			}
		}
		if ic.proxyConn != nil {
			ic.proxyConn.Close()
		}
	}()
	go func() {
		if ic.Conn != nil && ic.proxyConn != nil {
			_, err := io.Copy(ic.Conn, ic.proxyConn)
			if err != nil {
				log.Println(err.Error())
			}
		}
		if ic != nil {
			ic.Close()
		}
	}()
}

// 关闭连接
func (ic *InnerConnection) Close() {
	ic.Status = StatusClose
	ic.StatusMonitor(ic.Id, StatusClose)
	if ic.Conn != nil {
		ic.Conn.Close()
		ic.Conn = nil
	}
	if ic.proxyConn != nil {
		ic.proxyConn.Close()
		ic.proxyConn = nil
	}
}
