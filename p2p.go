package main

import (
	"os"
	"p2psystem/cli"
	"p2psystem/client"
	"p2psystem/server"
)

func main(){
	args := os.Args;
	nick := "Guest";
	if (len(args) > 1){
		nick = args[1];
	}

	client.Init(nick);
	server.Init("localhost", 9001);
	cli.Init();
}