package rpc

import (
	"encoding/json"
	"sync"

	"github.com/thomaskhub/mqtt-docker-sdk/docker"
	"go.uber.org/zap"
)

var startLock sync.Mutex

func (r *Rpc) HandleStartDocker(req *RpcReq) *RpcResp {
	//Ensure that we can call this only one after another
	startLock.Lock()
	defer startLock.Unlock()

	r.logger.Debug("Handle the start of the docker container", zap.Any("request", req.Params))

	params := &RpcStartDockerParams{}
	paramsJson, err := json.Marshal(req.Params)
	if err != nil {
		return &RpcResp{
			Id:      req.Id,
			Jsonrpc: "2.0",
			Error: &RpcErr{
				Code:  RCP_ERR_CODE_INTERNAL_ERROR,
				Error: "could not convert parameters (marshal)",
			},
			Result: nil,
		}
	}

	err = json.Unmarshal(paramsJson, params)
	if err != nil {
		return &RpcResp{
			Id:      req.Id,
			Jsonrpc: "2.0",
			Error: &RpcErr{
				Code:  RCP_ERR_CODE_INTERNAL_ERROR,
				Error: "could not convert parameters (unmarshal)",
			},
		}
	}

	//TODO: white list is not needed because we now have the check if its in the local registry
	// if its not on the machine we do not start it. This ways no one can just start their own containers
	// ont it
	// for i, name := range r.dockerImgWhiteList {
	// 	tmp := strings.Split(params.ImageName, ":") //split name and version
	// 	if strings.HasPrefix(tmp[0], name) {
	// 		break
	// 	}

	// 	if i == len(r.dockerImgWhiteList)-1 {
	// 		return &RpcResp{
	// 			Id:      req.Id,
	// 			Jsonrpc: "2.0",
	// 			Error: &RpcErr{
	// 				Code:  RPC_ERR_CODE_INVALID_PARAMETERS,
	// 				Error: fmt.Sprintf("requested docker image is not white listed [%s]", params.ImageName),
	// 			},
	// 		}
	// 	}
	// }

	//check if the docker image exists throw an error. Its the callers responsibiliry
	//to ensure they only call images available on the system
	imageExists := r.dockerClient.ImageExists(params.ImageName)
	if !imageExists {
		return &RpcResp{
			Id:      req.Id,
			Jsonrpc: "2.0",
			Error: &RpcErr{
				Code:  RPC_ERR_CODE_DOCKER_IMAGE_NOT_FOUND,
				Error: "could not clonde specified git repository",
			},
		}
	}

	//
	// Configure and Create the container, hock it up to the network and start its
	//
	id, warnings, err := r.dockerClient.ContainerCreateAndStart(
		params.ImageName,
		params.User,
		params.ContainerName,
		params.Restart,
		"", //ip address will not be used from docker start job as of know
		params.Ports,
		params.Volumes,
		params.Environment,
		params.Commands,
	)

	if err != nil {
		return &RpcResp{
			Id:      req.Id,
			Jsonrpc: "2.0",
			Error: &RpcErr{
				Code:  RCP_ERR_CODE_INTERNAL_ERROR,
				Error: err.Error(),
			},
		}
	}

	return &RpcResp{
		Id:      req.Id,
		Jsonrpc: "2.0",
		Result: StartDockerResult{
			ContainerId: id,
			Warnings:    warnings,
		},
	}
}

func (r *Rpc) HandleEventDocker(resp chan *RpcResp2) *RpcResp2 {
	event := make(chan docker.ContainerEventData)
	go r.dockerClient.ContainerEvents(event)

	// if err != nil {
	// 	return &RpcResp{
	// 		Id:      req.Id,
	// 		Jsonrpc: "2.0",
	// 		Error: &RpcErr{
	// 			Code:  RCP_ERR_CODE_INTERNAL_ERROR,
	// 			Error: err.Error(),
	// 		},
	// 	}
	// }

	go func() {
		for {
			lastContainerEventData := <-event

			resp <- &RpcResp2{
				Id:      12,
				Jsonrpc: "2.0",
				Method:  EventMapping[lastContainerEventData.Status],
				Result: EventsDockerResult{
					ContainerId: lastContainerEventData.ID,
					Image:       lastContainerEventData.Image,
					Name:        lastContainerEventData.Name,
					Status:      lastContainerEventData.Status,
					ExitCode:    lastContainerEventData.ExitCode,
				},
			}
		}
	}()
	return nil
}
