package services

import (
	"sync"
	"log"
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
)


type msg struct {
	EventType string
	EventUrl string
}

var (
	serverList = make(map[string]string, 0)	//服务列表
	lock       sync.Mutex
	change = make(chan msg, 0)
)


//客户端开始 先获取etcd的服务列表
func DiscoverServices(services string) error {
	//根据前缀获取现有的key
	resp, err := cli.Get(context.Background(), services, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	//获取服务列表  使用一个map存储 可以使用一个key 存储一个列表
	for _, ev := range resp.Kvs {
		SetServiceList(string(ev.Key), string(ev.Value))
	}

	//监视前缀，修改变更的server
	go watcher(services)
	return nil
}

//watcher 监听前缀 哈希环 以及  map映射需要删除
func watcher(services string) {
	rch := cli.Watch(context.Background(), services, clientv3.WithPrefix())
	log.Printf("watching prefix:%s now...", services)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut: //修改或者新增
				SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
				change<-msg {
					EventType: "put",
					EventUrl: string(ev.Kv.Value),
				}
			case clientv3.EventTypeDelete: //删除
				DelServiceList(string(ev.Kv.Key))
				change<-msg {
					EventType: "del",
					EventUrl: string(ev.Kv.Value),
				}
			}
		}
	}
}

//SetServiceList 新增服务地址
func SetServiceList(key, val string) {
	lock.Lock()
	defer lock.Unlock()
	serverList[key] = string(val)
	log.Println("put key :", key, "val:", val)
}

//DelServiceList 删除服务地址
func DelServiceList(key string) {
	lock.Lock()
	defer lock.Unlock()
	delete(serverList, key)
	log.Println("del key:", key)
}

//GetServices 获取服务地址
func GetServices() []string {
	lock.Lock()
	defer lock.Unlock()
	addrs := make([]string, 0)

	for _, v := range serverList {
		addrs = append(addrs, v)
	}
	return addrs
}

//Close 关闭服务
func  Close() error {
	return cli.Close()
}

func RetChan() <-chan msg {
	return change
}

