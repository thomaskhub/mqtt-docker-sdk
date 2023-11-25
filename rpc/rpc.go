package rpc

import (
	"fmt"

	"github.com/thomaskhub/mqtt-docker-sdk/docker"
	"github.com/thomaskhub/mqtt-docker-sdk/utils"
)

const (
	CONTAINER_START  = "start"
	CONTAINER_CREATE = "create"
	CONTAINER_DIE    = "die"
)

var EventMapping = map[string]string{
	CONTAINER_START:  "docker_event_start",
	CONTAINER_CREATE: "docker_event_create",
	CONTAINER_DIE:    "docker_event_die",
}

const (
	RPC_METHOD_START_DOCKER = "start_docker"
	RPC_METHOD_ERROR_DOCKER = "error_docker"
	RPC_METHOD_STOP_DOCKER  = "stop_docker"
)

const (
	RPC_ERR_CODE_PARSE_ERROR            = -32700
	RPC_ERR_CODE_INVALID_REQUEST        = -32600
	RPC_ERR_CODE_METHOD_NOT_FOUND       = -32601
	RPC_ERR_CODE_INVALID_PARAMETERS     = -32602
	RCP_ERR_CODE_INTERNAL_ERROR         = -32603
	RPC_ERR_CODE_DOCKER_IMAGE_NOT_FOUND = -32604
)

type RpcReq struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type RpcErr struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

type RpcResp struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RpcErr     `json:"error,omitempty"`
}

type RpcStartDockerParams struct {
	ImageName     string `json:"imageName"`
	ContainerName string `json:"containerName"`
	Restart       string `json:"restart,omitempty"`
	User          string `json:"user,omitempty"`
	// GitUrl        string   `json:"gitUrl,omitempty"`
	// GitBranch     string   `json:"gitBranch,omitempty"`
	Environment []string `json:"environment,omitempty"`
	Ports       []string `json:"ports,omitempty"`
	Volumes     []string `json:"volumes,omitempty"`
	Commands    []string `json:"commands,omitempty"`
}

type StartDockerResult struct {
	ContainerId string   `json:"containerId"`
	Warnings    []string `json:"warnings"`
}

type RpcHandler func(req *RpcReq) *RpcResp

type Rpc struct {
	handlerMap map[string]RpcHandler
	logger     utils.Logger
	// dockerImgWhiteList []string
	dockerClient *docker.Docker
}

type EventsDockerResult struct {
	ContainerId string `json:"containerId"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	ExitCode    string `json:"exitCode,omitempty"`
}

// func (r *Rpc) Init(loggerMode string, dockerImgWhiteList []string, dockerClient *docker.Docker) {
func (r *Rpc) Init(loggerMode string, dockerClient *docker.Docker) {
	r.handlerMap = make(map[string]RpcHandler)
	r.logger = utils.Logger{}
	r.logger.Init(loggerMode)
	// r.dockerImgWhiteList = dockerImgWhiteList
	r.dockerClient = dockerClient
}

func (r *Rpc) AddHandler(name string, handler RpcHandler) {
	r.handlerMap[name] = handler
}

func (r *Rpc) HandleRpcCall(req *RpcReq) *RpcResp {
	// if len(req.Jsonrpc) <= 0 {
	// 	return nil
	// }

	if req.Jsonrpc != "2.0" {
		return &RpcResp{
			Jsonrpc: "2.0",
			Id:      req.Id,
			Error: &RpcErr{
				Code:  RPC_ERR_CODE_INVALID_REQUEST,
				Error: "rpc version not supported",
			},
		}
	}

	fmt.Printf("req.Method: %v\n", req.Method)

	if _, ok := r.handlerMap[req.Method]; !ok {
		return &RpcResp{
			Jsonrpc: "2.0",
			Id:      req.Id,
			Error: &RpcErr{
				Code:  RPC_ERR_CODE_METHOD_NOT_FOUND,
				Error: fmt.Sprintf("method %s not found", req.Method),
			},
		}
	}

	return r.handlerMap[req.Method](req)
}
