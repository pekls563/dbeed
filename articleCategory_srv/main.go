package main

import (
	"bigEventProject/articleCategory_srv/biz"
	"bigEventProject/articleCategory_srv/internal"
	"bigEventProject/articleCategory_srv/model"
	"bigEventProject/articleCategory_srv/myredis"
	"bigEventProject/rpcProject"
	"bigEventProject/rpcProject/registry"
	"context"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"log"
	"net"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"time"
)

//articleCategory服务端,随机可用端口

func init() {
	internal.InitDB()
	myredis.InitRedis()
}

func startServer(registryAddr string, name string) {
	var articleCategoryServer biz.ArticleCategoryServer
	l, _ := net.Listen("tcp", ":0")
	server := rpcProject.NewServer()
	_ = server.Register(&articleCategoryServer)
	//registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	registry.Heartbeat(registryAddr, l.Addr().String(), 0, name)

	fmt.Println("articleCategory_srv启动在: ", l.Addr().String())

	server.Accept(l)
}

func main() {

	//rocketmq模拟文章标题未过审

	rocketmqUrl := "127.0.0.1:10909"
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("wei_guo_shen1"),
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{rocketmqUrl})),
	)
	if err != nil {
		panic(err)
	}
	err = c.Subscribe("wei_guo_shen", consumer.MessageSelector{},
		func(ctx context.Context, ext ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			for i := range ext {
				//fmt.Printf("订阅消息，消费%v \n", ext[i])
				//fmt.Println("----------------------------------------------------------------------------------------------------------------------")

				if ext[i].Body != nil {
					s := string(ext[i].Body)
					r := internal.DB.Where(&model.Article{Title: s}).Delete(&model.Article{})
					if r.RowsAffected < 1 {
						//log.Logger.Error(time.Now().String() + "删除未过审文章失败")
						log.Println(time.Now().String() + "删除未过审文章失败")
					}

				}
			}
			return consumer.ConsumeSuccess, nil
		})
	if err != nil {
		//log.Logger.Error(time.Now().String() + "消费消息错误" + err.Error())
		log.Println(time.Now().String() + "消费消息错误" + err.Error())
	}

	go func() {

		err = c.Start()
		if err != nil {
			//log.Logger.Error(time.Now().String() + "开启消费者错误" + err.Error())
			log.Println(time.Now().String() + "开启消费者错误" + err.Error())
		}
	}()

	ch := make(chan int, 0)
	registryAddr := "http://localhost:9091/krpc_/registry"
	serveName := "articleCategory_srv"
	startServer(registryAddr, serveName)
	<-ch

	//--------------------------------------------------------------------------------------------------
	//fmt.Println(fmt.Sprintf("%s启动在%d", randUUID, port))

}
