package miniweb

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

type HandlerFunc func(*Context)

//Group分组控制

type RouterGroup struct {
	prefix      string        //分组的前缀
	middlewares []HandlerFunc // 中间件,一般以组为单位用中间件
	parent      *RouterGroup  //
	engine      *Engine       //
}

// Engine implement the interface of ServeHTTP
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup

	//html模板
	htmlTemplates *template.Template // for html render  将所有的模板加载进内存
	funcMap       template.FuncMap   //所有的自定义模板渲染函数
}

//下面两个方法用于设置Engine的html模板属性

func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHTMLGlob(pattern string) {

	//ParseGlob()设置模板文件的路径

	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

//相比New(),多加了2个中间件

func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

// New is the constructor of miniweb.Engine
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}

	return engine
}

func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

//GET            /hello/:name
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

//注意*RouterGroup内嵌于Engine中，所以调用Engine.GET(...)等价于调用Engine.*RouterGroup.GET(...)
//调用Engine.GET(...),执行group.addRoute(...),group.engine引用到group所属的那个Engine结构体,调用engine.router.addRouter(...)

//r.GET("/hello/:name", func(c *miniweb.Context)

// GET defines the method to add GET request
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// PUT defines the method to add PUT request
func (group *RouterGroup) PUT(pattern string, handler HandlerFunc) {
	group.addRoute("PUT", pattern, handler)
}

// PATCH defines the method to add PATCH request
func (group *RouterGroup) PATCH(pattern string, handler HandlerFunc) {
	group.addRoute("PATCH", pattern, handler)
}

// DELETE defines the method to add DELETE request
func (group *RouterGroup) DELETE(pattern string, handler HandlerFunc) {
	group.addRoute("PUT", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

//中间件的use函数

func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

//接收到请求后调用ServeHTTP
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	var middlewares []HandlerFunc
	for _, group := range engine.groups {

		//当group.prefix=""时也会判定通过,"/v1/v2"组会先调用""组的中间件,再调用"/v1"组的中间件,最后调用"/v1/v2"组的中间件
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}

	//初始化context,
	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c) //从map中寻找对应的函数进行处理
}

//模板Template

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	//如果用户访问/v1/assets/js/kls.js

	//absolutePath=/v1/assets
	//fs=/usr/kls/blog/static

	absolutePath := path.Join(group.prefix, relativePath) //将两个文件路径组合成一个文件路径,总的请求路径

	//对于用户请求/v1/assets/js/kls.js,
	//fileServer视作请求/usr/kls/blog/static/js/kls.js
	//其中js/kls.js部分由c.Param("filepath")得到
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs)) //将路径中absolutePath部分替换成fs部分
	return func(c *Context) {

		//file=js/kls.js
		file := c.Param("filepath")

		//fileServer的工作目录为/usr/kls/blog/static
		//在/usr/kls/blog/static/下寻找file

		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// serve static files
//将本地文件路径映射为请求路径
//r.Static("/assets", "/usr/kls/blog/static")

func (group *RouterGroup) Static(relativePath string, root string) {
	handler := group.createStaticHandler(relativePath, http.Dir(root)) //将root转化为Dir类型，因为Dir实现了http.FileSystem接口
	//假设group.prefix=/v1  relativePath=/assets
	//则urlPattern=/assets/*filePath
	urlPattern := path.Join(relativePath, "/*filepath")
	// 请求路径为/v1/assets/*filePath

	//如果用户访问/v1/assets/js/kls.js,则filePath=js/kls.js
	group.GET(urlPattern, handler)
}
