package rpcProject

import (
	"bigEventProject/rpcProject/codec"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

//seq 用于给发送的请求编号，每个请求拥有唯一编号

type Call struct {
	Seq           uint64      //给发送的请求编号
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // arguments to the function
	Reply         interface{} // reply from the function
	Error         error       // if error occurs, it will be set

	Done chan *Call
}

//为支持异步调用，会调用 call.done() 通知调用方

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc  codec.Codec //消息的编解码器
	opt *Option

	sending sync.Mutex //确保Client向服务端发送的消息是一条一条发送的，防止多条消息混杂在一起发送

	header codec.Header
	mu     sync.Mutex //修改Client的成员属性时使用此锁

	//给发送的请求编号,每次来一个Call,使用Call.seq=Client.seq给Call编号并
	//让Client.seq++
	seq uint64

	pending map[uint64]*Call //存储未处理完的请求

	//closing 和 shutdown 任意一个值置为 true，则表示 Client 处于不可用的状态，但有些许的差别，
	//closing 是用户主动关闭的，即调用 Close 方法，而 shutdown 置为 true 一般是有错误发生
	//
	closing  bool // 客户端关闭连接
	shutdown bool // 服务端关闭连接
}

//静态检查Client是否实现了io.Closer接口

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// Close the connection
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable return true if the client does work
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}

	//client.seq是下一个要registerCall的call的Seq

	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header

		if err = client.cc.ReadHeader(&h); err != nil {
			//fmt.Println(err)
			//若服务端关闭net.Conn，进入此分支
			break
		}
		call := client.removeCall(h.Seq)
		//call := client.removeCall(1)
		switch {
		case call == nil:

			//这个分支在外层调用ctx.Done()后执行removeCall(),导致本函数receive()中执行的removeCall()返回nil
			//ctx.Done()的原因可能是Call超时等，不包括Dial()超时

			err = client.cc.ReadBody(nil)

		case h.Error != "":
			//若服务端给h.Error赋值并发给了客户端，则进入此分支
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}

	client.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}

	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}

	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}

	go client.receive()
	return client
}

func parseOptions(opts ...*Option) (*Option, error) {

	//客户端程序不添加Option，则使用默认的Option

	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber

	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()

	//未限制创建连接超时
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}

	select {
	//10s内未读取并解析成功option返回错误
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

// Dial connects to an RPC server at the specified network address
func Dial(network, address string, opts ...*Option) (client *Client, err error) {

	//return NewClient(conn, opt)返回的client为nil时执行

	//Dial流程执行完毕client没有初始化成功则关闭net.Conn连接,之后
	//客户端的net.Conn连接没有其他关闭途径，需要在客户端的main函数中调用client.Close()手动关闭

	return dialTimeout(NewClient, network, address, opts...)
}

func (client *Client) send(call *Call) {
	// make sure that the client will send a complete request
	client.sending.Lock()
	defer client.sending.Unlock()

	//如果服务端解析Option出错，服务端关闭net.Conn,那么
	//客户端在Dial()时开出了一个协程进行receive(),由于服务端关闭了net.Conn,客
	//户端receive()内部for循环内执行cc.ReadHeader()立即返回EOF错误,跳出for循环，
	//client.terminateCalls()执行,shutdown被置为true,然后客
	//户端在执行client.registerCall时返回ErrShutdown错误

	//客户端执行Dial()中的conn.Close()关闭连接

	// register this call.
	//本地给call编号

	seq, err := client.registerCall(call)
	if err != nil {

		//registerCall失败不用removeCall

		call.Error = err
		call.done()
		return
	}

	// prepare request header
	//将方法名,call的编号发送给服务端

	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	// encode and send the request

	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)

		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

//Go 和 Call 是客户端暴露给用户的两个 RPC 服务调用接口，Go 是一个异步接口，返回 call 实例。
//Call 是对 Go 的封装，阻塞 call.Done，等待响应返回，是一个同步接口

// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation.
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {

	//拒绝为nil或者未分配缓存空间的chan *Call

	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}

	//注意下面的方法func (client *Client) Call
	//主要是为了下面call结构体的生成

	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done, //注意通道是引用传递
	}
	client.send(call)
	return call
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.

//请求处理后call.Done <- call,就可以从call.Done中获取返回的信息了

//每来一次请求都要make(chan *Call, 1),这个通道感觉没必要？？？？？？

//关于异步调用，客户端Dial()成功后开5个协程向服务端发送5个请求,这5个请求先在本地编号再发给服务端，由于
//服务端对每个请求解析成功后都开一个协程handleRequest,因此这5个请求在服务端处理完成并发送的顺序可能不一样,
//客户端只需在receive里找到对应的call编号并调用call.done来结束一个call的rpc流程
//所以在这里chan *Call通道实现了异步调用，使用return不行，因为需要等待服务端返回

//客户端需要发送请求的参数(一个结构体指针)和返回值（一个指针），而这些东西的类型是不确定的，因此使用interface{}接收

//ctx用于将超时处理(或者其他处理，这些处理都在传入的ctx中)的控制权交给用户,控制更为灵活

func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {

	//call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	//return call.Error

	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}

}
