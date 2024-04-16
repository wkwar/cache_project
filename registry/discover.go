package registry

import (
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"

	"google.golang.org/grpc"
)

//使用grpc请求服务，建立连接
func EtcdDial(c *clientv3.Client, service, target string) (*grpc.ClientConn, error) {
	// 创建 etcd 实现的 grpc 服务注册发现模块 resolver
	etcdResover, err := resolver.NewBuilder(c)
	if err != nil {
		return nil, err
	}
	
	// 创建 grpc 连接代理
	return grpc.Dial(
		"etcd:///" + service + "/"+target,
		grpc.WithResolvers(etcdResover),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
}