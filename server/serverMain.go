package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"p2psystem/common"
	"sync"
)

// The server struct keeps track of which clients are connected to it
type ServerRoom struct {
	socket net.Listener;
	instructions chan uint8;
	clients []*serverConnection;
	maxClients uint8;
	mainThread sync.WaitGroup;		// Tracks the goroutine running serverMain
	childThreads sync.WaitGroup;	// Tracks the goroutines running connectionMain
}

const (
	// InitialMaxClients is the initial max clients each ServerRoom is set to
	InitialMaxClients = 10;
	// ConnType is the protocol to use when creating new sockets
	ConnType = "tcp";
	// ServerStop indicates the server should stop listening for new connections and closes all existing ones
	ServerStop = 127; 
)

// For now we're just keeping it as one client has one server
var serv ServerRoom = ServerRoom{
	socket: nil,
	maxClients: InitialMaxClients,
	instructions: make(chan uint8, 1),
	clients: make([]*serverConnection, 0, InitialMaxClients),

	mainThread: sync.WaitGroup{},
	childThreads: sync.WaitGroup{},
};

// AnnounceMsg sends an ANC packet to all connected clients with the given message
func AnnounceMsg(server *ServerRoom, msg string) (error){
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
		if ((conn == nil) || conn.dead){
			continue;
		}

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

// GetServerRoom returns the given server room
func GetServerRoom()(*ServerRoom){
	return &serv;
}

// Shutdown will close the given serverRoom and disconnect the connected clients
func Shutdown(server *ServerRoom) (error){
	// Prepare a packet
	pkt := common.MsgPacket{
		PktType: common.PktKCK,
	}
	data := make([]byte, common.PktBufferSize);

	err := common.EncodeMessage(&pkt, "server shutdown");
	err = common.SerializePacket(&pkt, data);
	if (err != nil){
		fmt.Printf("serverMain.Shutdown: unable to send packet to clients: %s", err);
		return fmt.Errorf("serverMain.Shutdown: %s", err);
	}
	server.instructions <- uint8(ServerStop);

	server.mainThread.Wait();

	return nil;
}

func Init(IP string, port int16) (error){
	fmt.Print("serverMain: Initialised server component\n");

	connection, err := net.Listen(ConnType, ":9002"); // TODO: replace with value
	if (err != nil){
		return fmt.Errorf("serverMain: %s", err);
	}

	serv.socket = connection;

	serv.mainThread.Add(1);
	go serverMain(&serv);
	return nil;
	
}