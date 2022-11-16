package main

import (
	"fmt"
	"os"
	"tcp_test/client"
	"tcp_test/master"
)

func main() {
	if len(os.Args) == 1 {
		Tips()
		return
	}

	if os.Args[1] == "server" {
		master.Server = master.MasterInit()
		master.Server.Start()
		defer func() {}()
	} else if os.Args[1] == "client" {
		client.Clients = client.Initialize()
		client.Clients.Start()
	}
}

func Tips() {
	fmt.Println("tcp_test server [ip:port]")
	fmt.Println("tcp_test client [ip:port]")
	return
}
