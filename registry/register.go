package registry

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
)

var (
	GlobalClientConfig *clientv3.Config = defaultEtcdConfig
	defaultEtcdConfig = &clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
)

func etcdAdd(c *clientv3.Client, lid clientv3.LeaseID, service, addr string) error {
	em, err := endpoints.NewManager(c, service)
	if err != nil {
		return err
	}

	return em.AddEndpoint(c.Ctx(), service + "/" + addr, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(lid))

}

//注册服务节点 到 etcd中
func Register(service, addr string, stop chan error) error {
	cli, err := clientv3.New(*GlobalClientConfig)
	if err != nil {
		return fmt.Errorf("create etcd client failed: %v", err)
	}
	defer cli.Close()

	//创建一个续约
	resp, err := cli.Grant(context.Background(), 2)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}

	leaseID := resp.ID
	//注册服务
	err = etcdAdd(cli, leaseID, service, addr)
	if err != nil {
		return fmt.Errorf("add etcd record failed: %v", err)
	}
	//设置 心跳检测

	ch, err := cli.KeepAlive(context.Background(), leaseID)
	if err != nil {
		return fmt.Errorf("set keepalive failed: %v", err)
	}

	log.Printf("[%s] register services success", addr)

	for {
		select {
			case err := <-stop:
				if err != nil {
					log.Println(err)
				}
			case <-cli.Ctx().Done():
				log.Println("service closed")
				return nil
			case _, ok := <-ch:
				//监听租约
				if !ok {
					log.Println("keep channel closed")
					_, err := cli.Revoke(context.Background(), leaseID)
					return err
				}
		}
	}
}