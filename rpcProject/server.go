package rpcProject

import (
	"bigEventProject/rpcProject/codec"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

//服务端通信逻辑

const MagicNumber = 0x3bef5c

//| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
//| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
//在整个通信流程中，客户端只需发送一次Option信息，后续连续发送多次Header+Body

//type Header struct {
//	// format "Service.Method",Service为某一类型，通常为结构体，
//	//Method为该Service实现的某一个方法
//	ServiceMethod string
//
//	Seq   uint64 // sequence number chosen by client  请求序号
//	Error string
//}

//通信配置信息

type Option struct {
	MagicNumber int //辨识码，用于服务端校验

	//协商确定的消息编解码的方式
	CodecType codec.Type

	ConnectTimeout time.Duration //创建连接超时
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
}

type Server struct {
	//服务端在通信过程中会频繁访问serviceMap,因此选择并发安全的sync.Map
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }

var Gopool *Pool

func (server *Server) Accept(lis net.Listener) {
	Gopool = NewPool(10)
	Gopool.Run()

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}

		//一个协程处理一个连接
		//go server.ServeConn(conn)
		Gopool.AddTask(func() {

			server.ServeConn(conn)
		})

	}
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {

	//校验配置信息Option,校验错误则关闭连接
	defer func() { _ = conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}

	//根据配置信息获取对应的编解码器的构造函数
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled
	for {

		req, err := server.readRequest(cc)
		if err != nil {

			//Header读取失败，可能是客户端编码方式不对也可能是数据包损坏，直接关闭连接
			if req == nil {
				break
			}

			//Header读取成功,但是解析Header.ServiceMethod失败或者读取Body失败
			//将错误信息发送回客户端
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		//go server.handleRequest(cc, req, sending, wg, 10*time.Second)
		Gopool.AddTask(func() {

			//server.ServeConn(conn)
			server.handleRequest(cc, req, sending, wg, 10*time.Second)
		})
	}
	wg.Wait()
	_ = cc.Close()
}

// request stores all information of a call
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv存储用于调用rpc的参数，replyv用于存储调用rpc获得的返回值

	mtype *methodType
	svc   *service
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {

	h, err := server.readRequestHeader(cc)
	//Header读取失败
	if err != nil {
		return nil, err
	}

	//解析Header中的ServiceMethod
	req := &request{h: h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}

	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	// make sure that argvi is a pointer, ReadBody need a pointer as parameter
	argvi := req.argv.Interface()

	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	//读取Body
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {

	defer wg.Done()

	called := make(chan struct{})
	sent := make(chan struct{})

	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-sent
		return
	}

	select {
	case <-time.After(timeout):
		//超时了，仍然要sendResponse
		req.h.Error = fmt.Sprintf("rpc server: request handler timeout: expect within %s", timeout)
		server.sendResponse(cc, req.h, invalidRequest, sending)

	case <-called: //req.svc.call 10s内执行成功则不进入超时处理,继续sendResponse
		<-sent
	}

}

func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)

	//加入serviceMap中
	//dup=true表示键已经存在,返回现有的值
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

// Register publishes the receiver's methods in the DefaultServer.

func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {

	//查找"."在字符串serviceMethod中最后一次出现的位置(首字符的下标）
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}

	//切割字符串，获取类型名和方法名
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}
