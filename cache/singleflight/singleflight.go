package singleflight

import "sync"

//call:代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

//Group:管理不同 key 的请求(call)
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	if c, ok := g.m[key]; ok {
		c.wg.Wait()         //请求正在进行 则等待
		return c.val, c.err //上一个请求结束 返回结果
	}
	c := new(call)
	c.wg.Add(1)  //发起请求前需要加锁
	g.m[key] = c //添加到g.m

	c.val, c.err = fn() //请求结束调用fn
	c.wg.Done()         //请求结束解锁

	delete(g.m, key) //更新g.m
	return c.val, c.err
}
