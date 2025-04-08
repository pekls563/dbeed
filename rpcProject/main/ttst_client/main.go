package main

import (
	"bigEventProject/rpcProject"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Foo int

type Args struct {
	Num1 int
	Num2 int
}

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func main() {
	log.SetFlags(0)

	client, _ := rpcProject.Dial("tcp", ":9091")
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)

	start := time.Now()
	var wg sync.WaitGroup
	//500w单机并发请求
	for i := 0; i < 5000000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i}
			var reply int
			if err := client.Call(context.Background(), "Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()

	// 计算耗时
	duration := time.Since(start)
	fmt.Printf("代码运行时间: %v\n", duration)
}
