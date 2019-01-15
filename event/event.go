package event

import (
	"fmt"
	"strings"
	"encoding/json"
)

const PackageLength = 48 //包长

const Ping = 0       //ping
const Auth = 2       //授权
const StartProxy = 6 //开始转发工作

// 内网通讯信号每
type Signal struct {
	T   int    //信号类型
	Ext string //附件信息
}

// 将字符串填充成32个长度
func StringFormat32(raw string) string {
	l := len(raw)
	switch {
	case l == 32:
		return raw
	case l > 32:
		return fmt.Sprintf("%.32s", raw[:32])
	case l < 32:
		return fmt.Sprintf("%s%s", raw, strings.Repeat(" ", 32-l))
	}
	return raw
}

// 生成信息字节（总长度XX）
func GenerateSignal(t int, ext string) []byte {
	ext = StringFormat32(ext)
	s := &Signal{
		T:   t,
		Ext: ext,
	}
	sb, _ := json.Marshal(s)
	return sb
}
