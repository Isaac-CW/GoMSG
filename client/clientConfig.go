package client

import (
	"encoding/json"
	"fmt"
	"os"
)

// Handles everything to do with loading, parsing and saving the config
type savedRoom struct {
	Addr string;
	Alias string;
}

// Config stores all the configuration values for the 
type Config struct{
	// This is fed to a server to automatically set
	// the name of but can be overridden by the server
	DefaultName string;	

	// SavedRooms stores an array of the rooms that the user has saved and can
	// connect to by its alias
	SavedRooms []savedRoom;
}

// WriteConfig is the yang to ReadConfig's yin and writes the contents of the
// config struct as a JSON file to clientConfig.cfg
// If the file doesn't exist. it is created.
func WriteConfig(session *ClientSession, FilePath string) (error){
	file, err := os.Create(FilePath + string(os.PathSeparator) + "clientConfig.cfg")
	if (err != nil){
		fmt.Printf("WriteConfig: Unable to save client config: %s\n", err);
		return fmt.Errorf("WriteConfig: %s", err);
	}

	var buf []byte;
	buf, err = json.MarshalIndent(session.Config, "","\t");

	if (err != nil){
		fmt.Printf("WriteConfig: Unable to marshal JSON: %s\n", err);
		return fmt.Errorf("WriteConfig: %s", err);
	}

	_, err = file.Write(buf);
	if (err != nil){
		fmt.Printf("WriteConfig: Unable to write JSON to file: %s\n", err);
		return fmt.Errorf("WriteConfig: %s", err);
	}

	return nil;
}

// ReadConfig parses the .cfg file at FilePath and returns a ClientConfig pointer
// with the parsed configuration values
// The cfg file is formatted in JSON
func ReadConfig(session *ClientSession, FilePath string) (error){
	file, err := os.Open(FilePath);
	if (err != nil){
		fmt.Printf("clientMain: Unable to open file at path %s: %s\n", FilePath, err);
		return fmt.Errorf("clientMain.ReadConfig: %s", err);
	}

	var data []byte = make([]byte, 4096);
	_, err = file.Read(data);
	if (err != nil){
		fmt.Printf("clientMain.ReadConfig: Unable to read file at path %s: %s\n", FilePath, err);
		return fmt.Errorf("clientMain.ReadConfig: %s", err);
	}
	// Find the first occurrence of the null byte and slice to right before that
	var sliceLimit int = 1;
	for k,v := range data{
		if (v == 0){
			sliceLimit = k;
			break;
		}
	}

	var retCFG Config;
	err = json.Unmarshal(data[:sliceLimit], &retCFG);
	if (err != nil){
		fmt.Printf("clientMain: Unable to parse CFG file for config values: %s", err);
		return fmt.Errorf("clientMain.ReadConfig: %s", err);
	}

	file.Close();

	session.Config = &retCFG;

	var nameOccurrence map[string]int = map[string]int{};
	// Resolve any colliding savenames
	for index := range retCFG.SavedRooms{
		num, exists := nameOccurrence[retCFG.SavedRooms[index].Alias];
		if (exists){
			newAlias := retCFG.SavedRooms[index].Alias + fmt.Sprintf("%d", num);
			fmt.Printf("The alias %s, already exists as a saved alias. Changing name from %s -> %s\n", 
				retCFG.SavedRooms[index].Alias, 
				retCFG.SavedRooms[index].Alias, 
				newAlias);
			retCFG.SavedRooms[index].Alias = newAlias;
		} else {
			nameOccurrence[retCFG.SavedRooms[index].Alias] = 0;
		}

		nameOccurrence[retCFG.SavedRooms[index].Alias] = num + 1;
	}

	return nil;
}

// GetSavedRoom returns the address associated with the alias
// if no alias exists then the function returns a blank string
func GetSavedRoom(session *ClientSession, Alias string) (string, error){
	if (session.Config == nil){
		return "", fmt.Errorf("clientConfig: config has not been loaded yet");
	}

	for _, savedRoom := range session.Config.SavedRooms{
		if (savedRoom.Alias == Alias){
			return savedRoom.Addr, nil;
		}
	}

	return "", nil;
}

// DisplaySavedAliases prints all aliases and their addresses to stdout
func DisplaySavedAliases(session *ClientSession){
	for ind, room := range session.Config.SavedRooms{
		fmt.Printf("%d) : Alias: %s, Address: %s\n",ind, room.Alias, room.Addr);
	}
}