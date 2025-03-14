package codec

import (
	"io"
)

//客户端给ServiceMethod和Seq赋值，Error置空
//服务端读取Header失败会直接关闭连接,过程中出错则将错误值写入Error并把该结构体返回给客户端

type Header struct {
	// format "Service.Method",Service为某一类型，通常为结构体，
	//Method为该Service实现的某一个方法
	ServiceMethod string

	Seq   uint64 // sequence number chosen by client  请求序号
	Error string
}

//对消息体进行编解码的接口 Codec

type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

//抽象出 Codec 的构造函数，客户端和服务端可以通过 Codec 的 Type 得到构造函数

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemented
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec

}
