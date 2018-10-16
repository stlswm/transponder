package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"strings"
	"time"
	"ConfigAdapter/JsonConfig"
)

// 外网服务器关系维护
type OuterHolder struct {
	CommunicateAddress    string
	communicateConn       net.Conn
	communicateReadBuffer string
	ServerAddress         string
	ProxyAddress          string
}

// 启动
func (o *OuterHolder) Start() {
	o.connection()
	o.read()
}

// 重新启动
func (o *OuterHolder) restart() {
	if o.communicateConn != nil {
		o.communicateConn.Close()
		o.communicateConn = nil
	}
	o.connection()
}

// 连接到服务器
func (o *OuterHolder) connection() {
	log.Println("连接外网通讯服务器，" + o.CommunicateAddress)
	sc, err := net.Dial("tcp", o.CommunicateAddress)
	if err != nil {
		//连接错误3秒后重新连接
		log.Println("连接外网通讯服务器错误3秒后重新连接")
		log.Println(err.Error())
		time.AfterFunc(time.Second*3, func() {
			o.connection()
		})
		return
	}
	o.communicateReadBuffer = ""
	o.communicateConn = sc
}

// 读取主服务器数据
func (o *OuterHolder) read() {
	for {
		if o.communicateConn == nil {
			time.Sleep(time.Second * 3)
			continue
		}
		buf := make([]byte, 512)
		n, err := o.communicateConn.Read(buf)
		if err != nil {
			log.Println("read from remote error:" + err.Error())
			o.restart()
		}
		o.communicateReadBuffer = string(buf[0:n])
		for {
			pos := strings.IndexAny(o.communicateReadBuffer, "\r")
			if pos == -1 {
				break
			}
			nowPackage := o.communicateReadBuffer[0 : pos+1]
			nowPackage = strings.TrimRight(nowPackage, "\r")
			o.communicateReadBuffer = o.communicateReadBuffer[pos+1:]
			var nowPackageJson = struct {
				E int
			}{}
			err = json.Unmarshal([]byte(nowPackage), &nowPackageJson)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			switch nowPackageJson.E {
			case 1:
				//申请新连接
				o.newExchange()
			case 0:
				//ping
			}
		}
	}
}

// io 交换
func (o *OuterHolder) newExchange() {
	//log.Println("发起到外网服务器" + o.ServerAddress + "的连接")
	outConn, err := net.Dial("tcp", o.ServerAddress)
	if err != nil {
		log.Println(err.Error())
		return
	}
	outConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	outConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	//log.Println("发起到本地服务" + o.ProxyAddress + "的连接")
	proxyConn, err := net.Dial("tcp", o.ProxyAddress)
	if err != nil {
		outConn.Close()
		log.Println(err.Error())
		return
	}
	proxyConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	proxyConn.SetWriteDeadline(time.Now().Add(time.Second * 30))
	//log.Println("连接交换")
	go func() {
		io.Copy(outConn, proxyConn)
		proxyConn.Close()
		//log.Println("关闭到转发地址的连接")
	}()
	go func() {
		io.Copy(proxyConn, outConn)
		outConn.Close()
		//log.Println("关闭到外部服务器的连接")
	}()
}

type Config struct {
	CommunicateAddress string
	ServerAddress      string
	ProxyAddress       string
}

func main() {
	c := &Config{}
	JsonConfig.Load("config.json", c)
	//配置文件地址格式检查
	addrSlice := strings.Split(c.CommunicateAddress, "://")
	if len(addrSlice) < 2 {
		panic(c.CommunicateAddress + " format error.")
	}
	if addrSlice[0] != "tcp" {
		panic("address to communicate only support tcp.")
	}
	communicateAddress := addrSlice[1]
	addrSlice = strings.Split(c.ServerAddress, "://")
	if len(addrSlice) < 2 {
		panic(c.ServerAddress + " format error.")
	}
	if addrSlice[0] != "tcp" {
		panic("address to server only support tcp.")
	}
	serverAddress := addrSlice[1]
	addrSlice = strings.Split(c.ProxyAddress, "://")
	if len(addrSlice) < 2 {
		panic(c.ProxyAddress + " format error.")
	}
	if addrSlice[0] != "tcp" {
		panic("proxy address only support tcp.")
	}
	proxyAddress := addrSlice[1]
	h := &OuterHolder{
		CommunicateAddress: communicateAddress,
		ServerAddress:      serverAddress,
		ProxyAddress:       proxyAddress,
	}
	h.Start()
}
