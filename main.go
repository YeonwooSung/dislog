package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dislog/rpc"

	"google.golang.org/grpc"
)

const (
	defaultListenAddr = "0.0.0.0:9997"
)

var (
	listenAddr string
)

func init() {
	flag.StringVar(&listenAddr, "l", defaultListenAddr, "listen address")
	flag.Parse()
}

func main() {
	server, err := initServer()
	if err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)

	go func() {
		if <-sigC == os.Interrupt {
			log.Println("interrupted")
			server.Close()
		}
	}()

	// start running RPC server
	log.Println("running rpc_server")
	server.Run()
}

func initServer() (*rpc.RpcServer, error) {
	bufferSize := 128 * 1024
	rs, err := rpc.NewRpcServer(listenAddr, bufferSize, grpc.ReadBufferSize(bufferSize))
	if err != nil {
		return nil, err
	}
	return rs, nil
}
