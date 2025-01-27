package configmanager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Fields map[string]interface{} `json:"fields"`
}

func InitConfig() error {
	if _, err := os.Stat("config.json"); err == nil {
		return nil
	}

	defaultConfig := Config{
		Fields: map[string]interface{}{
			"publickey_name":  "publickey",
			"privatekey_name": "privatekey",
			"seed_nodes":      []string{},
			"max_connections": 50,
		},
	}

	return SaveConfig(defaultConfig)
}

func GetConfigPath() string {
	currentDir, _ := os.Getwd()
	return filepath.Join(currentDir, "config.json")
}

func SetItem(key string, value interface{}, config *Config, dontReplaceIfExist bool) error {
	if config.Fields == nil {
		config.Fields = make(map[string]interface{})
	}

	if dontReplaceIfExist {
		if _, exists := config.Fields[key]; exists {
			return nil
		}
	}

	config.Fields[key] = value
	return SaveConfig(*config)
}

func ResetConfig() error {
	defaultConfig := Config{
		Fields: map[string]interface{}{
			"publickey_name":  "publickey",
			"privatekey_name": "privatekey",
			"seed_nodes":      []string{},
			"max_connections": 50,
		},
	}
	return SaveConfig(defaultConfig)
}

func SaveConfig(config Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile("config.json", data, 0644)
}

func DelItem(key string, config *Config) error {
	if config.Fields == nil {
		return fmt.Errorf("fields map is nil")
	}

	if _, exists := config.Fields[key]; exists {
		delete(config.Fields, key)
		return SaveConfig(*config)
	}

	return fmt.Errorf("field %s not found in config", key)
}

func LoadConfig() (Config, error) {
	var config Config

	data, err := os.ReadFile("config.json")
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}
