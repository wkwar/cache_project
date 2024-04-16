package groupcache

import (
	"fmt"
	"groupcache/singleflight"
	"log"
	"sync"
	"time"
)

var (
	//注意，map要make初始化，不然赋值的时候报错
	groups = make(map[string]*Group)
	mu     sync.RWMutex
)

// 分组，存储数据 -- 组名 + 缓存区(mainCache, hotCache)
type Group struct {
	name      string	
	getter    Getter	//缓存未命中时的callback
	peers     PeerPicker
	mainCache cache
	hotCache  cache
	loader *singleflight.Group
}

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called multiple times")
	}
	g.peers = peers
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	//func(*sync.Once) Do()  只有第一次调用才会执行
	
	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
		loader: &singleflight.Group{},
		
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	if group, ok := groups[name]; ok {
		return group
	}
	return nil
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	return g.load(key)
}


func (g *Group) load(key string) (value ByteView, err error) {
	view, err := g.loader.Do(key, func() (interface{}, error) {
		//先查看是否存在远程节点
		if g.peers != nil {
			if peer, ok, itself := g.peers.PickPeer(key); ok {
				if itself {
					if v, ok := g.mainCache.get(key); ok {
						log.Println("[group-cache] hit")
						return v, nil
					}
				} else {
					if value, err := g.getFromPeer(peer, key); err == nil {
						return value, nil
					} else {
						log.Println("[group-cache] failed to get from peer", peer)
					}
				}
			}
		}
		//远程节点不存在该key，从源头处（数据库中）获取
		return g.getLocally(key)
	})
	if err == nil {
		return view.(ByteView), err
	}
	return
}


func (g *Group) Delete(key string) (bool, error) {
	if key == "" {
		return true, fmt.Errorf("key is required")
	}

	if g.peers == nil {
		return g.mainCache.delete(key), nil
	}

	peer, ok, isSelf := g.peers.PickPeer(key)
	if !ok {
		return false, nil
	}

	if isSelf {
		return g.mainCache.delete(key), nil
	} else {
		success, err := g.deleteFromPeer(peer, key)
		return success, err
	}
}


// 从远程节点处获取缓存数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}

	return ByteView{
		b: cloneBytes(bytes),
	}, nil
}


func (g *Group) deleteFromPeer(peer PeerGetter, key string) (bool, error) {
	success, err := peer.Delete(g.name, key)
	if err != nil {
		return false, err
	}
	return success, nil
}

// 从数据源头获取缓存数据
func (g *Group) getLocally(key string) (ByteView, error) {
	//try again
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Group-cache] hit")
		return v, nil
	}
	bytes, f, expirationTime := g.getter.Get(key)
	if !f {
		return ByteView{}, fmt.Errorf("data not found")
	}
	 
	bw := ByteView{b: cloneBytes(bytes)}
	if !expirationTime.IsZero() {
		g.mainCache.addWithExpiration(key, bw, expirationTime)
	} else {
		g.mainCache.add(key, bw)
	}
	return bw, nil
}


// 回调函数 -- 缓存不存在时，从源头处调用数据
// 用户自己定义
type Getter interface {
	Get(key string) ([]byte, bool, time.Time)
}

type GetterFunc func(key string) ([]byte, bool, time.Time)

func (f GetterFunc) Get(key string) ([]byte, bool, time.Time) {
	return f(key)
}

//删除group
func DestoryGroup(name string) {
	g := GetGroup(name)
	if g == nil {
		delete(groups, name)
		log.Printf("Destory cache [%s]", name)
	}
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
