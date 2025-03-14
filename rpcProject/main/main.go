package main

import (
	"bigEventProject/rpcProject"
	"bigEventProject/rpcProject/dclient"
	"bigEventProject/rpcProject/registry"
	"context"
	"log"
	"net"
	"net/http"

	"sync"
	"time"
)

type Empty struct {
}
type Foo int

type Item struct {
	Name string
}

type Replys struct {
	ItemList []*Item
}

func (f Foo) Sum(args Empty, reply *Replys) error {

	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "666"})
	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "777"})
	return nil
}

type Goo int

func (f Goo) Del(args Empty, reply *Replys) error {

	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "777"})
	(*reply).ItemList = append((*reply).ItemList, &Item{Name: "888"})
	return nil
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.DefaultGeeRegister.HandleHTTP("/krpc_/registry")
	wg.Done()
	//第二个参数传nil表示使用DefaultServeMux
	_ = http.Serve(l, nil)
}

func startServer(registryAddr string, wg *sync.WaitGroup, name string) {
	var foo Foo
	var goo Goo
	l, _ := net.Listen("tcp", ":0")
	server := rpcProject.NewServer()
	switch name {
	case "serve1":
		_ = server.Register(&foo)
	case "serve2":
		_ = server.Register(&goo)

	}

	registry.Heartbeat(registryAddr, l.Addr().String(), 0, name)
	wg.Done()
	server.Accept(l)
}

func foo(xc *dclient.XClient, ctx context.Context, typ, serviceMethod string, args *Empty) {
	var reply Replys
	var err error

	err = xc.Call(ctx, serviceMethod, args, &reply)

	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success: %s ", typ, serviceMethod, (*((reply.ItemList)[1])).Name)
	}
}

func call(registry string, name string) {

	//初始化客户端
	d := dclient.NewKRegistryDiscovery(registry, 0, name)
	xc := dclient.NewXClient(d, dclient.RoundRobinSelect, nil)

	var method string
	switch name {
	case "serve1":
		method = "Foo.Sum"
	case "serve2":
		method = "Goo.Del"
	}

	if method == "" {
		log.Fatal("method为空")
	}

	defer func() { _ = xc.Close() }()

	//调用RPC
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", method, &Empty{})
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9999/krpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	//服务中心开启完毕，开启服务
	wg.Add(4)
	go startServer(registryAddr, &wg, "serve1")
	go startServer(registryAddr, &wg, "serve2")
	go startServer(registryAddr, &wg, "serve1")
	go startServer(registryAddr, &wg, "serve2")
	wg.Wait()

	time.Sleep(time.Second)
	//服务开启完毕，开启客户端调用服务

	//这里处理得简单了一些，注册中心对服务端和客户端暴露的
	//请求路径是一样的，注册中心处理客户端的GET请求给客户端返回服务列表
	//注册中心处理服务端的POST请求 注册服务并检查心跳
	call(registryAddr, "serve1")
	call(registryAddr, "serve2")

}
