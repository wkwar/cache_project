package services

import (
	"context"
	"time"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	cli *clientv3.Client
)

func Init(serverAddr string) error {
	cl, err := clientv3.New(clientv3.Config{
        Endpoints: []string{serverAddr},
        DialTimeout: 5 * time.Second,
    })
    if err != nil {
        fmt.Println("etcd client init falied")
		return err
    }
	cli = cl
	fmt.Println("init etcd sucess")
	return nil
}

// 注册服务到 etcd 中
func RegisterService(service string, IPAddr string) error {
    // 创建租约，设置过期时间
    leaseResp, err := cli.Grant(context.Background(), 5)
    if err != nil {
        return err
    }
	leaseId := leaseResp.ID
    // 注册服务节点,并且添加续约
    _, err = cli.Put(context.Background(), service+"/"+ IPAddr, IPAddr, clientv3.WithLease(leaseId))
    if err != nil {
        return err
    }

    // 开启定时刷新租约
    //keepAliveChan, err := cli.KeepAlive(context.Background(), leaseResp.ID)
	keepAliveChan, err := cli.KeepAlive(context.Background(), leaseResp.ID)
    if err != nil {
        return err
    }
	fmt.Printf("[%s] register [%s] success\n", IPAddr, service)
	
	
	//监听
	for {
		select {
		case <-cli.Ctx().Done():
			fmt.Println("service closed")
			return nil
		case _, ok := <-keepAliveChan:
			// 监听租约
			if !ok {
				fmt.Println("keep alive channel closed")
				//撤销（revoke）注册表（registry）中的服务。
				_, err := cli.Revoke(context.Background(), leaseId)
				return err
			}
			// fmt.Printf("Recv reply from service: %s/%s, ttl:%d \n", service, IPAddr, leaseResp.TTL)
		}
	} 
}

