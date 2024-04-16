package groupcache

import (
	"context"
	"fmt"
	pb "groupcache/pb"
	"groupcache/registry"
	"log"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultServiceName = "group-cache"
	defaultAddr        = "127.0.0.1:8080"
)

type Server struct {
	pb.UnimplementedGroupcacheServer
	self       string //自己ip地址
	sname      string //服务名字
	status     bool   //服务器状态
	mu         sync.Mutex
	stopSignal chan error //停止信号
}

type ServerOptions func(*Server)

func NewServer(self string, opts ...ServerOptions) (*Server, error) {
	if self == "" {
		self = defaultAddr
	} 

	s := Server {
		self: self,
		sname: defaultServiceName,
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &s, nil
}

func ServiceName(sname  string) ServerOptions {
	return func(s *Server) {
		s.sname = sname
	}
}

func (s *Server) Log(format string, path ...interface{}) {
	log.Printf("[Server %s] %s", s.self, fmt.Sprintf(format, path...))
}

func (s *Server) Get(ctx context.Context, in *pb.Requst) (*pb.ResponseForGet, error) {
	group, key := in.GetGroup(), in.GetKey()
	out := &pb.ResponseForGet{}
	log.Printf("[Group-cache %s] Recv RPC Request for get- (%s)/(%s)", s.self, group, key)

	if key == "" {
		return out, fmt.Errorf("key required")
	}

	g := GetGroup(group)
	if g == nil {
		return out, fmt.Errorf("group not found")
	}
	view, err := g.Get(key)
	if err != nil {
		return out, err
	}

	out.Value = view.ByteSlice()
	return out, nil
}


func (s *Server) Delete(ctx context.Context, in *pb.Requst) (*pb.ResponseForDelete, error) {
	group, key := in.GetGroup(), in.GetKey()
	out := &pb.ResponseForDelete{}
	log.Printf("[Group-cache %s] Recv RPC Request for get- (%s)/(%s)", s.self, group, key)

	if key == "" {
		return out, fmt.Errorf("key required")
	}

	g := GetGroup(group)
	if g == nil {
		return out, fmt.Errorf("group not found")
	}
	isSuccess, err := g.Delete(key)
	if err != nil {
		return out, err
	}

	out.Value = isSuccess
	return out, nil
}


//开启rpc 服务节点
func (s *Server) Start() error {
	s.mu.Lock()
	if s.status {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	s.status = true
	s.stopSignal = make(chan error) 
	port := strings.Split(s.self, ":")[1]
	//监听 服务端 端口
	l, err := net.Listen("tcp", ":" + port)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	// 向服务端对象注册Server服务
	pb.RegisterGroupcacheServer(grpcServer, s)

	// 注册服务端反射服务
	reflection.Register(grpcServer)

	//开启一个协程，向ETCD注册一个 租约
	go func() {
		err := registry.Register(s.sname, s.self, s.stopSignal)
		if err != nil {
			log.Fatalf(err.Error())
		}
		close(s.stopSignal)
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.Printf("[%s] Revoke service and close tcp socket ok", s.self)
	} ()

	s.mu.Unlock()
	//开启 grpc服务
	if err := grpcServer.Serve(l); s.status && err != nil {
		return fmt.Errorf("failed to serve on %s: %v", port, err)
	}
	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	if !s.status {
		s.mu.Unlock()
		return 
	}

	s.stopSignal <- nil
	s.status = false
	s.mu.Unlock()
}