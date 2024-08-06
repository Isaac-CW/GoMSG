package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"p2psystem/common"
	"strings"
	"sync"
	"time"
)

//
func handleHandshake(session *ClientSession, connection *ClientConnection) (bool, error){
	var data []byte = make([]byte, common.PktBufferSize);

	//fmt.Printf("clientHandshake: connecting with %s from %s\n", connection.server.RemoteAddr(), connection.server.LocalAddr());
	connection.server.SetReadDeadline(time.Now().Add(4 * time.Second));
	_, err := connection.server.Read(data);
	if (err != nil){
		if (errors.Is(err, os.ErrDeadlineExceeded)){
			fmt.Printf("clientHandshake: server timed out\n");
			return false, nil;
		}

		fmt.Printf("clientHandshake: %s", err);
		return false, fmt.Errorf("clientHandshake: %s", err);
	}
	//fmt.Printf("clientHandshake: recieved packet\n");

	pkt := common.DeserializePacket(data);
	if (pkt.PktType == common.PktREF){
		fmt.Printf("clientHandshake: server refused connection\n");
		return false, nil;
	} else if (pkt.PktType != common.PktACP){
		fmt.Printf("clientHandshake: unrecognised packet type\n");
		return false, nil;
	}

	//fmt.Printf("clientHandshake: recevied ACP packet\n");
	// Then prepare the ACK packet
	pkt = common.MsgPacket{
		PktType: common.PktACK,
	}
	// Encode the data in
	var modifierpkt common.ClientModifcation = common.ClientModifcation{
		NewName: session.Config.DefaultName,
	}
	// Then write it to JSON
	ackPkt, err := json.Marshal(modifierpkt);
	if (err != nil){
		fmt.Printf("clientHandshake: unable to encode config to ACK packet: %s\n", err);
		return false, fmt.Errorf("clientHandshake: %s", err);
	}
	err = common.EncodeMessage(&pkt, strings.Trim(string(ackPkt), "\x00"));
	if (err != nil){
		fmt.Printf("clientHandshake: unable to encode json: %s\n", err);
		return false, fmt.Errorf("clientHandshake: %s", err);
	}

	err = common.SerializePacket(&pkt, data);
	if (err != nil){
		fmt.Printf("clientHandshake: unable to serialize ACK packet: %s\n", err);
		return false, fmt.Errorf("clientHandshake: %s", err);
	}
	_, err = connection.server.Write(data);
	if (err != nil){
		fmt.Printf("clientHandshake: uanble to send ACK packet: %s\n", err);
		return false, fmt.Errorf("clientHandshake: %s", err);
	}

	return true, nil;
}

// clientHandler contains the functions used by the goroutine that's run
// when a connection is successfully established

func connMain(connection *ClientConnection) (error){
	fmt.Printf("Connected to %s\n",connection.server.RemoteAddr().String());

	var dataBuffer [common.PktBufferSize]byte;
	var canRead chan bool = make(chan bool);

	connection.server.SetReadDeadline(time.Time{});

	childThreads := sync.WaitGroup{};

	childThreads.Add(1);
	go func(){
		defer childThreads.Done();

		for {
			_, err := connection.server.Read(dataBuffer[:]);
			if (err != nil){
				if ((errors.Is(err, io.ErrUnexpectedEOF)) || (errors.Is(err, io.EOF))){
					canRead <- false;
					break;
				}
				// Silently handle errors here
				//fmt.Printf("clientMain: Unable to read from server: %s\n", err);
				break;
			}

			canRead <- true;
		}
	}();

	var brk bool = false;

	for {
		if (brk){break;}
		select {
		case active := <- canRead:{
			if (!active){
				brk = true;
				continue;
			}
			// Deserialize the packet
			pkt := common.DeserializePacket(dataBuffer[:]);
			switch pkt.PktType{
			case common.PktMSG:{
				timestamp := time.Unix(int64(pkt.Timestamp), 0);
				msg, err := common.DecodeMessage(&pkt);
				if (err != nil){
					fmt.Printf("clientMain: unable to decode packet message\n");
					return fmt.Errorf("clientMain: %s", err);
				}
				fmt.Printf("%s %s : %s\n", pkt.SendNickname, timestamp.Format(time.Kitchen), msg);
			}
			case common.PktANC:{
				timestamp := time.Unix(int64(pkt.Timestamp), 0);
				msg, err := common.DecodeMessage(&pkt);
				if (err != nil){
					fmt.Printf("clientMain: unable to decode packet message\n");
					return fmt.Errorf("clientMain: %s", err);
				}
				fmt.Printf("Server %s: %s\n", timestamp.Format(time.Kitchen), msg);
			}
			case common.PktKCK:{
				timestamp := time.Unix(int64(pkt.Timestamp), 0);
				msg, err := common.DecodeMessage(&pkt);
				if (err != nil){
					fmt.Printf("clientMain: unable to decode packet message\n");
					return fmt.Errorf("clientMain: %s", err);
				}

				fmt.Printf("Server %s: %s\n", timestamp.Format(time.Kitchen), msg);
			}
			}
		}
		case currentIns := <- connection.instructions:{
			switch currentIns{
				case ClientDisconnect:{
					// Prepare a PktDCN packet
					pkt := common.MsgPacket{PktType: common.PktDCN};
					common.SerializePacket(&pkt, dataBuffer[:]);

					_, err := connection.server.Write(dataBuffer[:]);

					if (err != nil){
						if !((err == io.EOF) || (err == io.ErrUnexpectedEOF)){
							return err;
						}
					}

					brk = true;
					continue;
				}
			}
		}
		}
	}

	connection.server.Close();
	childThreads.Wait();
	return nil;
}

// Performs the handshake with the given connection and if successful, adds it
// to the clientSession
func createConnection(session *ClientSession, connection net.Conn) (error){
	// Create a client connection
	newClient := ClientConnection{
		server: connection,
		instructions: make(chan uint8),
		dead: false,
	}

	status, err := handleHandshake(session,&newClient);

	if (err != nil){
		fmt.Printf("clientHandler.makeConnection: unable to complete handshake: %s", err);
		return fmt.Errorf("clientHandler.makeConnection: %s", err);
	}
	if (!status){
		return nil;
	}
	// Find the first suitible location in the session
	var indexToInsertTo int = -1;
	for ind, val := range session.connectedServers{
		if (val == nil){continue;}
		if (val.dead){
			indexToInsertTo = ind;
			break;
		}
	}
	if (indexToInsertTo != -1){
		session.connectedServers[indexToInsertTo] = &newClient;
	} else {
		session.connectedServers = append(session.connectedServers, &newClient);
	}
	session.CurrentConnection = &newClient;
	go connMain(&newClient);

	return nil;
}