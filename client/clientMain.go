package client

import (
	"fmt"
	"net"
	"p2psystem/common"
	"time"
)

const (
	// ClientNetworkType is a config variable that's fed into Dial as the Network argument
	ClientNetworkType = "tcp";
	ClientDisconnect = 0;
	
)

type clientConnection struct{
	instructions chan int;
	server net.Conn;
}

var currentConnection *clientConnection;
var connectedServers []*clientConnection;
var nickname string;

// SendMessage will send the given string to the connection
func SendMessage(connection *clientConnection, msg string) (error){
	// prepare a packet
	var pkt common.MsgPacket = common.MsgPacket{
		PktType: common.PktMSG,
	}
	err := common.EncodeMessage(&pkt, msg);
	if (err != nil){
		fmt.Printf("clientMain: Unable to encode message: %s\n", err);
		return fmt.Errorf("SendMessage: %s", err);
	}

	var data []byte = make([]byte, common.PktBufferSize);
	err = common.SerializePacket(&pkt, data);

	_, err = connection.server.Write(data);
	if (err != nil){
		fmt.Printf("clientMain: Unable to send message: %s\n", err);
		return fmt.Errorf("SendMessage: %s", err);
	}

	return nil;
}

// GetCurrentConnection returns the connection the client is currently interfacing with
func GetCurrentConnection() (*clientConnection){
	return currentConnection;
}

// SetNickname modifies the nickname the client is currently using and broadcasts
// the change to the server
func SetNickname(newNick string){

}

// Connect will establish a connection to the given address
func Connect(addr string) (error){
	//fmt.Printf("clientMain: Connecting to %s\n", addr);
	conn, err := net.DialTimeout(ClientNetworkType, addr, (4 * time.Second));
	if err != nil {
		fmt.Printf("Unable to connect to %s: %s\n", addr, err);
		return fmt.Errorf("clientMain: %s", err);
	}
	//fmt.Printf("clientMain: dialled %s, from %s\n", conn.RemoteAddr().String(), conn.LocalAddr().String());
	// Create the client
	var client clientConnection = clientConnection{
		server: conn,
		instructions: make(chan int),
	}

	var status bool;
	//fmt.Printf("clientMain: beginning handshake\n");
	status, err = handleHandshake(&client);
	if (err != nil){
		client.server.Close();
		return fmt.Errorf("clientMain: %s", err);
	}
	if (!status){
		fmt.Printf("clientMain: connection refused from %s\n", client.server.LocalAddr());
		client.server.Close();
		return nil;
	}
	fmt.Printf("clientMain: handshake completed\n");

	currentConnection = &client;
	go connMain(&client);
	return nil;
}

func Init(startingNickname string) {
	fmt.Print("Client component initialised\n");
	nickname = startingNickname;
}