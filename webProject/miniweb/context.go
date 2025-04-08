package miniweb

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
)

//作为String,JSON,Data,HTML的解析数据源

//存储键值对
type H map[string]interface{}

//上下文context保留一些强相关信息，比如响应码StatusCode
//请求路径，方法以及底层的http.Request
//newContext设置了http.ResponseWriter(Writer),StatusCode以外的信息，而
//除newContext以外的其他方法设置http.ResponseWriter(Writer),StatusCode的信息

type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string

	Params map[string]string //动态路由映射表，记录包含：或者*的匹配过程
	// response info
	StatusCode int

	// middleware，中间件
	handlers []HandlerFunc
	index    int

	//存储engine的指针，方便调用engine的属性
	engine *Engine

	// This mutex protect Keys map
	mu sync.RWMutex

	// Keys is a key/value pair exclusively for the context of each request.
	Keys map[string]interface{}
}

func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}

	c.Keys[key] = value
	c.mu.Unlock()
}

func (c *Context) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.Keys[key]
	c.mu.RUnlock()
	return
}

func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path, //  /hello/a
		Method: req.Method,   //   GET
		index:  -1,
	}
}

//中间件的Next()

func (c *Context) Next() {
	c.index++
	s := len(c.handlers)
	for ; c.index < s; c.index++ {

		c.handlers[c.index](c)
	}
}

//中间件测试Fail

func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers) //发生错误，跳过所有中间件的执行，也跳过处理请求的函数的执行
	c.JSON(code, H{"message": err})
}

func (c *Context) Abort() {
	c.index = len(c.handlers) //发生错误，跳过所有中间件的执行，也跳过处理请求的函数的执行
	//c.JSON(code, H{"message": err})
}

func (c *Context) AbortWithStatus(code int) {
	c.Status(code)
	//c.Writer.WriteHeaderNow()
	//encoder := json.NewEncoder(c.Writer)
	//if err := encoder.Encode(obj); err != nil {
	//	http.Error(c.Writer, err.Error(), 500)
	//}
	c.Abort()
}

func (c *Context) PostForm(key string) string {
	return c.Req.FormValue(key) //?
}

func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

func (c *Context) Status(code int) {
	c.StatusCode = code

	//注意，一共有两个地方发送了http响应数据，第一个是WriteHeader(),发送已经设置好的响应头和状态码，第二个是json.NewEncoder().Encode,发送响应头
	c.Writer.WriteHeader(code)
	//WriteHeader用于设置响应码,需要在Header().Set()后调用
}

func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

func (c *Context) ShouldBindJSON(obj interface{}) error {

	// 检查目标是否是指针和非nil
	if reflect.TypeOf(obj).Kind() != reflect.Ptr || reflect.ValueOf(obj).IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	// 解析请求体中的JSON数据

	decoder := json.NewDecoder(c.Req.Body)
	if err := decoder.Decode(obj); err != nil {
		//	http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	return nil

}

//四种数据响应格式

func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

func (c *Context) JSON(code int, obj interface{}) {
	//
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {

		http.Error(c.Writer, err.Error(), 500)
		log.Println("JSON编码响应体报错" + err.Error())
	}
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	//c.Writer.Write([]byte(html))

	//根据模板文件名选择模板进行渲染。
	if err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Fail(500, err.Error())
	}
}
