package lru

import (
	"container/list"
)

type Cache struct {
	//maxBytes:允许使用的最大内存
	maxBytes int64
	//nbytes:当前已使用的内存
	nbytes int64
	ll     *list.List
	cache  map[string]*list.Element
	//OnEvicted:某条记录被移除时的回调函数，可以为 nil
	OnEvicted func(key string, value Value)
}
type entry struct {
	key   string
	value Value
}
type Value interface {
	Len() int
}

//New:实例化
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//Get:查找功能
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return
}

//RemoveOldest:删除最久未访问
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	c.ll.Remove(ele)
	kv := ele.Value.(*entry)
	delete(c.cache, kv.key)
	c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

//Add:新增
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{value: value, key: key})
		c.cache[key] = ele
		c.nbytes += int64(value.Len()) + int64(len(key))
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}
func (c *Cache) Len() int {
	return c.ll.Len()
}
