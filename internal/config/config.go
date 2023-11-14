package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorkflowsDir string `yaml:"workflows-dir"`
	Port 	     int    `yaml:"port"`
}

func NewConfig(workflowsDir string, port int) *Config {
	return &Config{WorkflowsDir: workflowsDir, Port: port}
}

func LoadConfig() (*Config, error) {
	file, err := os.Open("config.yaml")
	if err != nil {
		return nil, err
	} else {
		decoder := yaml.NewDecoder(file)
		var config Config
		err = decoder.Decode(&config)
		if err != nil {
			return nil, err
		} else {
			return &config, nil
		}
	}
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	} else {
		file, err := os.OpenFile("config.yaml", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, err = file.Write(data)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (c *Config) Validate() error {
	if c.WorkflowsDir == "" {
		return fmt.Errorf("workflows directory not set")
	}
	if c.Port == 0 {
		return fmt.Errorf("port not set")
	}
	return nil
}

func SaveDefaultConfig() error {
	config := NewConfig("workflows", 8888)
	err := config.Save()
	if err != nil {
		return err
	}
	return nil
}

func ConfigFileExists() bool {
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func GetString(key string) string {
	config, err := LoadConfig()
	if err != nil {
		return ""
	}
	switch key {
	case "workflows-dir":
		return config.WorkflowsDir
	default:
		return ""
	}
}

func GetInt(key string) int {
	config, err := LoadConfig()
	if err != nil {
		return 0
	}
	switch key {
	case "port":
		return config.Port
	default:
		return 0
	}
}