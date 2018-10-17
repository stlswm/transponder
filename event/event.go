package event

const Ping = 0                //ping
const NewConnection = 1       //申请新连接
const Auth = 2                //授权
const RegisterCommunicate = 3 //注册通讯连接

// 内网通讯信号
type Signal struct {
	T   int    //信号类型
	Ext string //附件信息
}
