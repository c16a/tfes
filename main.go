package main

import (
	"github.com/tfes-dev/tfes/pkg/net"
)

func main() {
	tcpPool := net.NewTcpHandlerPool()
	tcpPool.Start()
}
