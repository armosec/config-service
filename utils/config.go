package utils

import (
	"config-service/types"
	"fmt"
	"os"
	"sync"

	"github.com/tkanos/gonfig"
)

type DefaultConfigs struct {
	CustomerConfig *types.CustomerConfig `json:"customerConfig"`
}

type Configuration struct {
	Port           string          `json:"port"`
	Telemetry      TelemetryConfig `json:"telemetry"`
	Mongo          MongoConfig     `json:"mongo"`
	LoggerConfig   LoggerConfig    `json:"logger"`
	AdminUsers     []string        `json:"admins"`
	DefaultConfigs *DefaultConfigs `json:"defaultConfigs"`
}

type TelemetryConfig struct {
	JaegerAgentHost string `json:"jaegerAgentHost"`
	JaegerAgentPort string `json:"jaegerAgentPort"`
}

type LoggerConfig struct {
	Level       string `json:"level"`
	LogFileName string `json:"logFileName"`
}

type MongoConfig struct {
	Host       string `json:"host,omitempty"`
	Port       string `json:"port,omitempty"`
	User       string `json:"user,omitempty"`
	Password   string `json:"password,omitempty"`
	DB         string `json:"db,omitempty"`
	ReplicaSet string `json:"replicaSet"`
}

// globalConfig with defaults
var globalConfig = Configuration{
	Mongo: MongoConfig{
		Host: "localhost",
		Port: "27017",
		DB:   "caportalbe_db",
	},
}
var initOnce sync.Once

func GetConfig() Configuration {
	initOnce.Do(func() {
		configFileName := "config.json"
		if cfgPath := os.Getenv("CONFIG_PATH"); cfgPath != "" {
			fmt.Printf("<----- Config file from environment variable: %s ----->", cfgPath)
			configFileName = cfgPath
		}
		if err := gonfig.GetConf(configFileName, &globalConfig); err != nil {
			errMsg := fmt.Sprintf("Cannot open config file: %s, Error: %s", "config.json", err.Error())
			panic(errMsg)
		}
	})
	return globalConfig
}
