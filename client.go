package groupcache

import (
	"context"
	"fmt"
	"groupcache/registry"
	"log"
	"time"

	pb "groupcache/pb"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Client struct {
	addr        string
	serviceName string
}

func NewClient(addr, serviceName string) *Client {
	return &Client{
		addr:        addr,
		serviceName: serviceName,
	}
}

func (c *Client) Get(group, key string) ([]byte, error) {
	cli, err := clientv3.New(*registry.GlobalClientConfig)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer cli.Close()

	conn, err := registry.EtcdDial(cli, c.serviceName, c.addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	
	
	grpcClient := pb.NewGroupcacheClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	resp, err := grpcClient.Get(ctx, &pb.Requst{
		Group: group,
		Key: key,
	})

	if err != nil {
		return nil, fmt.Errorf("could not get %s-%s from peer %s", group, key, c.addr)

	}
	return resp.GetValue(), nil
}


func (c *Client) Delete(group string, key string) (bool, error) {
	cli, err := clientv3.New(*registry.GlobalClientConfig)
	if err != nil {
		log.Fatal(err)
		return false, err
	}
	defer cli.Close()

	conn, err := registry.EtcdDial(cli, c.serviceName, c.addr)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	

	grpcClient := pb.NewGroupcacheClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	resp, err := grpcClient.Delete(ctx, &pb.Requst{
		Group: group,
		Key: key,
	})

	if err != nil {
		return false, fmt.Errorf("could not get %s-%s from peer %s", group, key, c.addr)

	}
	return resp.GetValue(), nil
}
