package main

import (
	"bigEventProject/rpcProject/registry"
	"net"
	"net/http"
)

//启动注册中心,9091

//web获取服务列表GET，服务心跳检查POST
var registryAddr = "http://localhost:9091/krpc_/registry"

func startRegistry() {
	l, _ := net.Listen("tcp", ":9091")
	registry.DefaultKRegister.HandleHTTP("/krpc_/registry")

	//第二个参数传nil表示使用DefaultServeMux
	_ = http.Serve(l, nil)
}

func main() {
	ch := make(chan int, 0)

	startRegistry()
	<-ch

}
