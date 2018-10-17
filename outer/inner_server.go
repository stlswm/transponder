package outer

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"transponder/event"
)

// 内部服务对象
type InnerServer struct {
	Address            string        //监听地址
	AuthKey            string        //连接授权码
	tmpConn            sync.Map      //临时连接（存放没授权的连接）
	communicateConn    net.Conn      //内网通讯连接
	communicateReadBuf string        //内网通讯连接读缓存
	transmitQueue      chan net.Conn //待转发连接池
}

// 内部连接对象
type InnerConn struct {
	id      uint64   //连接id
	created int64    //创建时间
	conn    net.Conn //连接对象
	readBuf string   //读缓存
}

// 启动服务
func (i *InnerServer) StartServer() {
	log.Println("启动内部服务器服务，" + i.Address)
	addrSlice := strings.Split(i.Address, "://")
	if len(addrSlice) < 2 {
		panic(i.Address + " format error.")
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
	go i.overdueCheck()
	go i.communicateRead()
	var id uint64 = 0
	for {
		id = i.generateConnId(id)
		tcpConn, _ := listener.AcceptTCP()
		innerConn := &InnerConn{
			id:      id,
			created: time.Now().Unix(),
			conn:    tcpConn,
		}
		i.tmpConn.Store(id, innerConn)
		go i.Read(innerConn)
	}
}

// 连接id生成
func (i *InnerServer) generateConnId(id uint64) uint64 {
	id++
	if id > 4294967296 {
		id = 1
	}
	_, ok := i.tmpConn.Load(id)
	if !ok {
		return id
	}
	id = i.generateConnId(id)
	return id
}

// 过期连接检查
func (i *InnerServer) overdueCheck() {
	t := time.NewTicker(time.Second * 5)
	for {
		<-t.C
		timeNow := time.Now().Unix()
		i.tmpConn.Range(func(key, value interface{}) bool {
			innerConn := value.(*InnerConn)
			if timeNow-innerConn.created > 10 {
				//超时没有认证的连接关闭并释放资源
				log.Println(innerConn.conn.RemoteAddr().String() + " auth timeout now closing")
				innerConn.conn.Close()
				i.tmpConn.Delete(key)
			}
			return true
		})
	}
}

// 读取内网连接数据
func (i *InnerServer) Read(innerConn *InnerConn) {
	for {
		buf := make([]byte, 512)
		n, err := innerConn.conn.Read(buf)
		if err != nil {
			log.Println(err.Error())
			innerConn.conn.Close()
			i.tmpConn.Delete(innerConn.id)
			return
		}
		innerConn.readBuf = string(buf[0:n])
		for {
			pos := strings.IndexAny(innerConn.readBuf, "\r")
			if pos == -1 {
				break
			}
			nowPackage := innerConn.readBuf[0 : pos+1]
			nowPackage = strings.TrimRight(nowPackage, "\r")
			innerConn.readBuf = innerConn.readBuf[pos+1:]
			signal := &event.Signal{}
			err = json.Unmarshal([]byte(nowPackage), signal)
			if err != nil {
				log.Println("无法解析内部服务器上传数据：" + nowPackage + " 错误信息：" + err.Error())
				innerConn.conn.Close()
				i.tmpConn.Delete(innerConn.id)
				return
			}
			switch signal.T {
			case event.Auth:
				if signal.Ext != i.AuthKey {
					log.Println("连接：" + innerConn.conn.RemoteAddr().String() + "授权失败（" + i.AuthKey + "!=" + signal.Ext + "）")
					innerConn.conn.Close()
					i.tmpConn.Delete(innerConn.id)
					return
				}
				//授权成功进程待转发队列
				i.tmpConn.Delete(innerConn.id)
				i.transmitQueue <- innerConn.conn
				return
			case event.RegisterCommunicate:
				if signal.Ext != i.AuthKey {
					log.Println("连接：" + innerConn.conn.RemoteAddr().String() + "注册授权失败（" + i.AuthKey + "!=" + signal.Ext + "）")
					innerConn.conn.Close()
					i.tmpConn.Delete(innerConn.id)
					return
				}
				if i.communicateConn != nil {
					i.communicateConn.Close()
				}
				i.tmpConn.Delete(innerConn.id)
				i.communicateConn = innerConn.conn
				return
			}
			log.Println("未知事件类型：" + strconv.Itoa(signal.T))
			innerConn.conn.Close()
			i.tmpConn.Delete(innerConn.id)
			return
		}
	}
}

// 内部通讯服务连接读取
func (i *InnerServer) communicateRead() {
	for {
		if i.communicateConn == nil {
			time.Sleep(time.Second * 3)
			continue
		}
		buf := make([]byte, 512)
		n, err := i.communicateConn.Read(buf)
		if err != nil {
			log.Println("通讯服务连接异常：" + err.Error())
			i.communicateConn.Close()
			i.communicateConn = nil
			continue
		}
		i.communicateReadBuf = string(buf[0:n])
		for {
			pos := strings.IndexAny(i.communicateReadBuf, "\r")
			if pos == -1 {
				break
			}
			nowPackage := i.communicateReadBuf[0 : pos+1]
			nowPackage = strings.TrimRight(nowPackage, "\r")
			i.communicateReadBuf = i.communicateReadBuf[pos+1:]
			signal := &event.Signal{}
			err = json.Unmarshal([]byte(nowPackage), signal)
			if err != nil {
				log.Println("无法解析内部服务器通讯连接上行数据：" + nowPackage + " 错误信息：" + err.Error())
				i.communicateConn.Close()
				i.communicateConn = nil
				continue
			}
			switch signal.T {
			case event.Ping:
				//nothing
			default:
				log.Println("内部服务器通讯连接未知事件类型：" + strconv.Itoa(signal.T))
			}
		}
	}
}

// 与内网服务器通讯
func (i *InnerServer) communicate(single int) error {
	if i.communicateConn == nil {
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
	default:
		return errors.New("不支持的信号类型")
	}
	sByte, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = i.communicateConn.Write([]byte(string(sByte) + "\r"))
	if err == nil {
		return nil
	}
	i.communicateConn.Close()
	i.communicateConn = nil
	return err
}

// 新连接请求
func (i *InnerServer) NewClientRequest() (error, net.Conn) {
	err := i.communicate(event.NewConnection)
	if err == nil {
		select {
		case conn := <-i.transmitQueue:
			return nil, conn
		case <-time.After(10 * time.Second): //超时10秒
			return errors.New("get connection timeout"), nil
		}
	}
	return err, nil
}

// 获取内部服务实例
func NewInnerServer() *InnerServer {
	i := &InnerServer{
		transmitQueue: make(chan net.Conn, 10240),
	}
	return i
}
