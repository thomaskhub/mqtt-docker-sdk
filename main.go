package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/thomaskhub/mqtt-docker-sdk/client"
	"github.com/thomaskhub/mqtt-docker-sdk/docker"
	"github.com/thomaskhub/mqtt-docker-sdk/rpc"
	"github.com/thomaskhub/mqtt-docker-sdk/utils"
	"go.uber.org/zap"
)

var dockerClient docker.Docker = docker.Docker{}

func main() {

	logger := utils.Logger{}
	logger.Init(utils.LOGGER_MODE_DEBUG)

	logger.Debug("Starting mqtt-docker-sdk")

	configFile := flag.String("config", "config.yaml", "Path to the config file")
	flag.Parse()

	//befire starting check if git command is installed on the server because it is needed
	// Check if git command is installed
	_, err := exec.LookPath("git")
	if err != nil {
		log.Fatal("Git command is not installed on the server")
	}

	cfg := utils.ParseConfig(*configFile)
	cfg.PubTopic = `from/` + cfg.Mqtt.AppName
	cfg.SubTopic = `to/` + cfg.Mqtt.AppName

	err = dockerClient.Init(
		cfg.Docker.NetworkId,
		cfg.Docker.NetworkSubnet,
		cfg.Docker.NetworkGateway,
	)
	if err != nil {
		logger.Fatal("could not init docker client")
		os.Exit(1)
	}

	//now create the network
	_, _, err = dockerClient.NetworkCreate(
		cfg.Docker.NetworkId,
		cfg.Docker.NetworkSubnet,
	)

	if err != nil {
		logger.Fatal("could not create the docker network", zap.Error(err))
		os.Exit(1)
	}

	//docker is now ready to be called via mqtt
	client := client.NewMqttClientWithConfig(
		cfg.Mqtt.Broker,
		cfg.Mqtt.ClientId,
		cfg.Mqtt.Username,
		cfg.Mqtt.Password,
	)

	//Prepare RPC interface
	r := rpc.Rpc{}
	r.Init(utils.LOGGER_MODE_DEBUG, &dockerClient)
	r.AddHandler(rpc.RPC_METHOD_START_DOCKER, r.HandleStartDocker)

	//this hangs until the broker becomes available
	client.Connect()

	//handle rpc requests
	rxMsg := func(c mqtt.Client, message mqtt.Message) {

		rpcReq := rpc.RpcReq{}
		err := json.Unmarshal(message.Payload(), &rpcReq)

		if err != nil {
			fmt.Printf("could not unmarshal rpc request: %v\n", err)
			return
		}

		resp := r.HandleRpcCall(&rpcReq)
		respString, _ := json.Marshal(resp)
		if resp != nil {
			client.Publish(cfg.PubTopic, respString, 2)
		}
	}

	client.Subscribe(cfg.SubTopic, rxMsg, 2)

	select {}
}
