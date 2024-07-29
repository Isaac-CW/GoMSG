package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"p2psystem/common"
)

// The server struct keeps track of which clients are connected to it
type serverRoom struct {
	socket net.Listener;
	instructions chan uint8;
	clients []*serverConnection;
	maxClients uint8;
}

const (
	// InitialMaxClients is the initial max clients each serverRoom is set to
	InitialMaxClients = 10;
	// ConnType is the protocol to use when creating new sockets
	ConnType = "tcp";
	// ServerStop indicates the server should stop listening for new connections and closes all existing ones
	ServerStop = 1; 
)

// For now we're just keeping it as one client has one server
var serv serverRoom = serverRoom{
	socket: nil,
	maxClients: InitialMaxClients,
	clients: make([]*serverConnection, 0, InitialMaxClients),
};

// AnnounceMSG sends an ANC packet to all connected clients with the given message
func AnnounceMsg(server *serverRoom, msg string) (error){
	pkt := common.MsgPacket{
		PktType: common.PktANC,
	}
	err := common.EncodeMessage(&pkt, msg);
	if (err != nil){
		fmt.Printf("AnnounceMsg: Unable to encode message: %s\n", err);
		return fmt.Errorf("AnnounceMsg: %s", err);
	}

	var dataBuffer []byte = make([]byte, common.PktBufferSize);
	err = common.SerializePacket(&pkt, dataBuffer);
	if (err != nil){
		fmt.Printf("AnnounceMsg: Unable to serialize message: %s\n", err);
		return fmt.Errorf("AnnounceMsg: %s", err);
	}

	for ind, conn := range server.clients{
		_, err = conn.client.Write(dataBuffer);
		if (err != nil){
			if (errors.Is(err, io.EOF)){
				server.clients[ind].dead = true;
				server.clients[ind].client.Close();
				continue;
			}
			fmt.Printf("AnnounceMsg: Unable to send announcment to client at index %d: %s\n", ind, err);
			return fmt.Errorf("AnnounceMsg: %s", err);
		}
	}
	return nil;
}

// Continuously accepts connections and runs a new goroutine running 
// connectionMain to serve it
func serverMain(server *serverRoom){
	inbound := make(chan net.Conn);

	go func(){
		for {
			var conn net.Conn;
			conn, err := server.socket.Accept();
			if (err != nil){
				if ((err == io.EOF) || (err == io.ErrUnexpectedEOF)){
					print("EOF");
					break;
				}
				fmt.Printf("serverMain: unable to accept connection: %s\n",err);
				break;
			}

			inbound <- conn;
		}
	}()

	for {
		select {
		case newConnection := <- inbound:{
			createConnection(&serv, newConnection);
		}
		case currentInstruction := <- server.instructions:{
			if (currentInstruction == ServerStop){
				serv.socket.Close();
			}
		}
		}
	}
}

func Init(IP string, port int16) (error){
	fmt.Print("serverMain: Initialised server component\n");

	connection, err := net.Listen(ConnType, ":9002"); // TODO: replace with value
	if (err != nil){
		return fmt.Errorf("serverMain: %s");
	}

	serv.socket = connection;

	go serverMain(&serv);
	return nil;
	
}