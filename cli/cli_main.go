package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"p2psystem/client"
	"p2psystem/server"
	"strings"
)

/**
Initialises the CLI for the p2p system
*/
func Init(){
	fmt.Print("CLI initialised\n");
	var stdinStr string;
	var err error;
	reader := bufio.NewReader(os.Stdin);
	brk := false;
	for (true){
		stdinStr, err = reader.ReadString('\n');
		if (err == io.EOF){break;}
		// Strip out the newline at the end
		if (stdinStr[len(stdinStr) - 1] == '\n'){
			stdinStr = strings.Replace(stdinStr, "\n", "", -1);
		}

		var parseResult CLIParse = ParseStr(stdinStr);
		session := client.GetSession();

		switch parseResult.CmdType{
		case MSG: {
			//delStr := strings.Repeat("\b", len(stdinStr) + 1);
			//fmt.Printf("%s",delStr);
			fmt.Print("\033[F");
			client.SendMessage(session.CurrentConnection, stdinStr);
		}
		case Quit: {
			fmt.Print("Quitting\n");
			brk = true;
		}
		case Connect: {			
			aliasedAddr, err := client.GetSavedRoom(session, parseResult.address);

			if (err != nil){
				fmt.Printf("cliMain: unable to parse /connect: %s", err);
				return;
			}

			if (aliasedAddr == ""){
				fmt.Printf("Connecting to address\n");
				client.Connect(parseResult.address);
			} else {
				fmt.Printf("Connecting to alias %s, addr: %s\n", parseResult.address, aliasedAddr);
				client.Connect(aliasedAddr)
			}
		}
		case ViewSaved:{
			client.DisplaySavedAliases(session);
		}

		case Nickname: {
			client.ChangeNickname(session.CurrentConnection, parseResult.info);
		}
		default:{
			fmt.Print("Invalid command\n");
		}
		}
		if (brk){
			break;
		}
	}
	// Shutdown the client by disconnecting from all servers
	client.DisconnectAll(client.GetSession());
	server.Shutdown(server.GetServerRoom());

	client.WriteConfig(client.GetSession(), "config");

}