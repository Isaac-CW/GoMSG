package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"p2psystem/client"
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
		switch parseResult.CmdType{
		case MSG: {
			delStr := strings.Repeat("\b", len(stdinStr) + 1);
			fmt.Printf("%s",delStr);
			client.SendMessage(client.GetCurrentConnection(), stdinStr);
		}
		case Quit: {
			fmt.Print("Quitting\n");
			brk = true;
		}
		case Connect: {			
			client.Connect(parseResult.address);
		}
		default:{
			fmt.Print("Invalid command\n");
		}
		}
		if (brk){
			break;
		}
	}

}