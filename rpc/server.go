package rpc

import (
	"net"
	"sync"

	"google.golang.org/grpc"
)

type RpcServer struct {
	listener   net.Listener
	grpcServer *grpc.Server
	address    string
	wg         sync.WaitGroup
}

func NewRpcServer(serverAddr string) (*RpcServer, error) {
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
