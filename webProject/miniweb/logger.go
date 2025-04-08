package miniweb

import (
	"log"
	"time"
)

//默认的日志中间件

func Logger() HandlerFunc {
	return func(c *Context) {

		t := time.Now()

		c.Next()

		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}
