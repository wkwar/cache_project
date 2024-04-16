package groupcache

import (
	"context"
	"fmt"
	"groupcache/consistenthash"
	"groupcache/registry"
	"log"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// 根据key 获取 远程节点
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool, isSelf bool)
}

// 客户端 发起 请求
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
	Delete(group string, key string) (bool, error)
}

// 每个网络节点都有一个缓存区 peer
type ClientPicker struct {
	self        string //自己的地址: 主机名/IP + 端口号
	serviceName string
	mu          sync.RWMutex
	consHash    *consistenthash.Map //节点管理 真实节点与虚拟节点的映射Map
	clients     map[string]*Client
}

func NewClientPicker(self string, opts ...PickerOptions) *ClientPicker {
	picker := ClientPicker{
		self:        self,
		serviceName: defaultServiceName,
		clients:     make(map[string]*Client),
		mu:          sync.RWMutex{},
		consHash:    consistenthash.New(),
	}

	//根据选择项，更改设置
	picker.mu.Lock()
	for _, opt := range opts {
		opt(&picker)
	}
	picker.mu.Unlock()

	//开启服务端，注册服务节点到哈希MAP
	picker.set(picker.self)
	//增量更新  --- 用于监控 增加服务节点 或者 删除服务节点(后续过程中)
	go func() {
		cli, err := clientv3.New(*registry.GlobalClientConfig)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer cli.Close()

		//监控服务节点列表
		watcher := clientv3.NewWatcher(cli)
		watchCh := watcher.Watch(context.Background(), picker.serviceName, clientv3.WithPrefix())
		for {
			a := <-watchCh
			go func() {
				picker.mu.Lock()
				defer picker.mu.Unlock()
				for _, x := range a.Events {
					key := string(x.Kv.Key)
					idx := strings.Index(key, picker.serviceName)
					addr := key[idx + len(picker.serviceName) + 1 : ]
					if addr == picker.self {
						continue
					}

					if x.IsCreate() {
						if _, ok := picker.clients[addr]; !ok {
							picker.set(addr)
						}
					} else if x.Type == clientv3.EventTypeDelete {
						if _, ok := picker.clients[addr]; !ok {
							picker.remove(addr)
						}
					}

				}
			}()
		}
	}()

	//全量更新  --- 查看全部ETCD所有服务列表，进行补充(刚开始添加阶段，后添加的服务节点无法通过监控得到服务节点)
	go func() {
		picker.mu.Lock()
		cli, err := clientv3.New(*registry.GlobalClientConfig)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer cli.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
		defer cancel()

		resp, err := cli.Get(ctx, picker.serviceName, clientv3.WithPrefix())
		if err != nil {
			log.Panic("[Event] full copy request failed")
		}
		kvs := resp.OpResponse().Get().Kvs
		defer picker.mu.Unlock()

		for _, kv := range kvs {
			key := string(kv.Key)
			idx := strings.Index(key, picker.serviceName)
			addr := key[idx + len(picker.serviceName) + 1 : ]
			if _, ok := picker.clients[addr]; !ok {
				picker.set(addr)
			}
		}

	}()
	return &picker
}


type PickerOptions func(*ClientPicker) 

func PickerServiceName(serviceName string) PickerOptions {
	return func(picker *ClientPicker) {
		picker.serviceName = serviceName
	}
}

func ConsOptions(opts ...consistenthash.ConsOptions) PickerOptions {
	return func(picker *ClientPicker) {
		picker.consHash = consistenthash.New(opts...)
	}
}


func (p *ClientPicker) set(addr string) {
	p.consHash.Add(addr)
	p.clients[addr] = NewClient(addr, p.serviceName)
}

func (p *ClientPicker) remove(addr string) {
	p.consHash.Remove(addr)
	delete(p.clients, addr)
}

func (s *ClientPicker) PickPeer(key string) (PeerGetter, bool, bool) {
	s.mu.Lock()
	defer s.mu.RUnlock()

	if peer := s.consHash.Get(key); peer != "" {
		s.Log("Pick peer %s", peer)
		return s.clients[peer], true, peer == s.self
	}
	return nil, false, false
}

func (s *ClientPicker) Log(format string, path ...interface{}) {
	log.Printf("[Server %s] %s", s.self, fmt.Sprintf(format, path...))
}