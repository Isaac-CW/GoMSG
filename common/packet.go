package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	// NicknameMaxSize is the maximum number of characters that a nickname can be
	// anything greater should be truncated down to this value
	NicknameMaxSize = 64;
	// PktBufferSize is the size of the byte array to serialize and deserialize the packet into
	PktBufferSize = 4096;	

	// PktMSG indicates that the inbound packet's payload has a message to be
	// sent to all other clients.
	PktMSG = 1;		

	// PktANC is a server announcement sent from the server to all clients 
	// with a payload that has a message, similar to PktMSG. 
	PktANC = 2;		

	// PktREF is used in the connection process and used if the server refused
	// the client's connection
	PktREF = 3;

	// PktACP is short for ACCEPT and indicates that the server accepted the client's
	// connection and has the information of the server
	PktACP = 4;
	
	// PktACK is the acknowledgement packet sent from the client to server with
	// the client's info
	PktACK = 5;	

	// PktDCN is sent from the client and indicates the client wishes to gracefully
	// disconnect
	PktDCN = 6;
)

// MsgPacket is what is sent over sockets
type MsgPacket struct {
	PktType uint8
	Timestamp uint64
	SendNickname string;	// Filled in by the server
	PayloadSize uint16;
	Payload [2048]byte
}

// Encodes the given number in network order and returns an array of bytes
func encodeNumber64(number uint64, dest []byte){
	binary.BigEndian.PutUint64(dest, number);
}

func encodeNumber16(number uint16, dest []byte){
	binary.BigEndian.PutUint16(dest, number);
}

// Decodes the given number from network order and returns the decoded number
func decodeNumber64(array []byte) (uint64){
	var retVal uint64;
	retVal = binary.BigEndian.Uint64(array[:]);
	return retVal;
}

func decodeNumber16(array[]byte) (uint16){
	var retVal uint16;
	retVal = binary.BigEndian.Uint16(array[:]);
	return retVal;}

// SerializePacket takes a pointer to the given packet and writes its contents
// to the given destination array of bytes.
// Its assumed that the destination array is large enough to fit the packet
func SerializePacket(pkt *MsgPacket, dest []byte) (error){
	var cursor uint64 = 0;
	dest[cursor] = pkt.PktType;
	cursor ++;

	encodeNumber64(pkt.Timestamp, dest[cursor:])
	cursor += 8;

	copy(dest[cursor:], []byte(pkt.SendNickname));
	cursor += NicknameMaxSize;

	encodeNumber16(pkt.PayloadSize, dest[cursor:]);
	cursor += 2;

	copy(dest[cursor:], pkt.Payload[:]);

	return nil;
}

// DeserializePacket takes an array of bytes and returns a packet with the decoded
// information
func DeserializePacket(byteArray []byte) (MsgPacket){
	var pktType uint8;
	var timestamp uint64;
	var size uint16;
	var payload [2048]byte;

	var cursor uint64;

	pktType = byteArray[cursor];
	cursor ++;

	timestamp = decodeNumber64(byteArray[cursor:]);
	cursor += 8;

	nick := string(byteArray[cursor:(cursor + NicknameMaxSize)]);
	cursor += NicknameMaxSize;

	size = decodeNumber16(byteArray[cursor:]);
	cursor += 2;

	copy(payload[:], byteArray[cursor:]);

	return MsgPacket{PktType: pktType, Timestamp: timestamp, SendNickname: nick, PayloadSize: size, Payload: payload};
}

// EncodeMessage takes the given packet and string and compresses the string,
// sets the timestamp and the packet size. The packet is modified in place
func EncodeMessage(pkt *MsgPacket, content string) (error){
	var stringBuffer bytes.Buffer;

	pkt.Timestamp = uint64(time.Now().Unix());

	_, err := WriteTo(&stringBuffer, content);
	if (err != nil){
		return fmt.Errorf("packet: %s", err);
	}
	pkt.PayloadSize = uint16(stringBuffer.Len());
	copy(pkt.Payload[:], stringBuffer.Bytes());

	return nil;
}

// Decode message takes the given pointer and returns the string associated with
// the payload
func DecodeMessage(pkt *MsgPacket) (string, error){
	var retVal string;
	var err error;

	var buffer bytes.Buffer;
	buffer.Write(pkt.Payload[:pkt.PayloadSize]);

	retVal, err = ReadFrom(&buffer)
	if (err != nil){
		return "", err;
	}
	return retVal, nil;

}