package rpc

import (
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type RpcServer struct {
	listener   net.Listener
	grpcServer *grpc.Server
	address    string
	wg         sync.WaitGroup
}

func NewRpcServer(serverAddr string, bufsize int, gRpcServerOptions grpc.ServerOption) (*RpcServer, error) {
	sz := 1 << 10
	if bufsize > 0 {
		sz = bufsize
	}
	lis := bufconn.Listen(sz)
	rpcServer := &RpcServer{
		listener:   lis,
		grpcServer: grpc.NewServer(gRpcServerOptions),
		address:    lis.Addr().String(),
	}
	return rpcServer, nil
}

func NewRpcTestServer(serverAddr string) (*RpcServer, error) {
	// lis, err := net.Listen("tcp", "127.0.0.1:0")
	lis, err := net.Listen("tcp", serverAddr)
	// check if error occured
	if err != nil {
		return nil, err
	}
	addr := lis.Addr().String()

	rpcServer := &RpcServer{
		listener:   lis,
		grpcServer: grpc.NewServer(),
		address:    addr,
	}
	return rpcServer, nil
}

func (rs *RpcServer) Run() {
	rs.wg.Add(1)
	go func() {
		defer rs.wg.Done()
		_ = rs.grpcServer.Serve(rs.listener)
	}()
}

func (rs *RpcServer) Close() {
	rs.grpcServer.Stop()
	_ = rs.listener.Close()
	rs.wg.Wait()
}

func (rs *RpcServer) Address() string {
	return rs.address
}
