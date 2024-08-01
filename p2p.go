package main

import (
	"p2psystem/cli"
	"p2psystem/client"
	"p2psystem/server"
)

func main(){
	client.Init();
	server.Init("localhost", 9001);
	cli.Init();
}