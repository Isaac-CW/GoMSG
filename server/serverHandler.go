package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"p2psystem/common"
	"time"
)

type serverConnection struct{
	client net.Conn;
	nickname string;
	dead bool;	// true if the socket is closed
	instructions chan int8;
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
	//fmt.Printf("serverHandshake: accepted ACK packet\n");

	return true, nil;
}

func connectionMain(connection *serverConnection, server *serverRoom) (error){
	inbound := make(chan bool);
	var data [common.PktBufferSize]byte;

	connection.client.SetReadDeadline(time.Time{});

	go func(){
		for {
			_, err := connection.client.Read(data[:]);
			//fmt.Printf("connectionMain: Read data from %s\n", connection.client.RemoteAddr());
			if (err != nil){
				if (errors.Is(err, io.EOF)){
					return;
				}
				fmt.Printf("serverMain: Unable to read from client: %s\n", err);
				return;
			}

			inbound <- true;
		}
	}()

	brk := false;
	
	for {
		if (brk){break;}
		select {
		case <- inbound:{
			//fmt.Printf("serverMain: received packet\n");
			var readPKT common.MsgPacket = common.DeserializePacket(data[:]);
			readPKT.SendNickname = connection.nickname;
			
			switch readPKT.PktType{
			case common.PktMSG:{
				err := common.SerializePacket(&readPKT, data[:]);
				if (err != nil){
					fmt.Printf("serverMain: unable to seralize MSG pkt: %s\n", err);
					return fmt.Errorf("serverMain: %s", err);
				}
				// Sling it to every other client
				for _, conn := range server.clients{
					_, err = conn.client.Write(data[:]);
					if (err != nil){
						// Kill any closed sockets and continue
						if (errors.Is(err, io.EOF)){
							conn.dead = true;
							conn.client.Close();
							continue;
						}
						fmt.Printf("serverMain: unable to send MSG pkt: %s\n", err);
						return fmt.Errorf("serverMain: %s", err);
					}
				}
			}
			case common.PktDCN:{
				fmt.Printf("%s disconnected\n", connection.client.LocalAddr().String());
				connection.dead = true;
				connection.client.Close();
				brk = true;
				continue;
			}
			}
		}
		case CurrentIns := <- connection.instructions:{
			if CurrentIns == -1 {
				connection.client.Close();
				brk = true;
				continue;
			}
		}
		}
	}
	connection.dead = true;
	connection.client.Close();
	return nil;
}

//
func createConnection(server *serverRoom, inboundConnection net.Conn) (error){
	newConn := serverConnection{
		client: inboundConnection,
		instructions: make(chan int8),
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
	newConn.nickname = fmt.Sprintf("guest%d", index);
	AnnounceMsg(server, fmt.Sprintf("%s has joined the room", newConn.nickname));	

	// And fork a new connectionHandler to serve it
	go connectionMain(&newConn, server);

	return nil;
}