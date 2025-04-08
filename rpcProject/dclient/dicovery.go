package dclient

import (
	"bigEventProject/rpcProject"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"reflect"

	"strings"

	"sync"
	"time"
)

//客户端支持负载均衡,服务发现

type KRegistryDiscovery struct {
	*MultiServersDiscovery
	registry string
	timeout  time.Duration

	lastUpdate time.Time
	serveName  string
}

//客户端每隔10秒从注册中心拉取服务列表
const defaultUpdateTimeout = time.Second * 10

func NewKRegistryDiscovery(registerAddr string, timeout time.Duration, serveName string) *KRegistryDiscovery {
	if timeout == 0 {

		timeout = defaultUpdateTimeout
	}
	d := &KRegistryDiscovery{
		MultiServersDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:              registerAddr,
		timeout:               timeout,
		serveName:             serveName,
	}
	return d
}

//更新服务列表

func (d *KRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastUpdate = time.Now()
	return nil
}

//客户端从注册中心接收服务列表

func (d *KRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		//未达到需要刷新服务列表的时间
		return nil
	}
	log.Println("rpc registry: refresh "+d.serveName+" servers from registry", d.registry)
	resp, err := http.Get(d.registry + "?Mini-Get-Serve-Name=" + d.serveName)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}

	//defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		// 如果不是200 OK，则根据需要处理
		fmt.Printf("unexpected status code: %d\n", resp.StatusCode)
	}

	servers := strings.Split(resp.Header.Get("Mini-Servers"), ",")
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

func (d *KRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServersDiscovery.Get(mode)
}

func (d *KRegistryDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServersDiscovery.GetAll()
}

type SelectMode int

const (
	RandomSelect     SelectMode = iota // 随机选择 select randomly
	RoundRobinSelect                   // 轮询法 select using Robbin algorithm
)

//
type Discovery interface {
	Refresh() error                      //   从注册中心更新服务列表
	Update(servers []string) error       //手动更新服务列表
	Get(mode SelectMode) (string, error) //根据负载均衡策略，选择一个服务实例
	GetAll() ([]string, error)           //返回所有的服务实例
}

type MultiServersDiscovery struct {
	r       *rand.Rand
	mu      sync.RWMutex
	servers []string
	index   int
}

func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())), //初始化时使用时间戳设定随机数种子，避免每次产生相同的随机数序列。
	}
	//index 记录 Round Robin 算法已经轮询到的位置，为了避免每次从 0 开始，初始化时随机设定一个值。
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

//实现Discovery接口

var _ Discovery = (*MultiServersDiscovery)(nil)

// Refresh doesn't make sense for MultiServersDiscovery, so ignore it
func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

func (d *MultiServersDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n] // servers could be updated, so mode n to ensure safety
		d.index = (d.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

func (d *MultiServersDiscovery) GetAll() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}

type XClient struct {
	d       Discovery
	mode    SelectMode
	opt     *rpcProject.Option
	mu      sync.Mutex                    // protect following
	clients map[string]*rpcProject.Client //连接池
}

var _ io.Closer = (*XClient)(nil)

func NewXClient(d Discovery, mode SelectMode, opt *rpcProject.Option) *XClient {
	return &XClient{d: d, mode: mode, opt: opt, clients: make(map[string]*rpcProject.Client)}
}

func (xc *XClient) Close() error {
	xc.mu.Lock()
	defer xc.mu.Unlock()
	for key, client := range xc.clients {

		_ = client.Close()
		delete(xc.clients, key)
	}
	return nil
}

func (xc *XClient) dial(rpcAddr string) (*rpcProject.Client, error) {
	xc.mu.Lock()
	defer xc.mu.Unlock()

	client, ok := xc.clients[rpcAddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(xc.clients, rpcAddr)
		client = nil
	}
	if client == nil {
		var err error

		client, err = rpcProject.Dial("tcp", rpcAddr, xc.opt)
		if err != nil {
			return nil, err
		}
		xc.clients[rpcAddr] = client
	}
	return client, nil
}

func (xc *XClient) call(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	client, err := xc.dial(rpcAddr)
	//遇到注册中心还有效但实际上已经失效的服务,返回下面这个err
	if err != nil {
		return err
	}
	return client.Call(ctx, serviceMethod, args, reply)
}

func (xc *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	//根据负载均衡方法从注册中心寻找一个服务地址
	rpcAddr, err := xc.d.Get(xc.mode)
	if err != nil {
		return err
	}
	//调用服务
	return xc.call(rpcAddr, ctx, serviceMethod, args, reply)
}

func (xc *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := xc.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex // protect e and replyDone
	var e error
	//若reply为nil，则所有服务都不会给reply赋值；若reply不为nil,则reply最终会被赋的值是最后一个调用完成的服务返回的值
	replyDone := reply == nil              // if reply is nil, don't need to set value
	ctx, cancel := context.WithCancel(ctx) //借助 context.WithCancel 确保有错误发生时，快速失败
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var clonedReply interface{}
			if reply != nil {

				//代码1和代码2的共同作用是不改变reply的指针，直接在reply指向的那块内存写入clonedReply的数据
				clonedReply = reflect.New(reflect.ValueOf(reply).Elem().Type()).Interface() //代码1
			}

			err := xc.call(rpcAddr, ctx, serviceMethod, args, clonedReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				//当调用cancel时，它会取消ctx 以及其所有衍生的子 Context
				//cancel最终导致Client.Call()里的ctx.Done()，移除Call
				cancel() // if any call failed, cancel unfinished calls
			}
			if err == nil && !replyDone {
				//将clonedReply的值拷贝回reply
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(clonedReply).Elem()) //代码2
				replyDone = true
			}
			mu.Unlock()

		}(rpcAddr)
	}
	wg.Wait()
	return e
}
