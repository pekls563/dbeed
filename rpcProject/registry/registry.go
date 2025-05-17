package registry

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

//一个简单的注册中心
type kRegistry struct {
	timeout time.Duration
	mu      sync.Mutex // protect following
	servers map[string]([]*ServerItem)
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/krpc_/registry"
	defaultTimeout = time.Minute * 5 //任何注册的服务 5 min内未收到心跳检查，即视为不可用状态。
)

// New create a registry instance with timeout setting
func New(timeout time.Duration) *kRegistry {

	return &kRegistry{
		servers: make(map[string][]*ServerItem),
		timeout: timeout,
	}
}

var DefaultKRegister = New(defaultTimeout)

//添加服务实例，如果服务已经存在，则更新 start。start为服务注册时的时间
func (r *kRegistry) putServer(addr string, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[name]
	if s == nil {

		t := make([]*ServerItem, 0)
		t = append(t, &ServerItem{Addr: addr, start: time.Now()})
		r.servers[name] = t

	} else {
		for _, v := range r.servers[name] {
			if (*v).Addr == addr {
				(*v).start = time.Now()
				return
			}
		}
		r.servers[name] = append(r.servers[name], &ServerItem{Addr: addr, start: time.Now()})

	}
}

//返回可用的服务列表，如果存在超时的服务，则删除。

func (r *kRegistry) aliveServers(name string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	tmp := make([]*ServerItem, 0)
	for _, s := range r.servers[name] {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, s.Addr)
			tmp = append(tmp, s)
		}

	}
	r.servers[name] = tmp
	//排序
	sort.Strings(alive)
	return alive
}

//注册中心的ServeHTTP,用于和客户端，服务端交互
func (r *kRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {

	//"GET"分支,给客户端返回可用的服务列表
	case "GET":

		// keep it simple, server is in req.Header
		serveName := req.URL.Query().Get("Mini-Get-Serve-Name")
		//serveName := req.Header.Get("Mini-Get-Serve-Name")
		if serveName == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Println("获取到客户端的GET请求: ", serveName)

		w.Header().Set("Mini-Servers", strings.Join(r.aliveServers(serveName), ","))
	case "POST":
		// keep it simple, server is in req.Header
		//获取发送心跳检查的服务的地址

		addr := req.Header.Get("Mini-Serve-Addr")
		name := req.Header.Get("Mini-Serve-Name")
		if addr == "" || name == "" {
			fmt.Println("POST请求非法")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Println("获取到服务端的POST请求: ", name, " ,", addr)
		//更新服务，可能添加服务，也可能刷新服务的存活时间
		r.putServer(addr, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *kRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r) //对应路径调用r.ServeHTTP()
	log.Println("rpc registry path:", registryPath)
}

func Heartbeat(registry, addr string, duration time.Duration, name string) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	//一开始发送一次心跳检查，视为注册服务
	err = sendHeartbeat(registry, addr, name)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			//服务每隔 defaultTimeout-1min 向注册中心发送一次心跳检查
			<-t.C
			err = sendHeartbeat(registry, addr, name)
		}
	}()
}

func sendHeartbeat(registry, addr string, name string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	//registry="http://localhost:9999/_geerpc_/registry"
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("Mini-Serve-Addr", addr)
	req.Header.Set("Mini-Serve-Name", name)
	//忽略注册中心返回的http响应结果
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
