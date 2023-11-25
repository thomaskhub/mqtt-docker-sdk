package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

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

	// hostinfo, err := utils.ConvertHostInfoToJson()
	hostinfoByte, hostinfoMap, err := utils.GetHostInfoByteAndMap()
	if err != nil {
		logger.Fatal("could not convert hostinfo to json", zap.Error(err))
	}

	cfg := utils.ParseConfig(*configFile)
	cfg.Mqtt.BrokerSubscribeTopic = cfg.AppName + "/" + hostinfoMap["instance_id"].(string)
	cfg.Mqtt.BrokerPublishTopic = cfg.AppName + "/cmd/" + hostinfoMap["instance_id"].(string)

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
		hostinfoMap["instance_id"].(string), //client id

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
			client.Publish(cfg.Mqtt.BrokerPublishTopic, respString, 2)
		}
	}

	client.Subscribe(cfg.Mqtt.BrokerSubscribeTopic, rxMsg, 2)

	eventsChannel := make(chan *rpc.RpcReq)
	r.HandleEventDocker(eventsChannel)
	go func() {
		for {
			select {
			case event := <-eventsChannel:
				jsonData, _ := json.Marshal(event)
				client.Publish(cfg.Mqtt.BrokerPublishTopic, jsonData, 2)
			}
		}
	}()

	// Heartbeat
	go func() {
		//send hostinfo every 30 seconds
		ticker := time.NewTicker(time.Duration(cfg.Mqtt.HeartBeatInterval) * time.Second)

		for {
			<-ticker.C
			client.Publish(cfg.Mqtt.BrokerPublishTopic, hostinfoByte, 2)
		}
	}()

	select {}
}
