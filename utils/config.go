package utils

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Mqtt struct {
	Broker               string `yaml:"broker"`    //url of the remote broker
	ClientId             string `yaml:"client_id"` //client id to connect to remote broker
	Username             string `yaml:"username"`  //username to connect to remote broker
	Password             string `yaml:"password"`  //password to connect to remote broker
	EnableHeartbeat      bool   `yaml:"enable_heartbeat"`
	HeartBeatInterval    int    `yaml:"heartbeat_interval"`
	BrokerPublishTopic   string `yaml:"broker_publish_topic"`
	BrokerSubscribeTopic string `yaml:"broker_subscribe_topic"`
}

type Docker struct {
	// ImageWhitelist []string `yaml:"image_whitelist"`
	NetworkId      string `yaml:"network_id"`
	NetworkSubnet  string `yaml:"network_subnet"`
	NetworkGateway string `yaml:"network_gateway"`
}
type Config struct {
	Docker  Docker `yaml:"docker"`
	Mqtt    Mqtt   `yaml:"mqtt"`
	AppName string `yaml:"app_name"`
}

func ParseConfig(file string) *Config {
	// Read the yaml file
	cfg := Config{}
	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Unmarshal the yaml file into the config struct
	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// Parse the flags
	flag.Parse()

	return &cfg
}
