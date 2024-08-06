package server

// Contains all the private methods used to manage connections

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"p2psystem/common"
	"strings"
	"sync"
	"time"
)

type serverConnection struct{
	client net.Conn;
	nickname string;
	dead bool;	// true if the socket is closed
	instructions chan int8;
}

// Forcibly closes the client and issues a KCK packet to the client
func kickClient(conn *serverConnection, reason string) (error){



	return nil;
}

// changes the given client's nickname to the given new name and announces the
// change to all clients
func changeNickname(server *ServerRoom, conn *serverConnection, newName string) (error){
	if ((conn == nil) || conn.dead){
		return fmt.Errorf("Client is already closed");
	}

	if (conn.nickname == newName){
		return nil
	}
	oldNick := conn.nickname;
	conn.nickname = newName;

	AnnounceMsg(server, fmt.Sprintf("%s has changed their name to %s", oldNick, conn.nickname));

	return nil;
}

func handleHandshake(conn *serverConnection, allow bool) (bool, error){
	var data []byte = make([]byte, common.PktBufferSize);

	var pkt common.MsgPacket;
	if (allow){
		pkt = common.MsgPacket{
			PktType: common.PktACP,
		}
	} else {
		pkt = common.MsgPacket{
			PktType: common.PktREF,
		}
	}
	err := common.SerializePacket(&pkt, data);
	if (err != nil){
		return false, fmt.Errorf("serverHandler.handleHandshake: %s", err);
	}

	_, err = conn.client.Write(data);
	if (err != nil){
		if (errors.Is(err, io.EOF)){
			fmt.Printf("serverHandshake: server closed socket\n");
			return false, nil;
		}
		return false, fmt.Errorf("serverHandshake: %s", err);
	}

	// Then read the ACK packet
	conn.client.SetReadDeadline(time.Now().Add(4 * time.Second));
	_, err = conn.client.Read(data);
	if (err != nil){
		if (errors.Is(err, io.EOF)){
			fmt.Printf("serverHandshake: server closed socket\n");
			return false, nil;
		}
		return false, fmt.Errorf("serverHandshake: %s", err);
	}

	pkt = common.DeserializePacket(data);
	if (pkt.PktType != common.PktACK){
		fmt.Printf("serverHandshake: unrecognised packet type\n");
		return false, nil;
	}
	// Decode the packet's payload too
	jsonRaw, err := common.DecodeMessage(&pkt);
	jsonRaw = strings.Trim(jsonRaw, "\x00");
	var clientMod common.ClientModifcation;

	err = json.Unmarshal([]byte(jsonRaw), &clientMod);
	if (err != nil){
		fmt.Printf("serverHandshake: unable to unpack ACK packet: %s", err);
		return false, fmt.Errorf("serverHandshake: %s", err);
	}
	// TODO: Check if the server has already labelled this client

	conn.nickname = clientMod.NewName;

	//fmt.Printf("serverHandshake: accepted ACK packet\n");

	return true, nil;
}

