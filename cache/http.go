package cache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"whiteCache/cache/consistenhash"
)

const (
	defaultBasePath = "/_whitecache"
	defaultReplicas = 50
)

//HTTPPool:作为承载节点间 HTTP 通信的核心数据结构
//HTTPPool 既具备了提供 HTTP 服务的能力
//也具备了根据具体的 key，创建 HTTP 客户端从远程节点获取缓存值的能力
type HTTPPool struct {
	//self: 用来记录自己的地址，包括主机名/IP 和端口
	self string
	//basePath: 作为节点间通讯地址的前缀，默认是/_whitecache/
	// http://example.com/_whitecache/ 开头的请求，就用于节点间的访问
	basePath string

	mu sync.Mutex
	//peers: 哈希环
	peers *consistenhash.Map
	//httpGetters: 映射远程节点与对应的 httpGetter
	httpGetters map[string]*httpGetter
}

/***************服务端*******************/
//NewHTTPPool:实例化服务端 self为自己地址 basePath为通讯地址前缀
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[server %s]%s", p.self, fmt.Sprintf(format, v...))
}

//HTTP服务端
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("前缀错误" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	//约定访问路径格式为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath)+1:], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	//通过 groupname 得到 group 实例
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "不存在该命名空间:"+groupName, http.StatusNotFound)
		return
	}
	//再使用 group.Get(key) 获取缓存数据。
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

/***************客户端*******************/

//添加了传入的节点。
//并为每一个节点创建了一个 HTTP 客户端 httpGetter
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenhash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

//PickPeer: 包装了一致性哈希算法的 Get() 方法
//根据具体的 key，选择到真实的服务器节点，返回节点对应的 HTTP 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
