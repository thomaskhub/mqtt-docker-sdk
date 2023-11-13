package client

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Topic struct {
	Topic string
	Cb    mqtt.MessageHandler
}

var topicList []Topic

type MqttClient struct {
	client mqtt.Client
}

// var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
// 	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
// }

func NewMqttClientWithConfig(broker string, clientId string, username string, password string) *MqttClient {
	topicList = make([]Topic, 0)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientId)
	opts.SetUsername(username)
	opts.SetPassword(password)
	// opts.SetDefaultPublishHandler(messagePubHandler)

	opts.SetConnectRetry(true)
	opts.SetAutoReconnect(true)
	opts.SetResumeSubs(true)

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		for _, topic := range topicList {
			token := client.Subscribe(topic.Topic, 2, topic.Cb)
			token.Wait()
		}
	})

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		fmt.Printf("Connection lost: %v", err)
	}
	opts.OnReconnecting = func(client mqtt.Client, options *mqtt.ClientOptions) {
		fmt.Printf("Reconnecting:\n")
	}

	client := mqtt.NewClient(opts)

	return &MqttClient{
		client: client,
	}
}

// connect to mqtt
func (m *MqttClient) Connect() error {
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MqttClient) IsConnected() bool {
	return m.client.IsConnected()
}

func (m *MqttClient) Disconnect() {
	m.client.Disconnect(0)
}

func (m *MqttClient) Publish(topic string, payload interface{}, qos byte) {
	token := m.client.Publish(topic, qos, false, payload)
	token.Wait()
}

func (m *MqttClient) Subscribe(topic string, callback mqtt.MessageHandler, qos byte) {
	topicList = append(topicList, Topic{Topic: topic, Cb: callback})
	if token := m.client.Subscribe(topic, qos, callback); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}
