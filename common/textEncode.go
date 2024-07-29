package common

import (
	"compress/flate"
	"fmt"
	"io"
)

// Handles compressing and decompressing text

// WriteTo writes the given string into the writeInt
func WriteTo(writeInt io.Writer, content string) (int, error){

	flateWriter, err := flate.NewWriter(writeInt, flate.BestCompression);

	encodeBuffer := make([]byte, 1024);
	copy(encodeBuffer, []byte(content));

	_, err = flateWriter.Write(encodeBuffer);

	if (err != nil){
		return 0, fmt.Errorf("TextEncode: unable to write to buffer: %s", err);
	}

	err = flateWriter.Flush();
	if (err != nil){
		return 0, fmt.Errorf("TextEncode: unable to flush writer: %s", err);
	}

	err = flateWriter.Close();
	if (err != nil){
		return 0, fmt.Errorf("TextEncode: unable to close writer: %s", err);
	}

	return 0, nil;
}

// ReadFrom reads from the given readInt and decodes it into a string
func ReadFrom(readInt io.Reader) (string, error){
	var data []byte = make([]byte, 1024);
	flateReader := flate.NewReader(readInt);

	_, err := io.ReadFull(flateReader, data);
	if (err != nil){
		if (err == io.ErrUnexpectedEOF){
			return "", err
		}

		return "", fmt.Errorf("TextEncode: unable to read from reader: %s", err);
	}

	//fmt.Printf("Read %d bytes from reader\n", bytesRead);
	return string(data), nil;
}