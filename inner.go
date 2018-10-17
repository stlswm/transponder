package main

import (
	"ConfigAdapter/JsonConfig"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	"transponder/event"
)

// 外网服务器关系维护
type OuterHolder struct {
	RegisterAddress    string
	ProxyAddress       string
	AuthKey            string
	communicateConn    net.Conn
	communicateReadBuf string
}

// 启动
func (o *OuterHolder) Start() {
	o.registerCommunicate()
	go o.pingCommunicate()
	o.readCommunicate()
}

// 与外网服务器通讯
func (o *OuterHolder) communicate(single int) error {
	if o.communicateConn == nil {
		return errors.New("the connection between inner server and outer server is nil")
	}
	s := &event.Signal{}
	switch single {
	case event.Ping:
		//ping
		s.T = event.Ping
	case event.NewConnection:
		//新连接请求
		s.T = event.NewConnection
	case event.Auth:
		//授权
		s.T = event.Auth
		s.Ext = o.AuthKey
	case event.RegisterCommunicate:
		//注册为通讯服务连接
		s.T = event.RegisterCommunicate
		s.Ext = o.AuthKey
	default:
		return errors.New("不支持的信号类型")
	}
	sByte, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = o.communicateConn.Write([]byte(string(sByte) + "\r"))
	if err == nil {
		return nil
	}
	o.communicateConn.Close()
	o.communicateConn = nil
	return err
}

// 注册通讯连接
func (o *OuterHolder) registerCommunicate() {
	log.Println("连接外网通讯服务器，" + o.RegisterAddress)
	sc, err := net.Dial("tcp", o.RegisterAddress)
	if err != nil {
		//连接错误3秒后重新连接
		log.Println("连接外网通讯服务器错误:" + err.Error() + "3秒后重新连接")
		time.AfterFunc(time.Second*3, func() {
			o.registerCommunicate()
		})
		return
	}
	log.Println("成功连接外网通讯服务器")
	o.communicateConn = sc
	o.communicateReadBuf = ""
	err = o.communicate(event.RegisterCommunicate)
	if err != nil {
		//注册失败
		log.Println("发送注册包失败:" + err.Error() + "，正在关闭，3秒后重新连接")
		o.closeCommunicate()
		time.AfterFunc(time.Second*3, func() {
			o.registerCommunicate()
		})
		return
	}
}

// 关闭外网通讯连接
func (o *OuterHolder) closeCommunicate() {
	if o.communicateConn == nil {
		return
	}
	o.communicateConn.Close()
	o.communicateReadBuf = ""
	o.communicateConn = nil
}

// ping外网通讯连接
func (o *OuterHolder) pingCommunicate() {
	t := time.NewTicker(time.Second * 30)
	for {
		<-t.C
		o.communicate(event.Ping)
	}
}

// 读取外网通讯连接下行数据
func (o *OuterHolder) readCommunicate() {
	for {
		if o.communicateConn == nil {
			time.Sleep(time.Second * 3)
			continue
		}
		buf := make([]byte, 512)
		n, err := o.communicateConn.Read(buf)
		if err != nil {
			log.Println("通讯服务连接Read Error：" + err.Error() + "，正在关闭，3秒后重新连接")
			o.closeCommunicate()
			time.AfterFunc(time.Second*3, func() {
				o.registerCommunicate()
			})
			continue
		}
		o.communicateReadBuf = string(buf[0:n])
		for {
			pos := strings.IndexAny(o.communicateReadBuf, "\r")
			if pos == -1 {
				break
			}
			nowPackage := o.communicateReadBuf[0 : pos+1]
			nowPackage = strings.TrimRight(nowPackage, "\r")
			o.communicateReadBuf = o.communicateReadBuf[pos+1:]
			signal := &event.Signal{}
			err = json.Unmarshal([]byte(nowPackage), signal)
			if err != nil {
				log.Println("无法解析内部服务器通讯连接下行数据：" + nowPackage + " 错误信息：" + err.Error())
				o.communicateConn.Close()
				o.communicateConn = nil
				continue
			}
			switch signal.T {
			case event.Ping:
				//nothing
			case event.NewConnection:
				o.newExchange()
			default:
				log.Println("内部服务器通讯连接未知事件类型：" + strconv.Itoa(signal.T))
			}
		}
	}
}

// io 交换
func (o *OuterHolder) newExchange() {
	outConn, err := net.Dial("tcp", o.RegisterAddress)
	if err != nil {
		log.Println(err.Error())
		return
	}
	s := &event.Signal{}
	s.T = event.Auth
	s.Ext = o.AuthKey
	sByte, _ := json.Marshal(s)
	_, err = outConn.Write([]byte(string(sByte) + "\r"))
	if err != nil {
		outConn.Close()
		log.Println(err.Error())
		return
	}
	outConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	outConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	proxyConn, err := net.Dial("tcp", o.ProxyAddress)
	if err != nil {
		outConn.Close()
		log.Println(err.Error())
		return
	}
	proxyConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	proxyConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	go func() {
		io.Copy(outConn, proxyConn)
		proxyConn.Close()
	}()
	go func() {
		io.Copy(proxyConn, outConn)
		outConn.Close()
	}()
}

type InnerConfig struct {
	RegisterAddress string
	ProxyAddress    string
	AuthKey         string
}

func main() {
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
	h := &OuterHolder{
		RegisterAddress: registerAddress,
		ProxyAddress:    proxyAddress,
		AuthKey:         c.AuthKey,
	}
	h.Start()
}
