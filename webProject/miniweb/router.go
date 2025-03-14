package miniweb

import (
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// Only one * is allowed
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	//  pattern="/hello/*name/a/b/c"  parts=["hello","*name"]
	//  pattern="/hello/name/a/b/c"   parts=["hello","name","a","b","c"]
	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	//
	return parts
}

//                        GET                 /hello/:name
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	parts := parsePattern(pattern)

	//key=GET-/hello/:name
	key := method + "-" + pattern
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}
	//为GET方法的前缀树添加节点，height=0表示从根节点开始添加
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

//                        GET               /hello/a
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	//searchParts=["hello","a"]
	params := make(map[string]string)
	root, ok := r.roots[method] //获取GET方法的前缀树

	//该前缀树不存在，表示并没有给路由设置GET请求
	if !ok {
		return nil, nil
	}

	//height
	n := root.search(searchParts, 0)

	//n.pattern=/hello/*name

	if n != nil {
		parts := parsePattern(n.pattern) //parts=["hello","*name"]

		//以下操作使得params["hello"]="hello",account_srv_params["name"]="a"
		for index, part := range parts {
			if part[0] == ':' {
				//比如/:ta对应/hello
				//则令params["ta"]="hello"
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {

				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}

func (r *router) handle(c *Context) {

	//r.GET("/hello/:name", func(c *miniweb.Context) {
	//当请求/hello/a发生时
	n, params := r.getRoute(c.Method, c.Path)
	//n是最底层的孩子节点，params是动态路由映射表map
	//n.pattern=/hello/*name
	if n != nil {
		c.Params = params                 //将params加入到context中
		key := c.Method + "-" + n.pattern //key="GET-/hello/*name"

		//r.handlers[key](c)
		//将处理请求的函数加入到context.[]HandlerFunc中
		//注意这里是在处理请求阶段加入的，而其他中间件在设置路由时就加入了，因此能保证
		//所有中间件在处理请求的函数执行前执行
		c.handlers = append(c.handlers, r.handlers[key])
	} else {

		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}

	c.Next() //使得index从-1变成0，开始第一个中间件的执行
}