func connectionMain(connection *serverConnection, server *ServerRoom) (error){
	inbound := make(chan bool);
	var data [common.PktBufferSize]byte;

	connection.client.SetReadDeadline(time.Time{});

	go func(){
		for {
			_, err := connection.client.Read(data[:]);
			//fmt.Printf("connectionMain: Read data from %s\n", connection.client.RemoteAddr());
			if (err != nil){
				if (!errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed)){
					fmt.Printf("serverMain: Unable to read from client: %s\n", err);
				}
				inbound <- false;
				return;
			}

			inbound <- true;
		}
	}()

	brk := false;
	var err error = nil;
	
	for {
		if (brk){break;}
		select {
		case active := <- inbound:{
			if (!active){
				brk = true;
				continue;
			}

			//fmt.Printf("serverMain: received packet\n");
			var readPKT common.MsgPacket = common.DeserializePacket(data[:]);
			readPKT.SendNickname = connection.nickname;
			
			switch readPKT.PktType{
			case common.PktMSG:{
				err := common.SerializePacket(&readPKT, data[:]);
				if (err != nil){
					fmt.Printf("serverMain: unable to seralize MSG pkt: %s\n", err);
					err = fmt.Errorf("serverMain: %s", err);
					brk = true;
					continue;
				}
				// Sling it to every other client
				for _, conn := range server.clients{
					if ((conn == nil) || conn.dead){continue;}
					_, err = conn.client.Write(data[:]);
					if (err != nil){
						// Kill any closed sockets and continue
						if (errors.Is(err, io.EOF)){
							conn.dead = true;
							conn.client.Close();
							continue;
						}
						fmt.Printf("serverMain: unable to send MSG pkt: %s\n", err);
						err = fmt.Errorf("serverMain: %s", err);
						brk = true;
						continue;
					}
				}
			}
			case common.PktDCN:{
				//fmt.Printf("%s disconnected\n", connection.client.LocalAddr().String());
				brk = true;
				continue;
			}
			case common.PktMDF:{
				// Decode the message
				jsonRaw, err := common.DecodeMessage(&readPKT);
				if (err != nil){
					fmt.Printf("serverHandler.connectionMain: Unable to decode PktMDF packet: %s", err);
					err = fmt.Errorf("serverHandler.conncetionMain: %s", err);
					brk = true;
					continue;
				}
				
				asBytes := []byte(strings.Trim(jsonRaw, "\x00"));

				var jsonPkt common.ClientModifcation;
				err = json.Unmarshal(asBytes, &jsonPkt);
				if (err != nil){
					fmt.Printf("serverHandler.connectionMain: Unable to decode PktMDF payload: %s", err);
					err = fmt.Errorf("serverHandler.conncetionMain: %s", err);
					brk = true;
					continue;
				}
				
				changeNickname(server, connection, jsonPkt.NewName);
			}
			}
		}
		case CurrentIns := <- connection.instructions:{
			if (CurrentIns == ServerStop) {
				fmt.Printf("Shutting down connection\n");
				brk = true;
				continue;
			}
		}
		}
	}

	AnnounceMsg(server, fmt.Sprintf("%s disconnected from the room", connection.nickname));
	connection.dead = true;
	connection.client.Close();
	server.childThreads.Done();
	fmt.Printf("connectionHandler done\n");
	return err;
}

// Continuously accepts connections and runs a new goroutine running 
// connectionMain to serve it
func serverMain(server *ServerRoom){
	defer server.mainThread.Done();
	
	inbound := make(chan net.Conn);

	subThreads := sync.WaitGroup{};

	subThreads.Add(1);
	go func(){
		defer subThreads.Done();

		for {
			var conn net.Conn;
			conn, err := server.socket.Accept();
			if (err != nil){
				if !(errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed)){
					fmt.Printf("serverMain: unable to accept connection: %s\n",err);
				}
				break;
			}

			inbound <- conn;
		}
	}()

	brk := false;
	for {
		if (brk) {break;}
		select {
		case newConnection := <- inbound:{
			createConnection(&serv, newConnection);
		}
		case currentInstruction := <- server.instructions:{
			if (currentInstruction == ServerStop){
				// Close any non-dead connections
				for _, conn := range serv.clients{
					if ((conn == nil) || conn.dead){
						continue;
					}
					conn.instructions <- ServerStop;
				}
				brk = true;
				continue;
			}
		}
		}
	}
	serv.socket.Close();
	serv.childThreads.Wait();

	subThreads.Wait();
}

//
func createConnection(server *ServerRoom, inboundConnection net.Conn) (error){
	newConn := serverConnection{
		client: inboundConnection,
		instructions: make(chan int8, 1),
		dead: false,
	}
	
	var accept bool;
	var index int16 = -1;
	// Manually count the connections since we need to account for dead connections
	{
		count := 0;
		for ind, val := range server.clients {
			if (val.dead){
				index = int16(ind);
				// We don't need to go further because we've already found
				// a slot to place the connection into
				break;
			} else {
				count ++;
			}
		}
		if (count < int(serv.maxClients)){
			accept = true;
		} else {
			accept = false;
		}
	}

	//fmt.Printf("serverHandler: beginning handshake with %s\n", newConn.client.RemoteAddr());
	result, err := handleHandshake(&newConn, accept);
	if (err != nil){
		return fmt.Errorf("createConnection: %s", err);
	}
	if (!result){
		fmt.Printf("serverHandler: Could not finish handshake\n");
		return nil;
	}
	//fmt.Printf("serverHandler: completed handshake with %s\n", newConn.client.RemoteAddr());

	if (index != -1){
		server.clients[index] = &newConn;
	} else {
		// Otherwise append it onto the server
		server.clients = append(server.clients, &newConn);
		// reuse index here for assigning guest nicknames
		index = int16(len(server.clients));
	}

	// Assign the client a temp nickname
	if (newConn.nickname == ""){
		newConn.nickname = fmt.Sprintf("guest%d", index);
	}
	AnnounceMsg(server, fmt.Sprintf("%s has joined the room", newConn.nickname));	

	// And fork a new connectionHandler to serve it
	server.childThreads.Add(1);
	go connectionMain(&newConn, server);

	return nil;
}