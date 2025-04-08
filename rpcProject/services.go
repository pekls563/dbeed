package rpcProject

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

//服务端使用反射处理参数并调用本地方法获取返回值

//存储结构体所实现的一个个方法的信息

type methodType struct {
	method    reflect.Method //方法本身
	ArgType   reflect.Type   //第一个参数的类型
	ReplyType reflect.Type   //第二个参数的类型
	numCalls  uint64         //统计方法调用次数
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// arg may be a pointer type, or a value type

	if m.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType).Elem()

	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// reply must be a pointer type

	replyv := reflect.New(m.ReplyType.Elem())

	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

//service主要存储传入的类型的相关信息

type service struct {
	name   string                 //类型名称
	typ    reflect.Type           //类型的实例的指针的reflect.Type对象
	rcvr   reflect.Value          //该类型的实例的指针
	method map[string]*methodType //该类型实现的所有方法,必须是可注册的方法
}

//rcvr接收一个指针

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)

	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)

	//该类型必须是可导出的
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	//注册该类型实现的所有可导出的方法
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)

	//s.typ.NumMethod():s.typ类型实现的方法数量
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		//NumIn()获取参数数量
		//每个方法固定3个参数，1个error类型的返回值
		//3个参数中第一个是注册的类型实例,第二个用于存储调用rpc的参数args，第三个用于存储调用rpc之后得到的返回值reply
		//例子：func (f FPP) Sum(args Empty, reply *Replys) error {}
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		//Out()返回值的类型
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		//将符合条件的方法加入到map中
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)
	}
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	//类型t必须是基本类型或者可导出的自定义类型

	//官方库int string,bool等满足t.PkgPath() == "",自定义类型可能不满足
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	f := m.method.Func

	//调用方法
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})

	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
