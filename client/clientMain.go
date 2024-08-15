package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"p2psystem/common"
	"time"
)

const (
	// ClientNetworkType is a config variable that's fed into Dial as the Network argument
	ClientNetworkType = "tcp";
	// ClientDisconnect stops all handling functions
	ClientDisconnect = 0;
	
)

// ClientSession is a struct that represents the current running instance
// of the client and stores all the connections
type ClientSession struct {
	connectedServers []*ClientConnection;
	CurrentConnection *ClientConnection;

	Config *Config;
}

// ClientConnection represents a connection to a server
type ClientConnection struct{
	instructions chan uint8;
	dead bool;
	server net.Conn;
}

var client ClientSession = ClientSession{
	connectedServers: make([]*ClientConnection, 10),
	CurrentConnection:  nil,

	Config: nil,
}

// GetSession is the accessor for the client's session
func GetSession()(*ClientSession){
	return &client;
}

// ChangeNickname signals to the server to internally change the nickname of this
// slient. If the connection is closed then this silently doesn't raise any errors
func ChangeNickname(conn *ClientConnection, newNickname string) (error){
	if ((conn == nil) || conn.dead){
		fmt.Printf("Cannot change nickname: Server connection is closed\n");
		return nil;
	}

	if (len(newNickname) > common.NicknameMaxSize){
		fmt.Printf("Cannot change nickname: name is too long\n");
		return nil;
	}

	// Encode the pkt
	toChange := common.ClientModifcation{
		NewName: newNickname,
	}

	jsonBytes, err := json.Marshal(toChange);

	if (err != nil){
		fmt.Printf("clientMain.ChangeNickname: Unable to encode %s to MDF packet: %s\n", newNickname, err);
		return fmt.Errorf("clientMain.ChangeNickname: %s", err);
	}

	// Prepare a packet
	pkt := common.MsgPacket{
		PktType: common.PktMDF,
	}

	err = common.EncodeMessage(&pkt, string(jsonBytes));
	if (err != nil){
		fmt.Printf("clientMain.ChangeNickname: Unable to encode %s to MDF packet: %s\n", jsonBytes, err);
		return fmt.Errorf("clientMain.ChangeNickname: %s", err);
	}

	var data []byte = make([]byte, common.PktBufferSize);
	err = common.SerializePacket(&pkt, data);
	if (err != nil){
		fmt.Printf("clientMain.ChangeNickname: Unable to serialize MDF packet: %s\n", err);
		return fmt.Errorf("clientMain.ChangeNickname: %s", err);
	}

	// Send it over to the server
	_, err = conn.server.Write(data);
	if (err != nil){
		fmt.Printf("clientMain.ChangeNickname: Unable to send MDF packet: %s\n", err);
		return fmt.Errorf("clientMain.ChangeNickname: %s", err);
	}

	return nil;
}

// DisconnectAll will close every active connection in the given client session
func DisconnectAll(session *ClientSession) (error){
	// Prepare a DCN packet
	// This is done separately from Disconnect so that one packet can be used
	// for all connections rather than constantly making one for each connection
	pkt := common.MsgPacket{
		PktType: common.PktDCN,
	}
	data := make([]byte, common.PktBufferSize);

	err := common.SerializePacket(&pkt, data);
	if (err != nil){
		fmt.Printf("clientMain.DisconnectAll: unable to serialize packet: %s", err);
		return fmt.Errorf("clientMain.DisconnectAll: %s", err);
	}

	for _, v := range session.connectedServers{
		if ((v == nil) || v.dead){
			continue;
		}
		_, err = v.server.Write(data);
		if (err != nil){
			if (!errors.Is(err, io.EOF)){
				fmt.Printf("clientMain.DisconnectAll: Unable to send DCN packet: %s", err);
				return fmt.Errorf("clientMain.Disconnect: %s", err);
			}
		}
		v.instructions <- ClientDisconnect;
		v.dead = true;
		v.server.Close();
	}

	return nil;
}

// Disconnect closes the given ClientConnection
func Disconnect(connection *ClientConnection){
	connection.server.Close();
	connection.dead = true;
}

// SendMessage will send the given string to the connection
func SendMessage(connection *ClientConnection, msg string) (error){
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

// GetCurrentConnection returns the connection the given client session is currently interfacing with
func GetCurrentConnection(client *ClientSession) (*ClientConnection){
	return client.CurrentConnection;
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
	err = createConnection(&client, conn);

	if (err != nil){
		fmt.Printf("clientMain: Unable to create connection: %s", err);
		return fmt.Errorf("clientMain: %s", err);
	}

	return nil;
}

func Init() {
	err := ReadConfig(&client, "config/clientConfig.cfg");
	if (err != nil){
		fmt.Printf("Unable to parse client config: %s", err);
	}

	fmt.Print("Client component initialised\n");
}