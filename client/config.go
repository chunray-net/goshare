package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	ServerAddress  string `json:"server_address"`
	StoragePath    string `json:"storage_path"`
	AutoStart      bool   `json:"auto_start"`
	MinimizeToTray bool   `json:"minimize_to_tray"`
}

func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config.json")

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func LoadConfig() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(configDir, "config.json")

	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultConfig := &Config{
			ServerAddress:  "123.60.47.228:9000",
			StoragePath:    ".",
			AutoStart:      false,
			MinimizeToTray: true,
		}
		if err := defaultConfig.Save(); err != nil {
			return nil, err
		}
		return defaultConfig, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// 确保必要的字段有默认值
	if config.ServerAddress == "" {
		config.ServerAddress = "123.60.47.228:9000"
	}
	if config.StoragePath == "" {
		config.StoragePath = "."
	}

	return &config, nil
}

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".goshare"), nil
}

func (c *Config) GetStoragePath() string {
	if c.StoragePath == "" || c.StoragePath == "." {
		return "."
	}
	return c.StoragePath
}
