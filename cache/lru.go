package cache

import (
	"container/list"
	"sync"
	"time"
)



// 返回值类型size
type Value interface {
	Len() int
}

// 不是并发安全的；多用户同时操作时，需要加锁
type Cache interface {
	Get(key string) (Value, bool)
	Add(key string, value Value)
	AddWithExpiration(key string, value Value, expirationTime time.Time)
	Delete(key string) bool
}

type lruCache struct {
	lock     sync.Mutex
	maxBytes int64
	nbytes   int64
	//双向列表，存储数据的
	ll *list.List
	cache   map[string]*list.Element	//缓存存储
	expires map[string]time.Time 		//记录设置TTL的key
	//删除语句时，执行这个函数 -- 自己定义，log自知
	OnEvild func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

func NewLRUCache(maxBytes int64) *lruCache {
	answer :=  &lruCache{
		maxBytes: maxBytes,
		nbytes:   0,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
		expires:  make(map[string]time.Time),
	}

	//开启一个协程，周期性清除
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			//定时检测，清除过期的key
			answer.periodicMemoryClean()
		}
	} ()
	return answer
}

// 增加
func (c *lruCache) Add(key string, value Value) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	c.baseAdd(key, value)
	delete(c.expires, key)
	c.freeMemoryIfNeeded()
}

//设置了TTL的key
func (c *lruCache) AddWithExpiration(key string, value Value, expirationTime time.Time) {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	c.baseAdd(key, value)
	c.expires[key] = expirationTime
	c.freeMemoryIfNeeded()
	
	
}

//存储 value
func (c *lruCache) baseAdd(key string, value Value) {
	//修改
	if _, ok := c.cache[key]; ok {
		c.nbytes += int64(value.Len() - c.getValueSizeByKey(key))
		//update value
		c.cache[key].Value = &entry{key: key, value: value}
		c.ll.MoveToBack(c.cache[key])
		return
	} else {
		c.nbytes += int64(len(key) + value.Len())
		c.cache[key] = c.ll.PushBack(&entry{key: key, value: value})
	}
}

// 查看
func (c *lruCache) Get(key string) (value Value, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	expirationTime, ok := c.expires[key]
	//key 过期
	if ok && expirationTime.Before(time.Now())  {
		v := c.cache[key].Value.(*entry)
		c.nbytes -= int64(v.value.Len() + len(v.key))
		c.ll.Remove(c.cache[key])
		delete(c.cache, key)
		delete(c.expires, key)

		if c.OnEvild != nil {
			c.OnEvild(key, v.value)
		}
	}

	//没有过期
	if v, ok := c.cache[key]; ok {
		c.ll.MoveToBack(v)
		return v.Value.(*entry).value, true
	}

	return nil, false
}

// 移除
func (c *lruCache) Delete(key string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.nbytes -= int64(len(key) + c.getValueSizeByKey(key))
	delete(c.cache, key)
	delete(c.expires, key)
	return true
}

//删除元素
func (c *lruCache) freeMemoryIfNeeded() {
	for c.nbytes > c.maxBytes {
		v := c.ll.Front()
		if v != nil {
			c.ll.Remove(v)
			kv := v.Value.(*entry)
			delete(c.cache, kv.key)
			delete(c.expires, kv.key)
			c.nbytes -= int64(len(kv.key) + kv.value.Len())
			if c.OnEvild != nil {
				c.OnEvild(kv.key, kv.value)
			}
		}
	}
}

func (c *lruCache) periodicMemoryClean() {
	c.lock.Lock()
	defer c.lock.Unlock()
	
	n := len(c.expires) / 10
	//遍历设置TTL的key，随机删除 1/10 过期key
	for key := range c.expires {
		if c.expires[key].Before(time.Now()) {
			c.nbytes -= int64(len(key) + c.getValueSizeByKey(key))
			delete(c.expires, key)
			delete(c.cache, key)
		}
		n--
		if n == 0 {
			break
		}
	}
}

//获取key对应 value size
func (c *lruCache) getValueSizeByKey(key string) int {
	return c.cache[key].Value.(*entry).value.Len()
}