package main

import (
	"bigEventProject/account_srv/biz"
	"bigEventProject/account_srv/internal"
	"bigEventProject/rpcProject"
	"bigEventProject/rpcProject/registry"
	"fmt"
	"net"
)

//account 服务端 随机的可用端口

func init() {
	internal.InitDB()
}

func startServer(registryAddr string, name string) {
	var account_server biz.AccountServer
	l, _ := net.Listen("tcp", ":0")
	server := rpcProject.NewServer()
	_ = server.Register(&account_server)
	//registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	registry.Heartbeat(registryAddr, l.Addr().String(), 0, name)

	fmt.Println("account_srv启动在: ", l.Addr().String())

	server.Accept(l)
}

func main() {

	ch := make(chan int, 0)
	registryAddr := "http://localhost:9091/krpc_/registry"
	serveName := "account_srv"
	startServer(registryAddr, serveName)
	<-ch

}
