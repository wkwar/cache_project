// 对相同访问的一个合并请求,如果请求已经存在，其他的相同请求就会堵塞。
package singleflight

import (
	"fmt"
	"sync"
)

// 请求的封装
type call struct {
	wg  sync.WaitGroup //多线程管理,等待线程结束
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	//先判断是否已经有相同请求
	if c, ok := g.m[key]; ok {
		//该请求堵塞，等待前面相同请求结束即可
		g.mu.Unlock()
		c.wg.Wait()
		fmt.Println("后面请求等待")
		return c.val, c.err
	}
	//第一次请求
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c //映射
	fmt.Println("第一次请求")
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	//删除映射
	delete(g.m, key)
	g.mu.Unlock()
	c.wg.Wait()
	return c.val, c.err

}
