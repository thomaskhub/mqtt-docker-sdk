package utils

import (
	"encoding/json"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

func ConvertHostInfoToJson() (string, error) {
	filePath := "/etc/linux-hostinfo/hostinfo.yaml"

	// Read the hostinfo.yaml file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("error: %v", err)
		return "", err
	}

	// Convert the hostinfo.yaml to JSON
	jsonData, err := YamlToJson(data)
	if err != nil {
		log.Fatalf("error: %v", err)
		return "", err
	}

	return string(jsonData), nil
}

func YamlToJson(data []byte) ([]byte, error) {
	var result interface{}

	// Convert YAML to JSON
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		log.Fatalf("error: %v", err)
		return nil, err
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("error: %v", err)
		return nil, err
	}

	return jsonData, nil
}

func GetHostInfoByteAndMap() ([]byte, map[string]interface{}, error) {
	filePath := "/etc/linux-hostinfo/hostinfo.yaml"

	// Read the hostinfo.yaml file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("error: %v", err)
		return nil, nil, err
	}

	m := make(map[string]interface{})

	// Unmarshal the YAML data into the map
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		log.Fatalf("error: %v", err)
		return nil, nil, err
	}

	return data, m, nil
}
