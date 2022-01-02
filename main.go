package main

import (
	"encoding/json"
	"flag"
	"github.com/tfes-dev/tfes/pkg/net"
	"github.com/tfes-dev/tfes/pkg/schemas"
	"io/ioutil"
)

func main() {

	configFile := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	filesBytes, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}

	var config schemas.Config
	err = json.Unmarshal(filesBytes, &config)
	if err != nil {
		panic(err)
	}

	peerServer := net.NewPeerListener(&config)
	go peerServer.Start()

	tcpPool := net.NewTcpHandlerPool(&config, peerServer)
	err = tcpPool.Start()
	if err != nil {
		panic(err)
	}
}
