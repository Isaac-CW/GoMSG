package cli

import (
	"fmt"
	"strings"
)

/**
Each enum represents the parsed action to be taken from the given command.
All commands must have is 0th character be '/'
	- MSG			: Any inputs that isn't prepended with a /
	- Unknown 		: The input is an unrecognized command
	- Quit 			: The CLI input should stop and the application should shutdown
	- Connect 		: Connect to a given address and port in the format address:port
	- Disconnect 	:
	- Nickname		: Signals to the connected server to change the nickname of the client
*/
const (
	MSG int = -2
	Unknown int = -1
	Quit int = 0
	Connect int = 1
	Disconnect int  = 2
	Nickname int = 3

)

// A struct containing the information associated with the string that we
// interpreted as a command
type CLIParse struct {
	CmdType int;
	address string;
	info string;
}

// Given the input string, return a constant representing the command type
func ParseStr(cmdStr string) CLIParse{
	var retVal CLIParse;
	retVal.CmdType = Unknown;

	if (cmdStr == ""){
		return retVal;
	}

	if (cmdStr[0] != '/'){
		retVal.CmdType = MSG;
		return retVal;
	}

	var cmdChunks []string = strings.Split(cmdStr, " ");
	// Check the 0th element
	cmd := strings.Trim(cmdChunks[0], "\n");
	//fmt.Printf("%s|\n", cmd);
	switch cmd{
	case "/quit":
		fallthrough;
	case "/QUIT":{
		retVal.CmdType = Quit;
	}
	case "/connect":
		fallthrough;
	case "/CONNECT":{
		if (len(cmdChunks) == 1){
			fmt.Print("Missing IP and Port in /connect\n");
			return retVal;
		}
		retVal.address = cmdChunks[1]

		// Finalize by setting the CmdType to Connect
		retVal.CmdType = Connect;
	}

	case "/nickname": fallthrough;
	case "/NICKNAME": fallthrough;
	case "/NICK": fallthrough;
	case "/nick":{
		if (len(cmdChunks) == 1){
			fmt.Print("Missing nickname to change\n");
			return retVal;
		}
		retVal.info = cmdChunks[1];

		// Finalize by setting the CmdType to Nickname
		retVal.CmdType = Nickname;
	}

	default:{
		retVal.CmdType = Unknown;
	}
	}
	return retVal;
}