package cache

import (
	"fmt"
	"log"
	"sync"
	"whiteCache/cache/singleflight"
)

// 负责与外部交互，控制缓存存储和获取的主流程

type Getter interface {
	Get(key string) ([]byte, error)
}

//设计了一个回调函数(callback)，
//在缓存不存在时，调用这个函数，得到源数据
type GetterFunc func(key string) ([]byte, error)

//Get:定义函数类型 GetterFunc，并实现 Getter 接口的 Get 方法。

/*
函数类型实现某一个接口，称之为接口型函数，使用者在调用时既能够传入函数作为参数，
也能够传入实现了该接口的结构体作为参数
*/
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//Group:是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name
type Group struct {
	name string
	//getter: 缓存未命中时获取源数据的回调
	getter Getter
	//mainCache: 支持并发控制的并发缓存
	mainCache cache
	//peers: HTTP客户端
	peers PeerPicker
	//loader: 用于防止缓存击穿
	loader *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

//NewGroup:实例化group，并且将 group 存储在全局变量 groups 中。
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

//GetGroup:根据命名空间name返回命名空间实例
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RLock()
	return g
}

//RegiserPeers:实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegiserPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("无法重复注册")
	}
	g.peers = peers
}

//Get(): 从 mainCache 中查找缓存，如果存在则返回缓存值
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("已命中缓存", key)
		log.Println("[whiteCache] hit")
		return v, nil
	}
	//else{
	//	g.getter.Get(key)
	//}
	return g.load(key)
}

//load():缓存不存在，则调用 load 方法，load 调用 getLocally
func (g *Group) load(key string) (value ByteView, err error) {
	//使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				//peer是根据传入的key值，基于哈希环计算出来的对应的真实服务器节点的客户端
				//该客户端是可以根据传入的key从相应的group中来返回缓存值
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("mycache无法从远程节点获取到值")
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	//根据group的name以及key进行查找
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

//getLocally():调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
