package myredis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"

	"time"
)

var Redisclient *redis.Client

//初始化redis
func InitRedis() {

	redisUrl := "127.0.0.1:6379"

	Redisclient = redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ping := Redisclient.Ping(context.Background())
	/*fmt.Println(ping.String())
	fmt.Println("redis连接成功")
	fmt.Println("-------------------------------------------------------------------")*/
	//log.Logger.Info(time.Now().String() + "redis连接:" + ping.String())
	log.Println(time.Now().String() + "redis连接:" + ping.String())

}
