package cache

// 并发控制
import (
	"sync"
	"whiteCache/cache/lru"
)

//实例化 lru，封装 get 和 add 方法，并添加互斥锁 mu
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

//add: 判断了 c.lru 是否为 nil，如果等于 nil 再创建实例。这种方法称之为延迟初始化
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), true
	}
	return
}
