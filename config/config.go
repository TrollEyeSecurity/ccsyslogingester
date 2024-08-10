package config

import (
	"encoding/json"
	"os"
)

func LoadConfiguration(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		return config, err
	}
	if configFile != nil {
		jsonParser := json.NewDecoder(configFile)
		e := jsonParser.Decode(&config)
		if e != nil {
			return config, e
		}
		configFile.Close()
	}
	return config, nil
}
