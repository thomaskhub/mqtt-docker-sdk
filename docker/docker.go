package docker

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	sdkClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type Docker struct {
	dockerClient   *sdkClient.Client
	enabled        bool
	networkId      string
	networkSubnet  string
	networkGateway string
}

type ContainerEventData struct {
	ID       string
	Name     string
	Image    string
	Status   string
	ExitCode string
}

func (d *Docker) Init(networkId, networkSubnet string, networkGateway string) error {
	var err error
	d.dockerClient, err = sdkClient.NewClientWithOpts(sdkClient.FromEnv)
	if err != nil {
		return err
	}

	d.networkId = networkId
	d.networkSubnet = networkSubnet
	d.networkGateway = networkGateway
	return nil
}

func (d *Docker) Enabled() bool {
	return d.enabled
}

func (d *Docker) NetworkCreate(id, subnet string) (string, string, error) {
	ctx := context.Background()

	options := map[string]string{
		// "com.docker.network." + id + ".enable_icc":           "true",
		// "com.docker.network." + id + ".enable_ip_masquerade": "true",
		// "com.docker.network." + id + ".host_binding_ipv4":    "0.0.0.0",
		// "com.docker.network.driver.mtu":                      "1500",
	}

	netData, err := d.dockerClient.NetworkInspect(ctx, id, types.NetworkInspectOptions{})
	if err != nil {
		//Some error so network might not exist

		resp, err := d.dockerClient.NetworkCreate(ctx, id, types.NetworkCreate{
			Driver: "bridge",

			IPAM: &network.IPAM{
				Config: []network.IPAMConfig{
					{Subnet: subnet, Gateway: d.networkGateway},
				},
			},
			Options: options,
		})

		if err != nil {
			return "", "", err
		}

		return resp.ID, resp.Warning, nil
	}

	return netData.ID, "", nil
}

func (d *Docker) PullImage(imageName string) {
	ctx := context.Background()
	d.dockerClient.ImagePull(ctx, imageName, types.ImagePullOptions{})
}

// function that checks if an image exists
func (d *Docker) ImageExists(imageName string) bool {
	ctx := context.Background()
	images, _ := d.dockerClient.ImageList(ctx, types.ImageListOptions{})

	for _, image := range images {
		for _, tag := range image.RepoTags {

			//check if tag starts with :imageName
			if strings.HasPrefix(tag, imageName) {
				return true
			}
		}
	}

	return false
}

// checks if a container is running
// return bool --> true: container is running, false container is stopped
// return *string -->  "": container not found, id of the cointainer if it was found
func (d *Docker) ContainerRunning(containerName string) (bool, string) {
	ctx := context.Background()
	containerList, _ := d.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: false})
	for _, container := range containerList {

		if container.Names[0] == fmt.Sprintf("/%s", containerName) {
			return true, container.ID
		}
	}

	//Check if its under the running servers --> there must be an easier way than this
	containerList, _ = d.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	for _, container := range containerList {

		if container.Names[0] == fmt.Sprintf("/%s", containerName) {
			return false, container.ID
		}
	}

	return false, ""
}

// create a docker container and directly start it
func (d *Docker) ContainerCreateAndStart(imageName, user, containerName, restart, ip string, ports, volumes, environment, commands []string) (string, []string, error) {
	ctx := context.Background()

	d.PullImage(imageName)

	//prepare ports
	exposedPorts := make(nat.PortSet)
	portBinding := nat.PortMap{}
	for _, port := range ports {
		tmp := strings.Split(port, ":")
		portBinding[nat.Port(tmp[0])] = []nat.PortBinding{
			{
				// HostIP:   "0.0.0.0",
				HostPort: tmp[1],
			},
		}
		port := nat.Port(tmp[0])
		exposedPorts[port] = struct{}{}
	}

	//prepare mounts
	mounts := []mount.Mount{}

	for _, volume := range volumes {
		tmp := strings.Split(volume, ":")
		m := mount.Mount{
			Type:   mount.TypeBind,
			Source: tmp[0],
			Target: tmp[1],
		}
		mounts = append(mounts, m)
	}

	config := container.Config{
		Tty:             false,
		AttachStdin:     true,
		AttachStdout:    true,
		Env:             environment,
		WorkingDir:      "/app", //every docker container that needs sources must put it under /app
		Image:           imageName,
		User:            user,
		ExposedPorts:    exposedPorts,
		NetworkDisabled: false,
		Cmd:             commands,
		Hostname:        containerName,
	}

	hostConfig := container.HostConfig{
		Mounts:       mounts,
		AutoRemove:   false,
		PortBindings: portBinding,
		RestartPolicy: container.RestartPolicy{
			Name: restart,
		},
		// LogConfig:   logConfig,
		NetworkMode: container.NetworkMode(d.networkId),
	}

	endPoint := network.EndpointSettings{
		Aliases:   []string{containerName},
		NetworkID: d.networkId,
		IPAMConfig: &network.EndpointIPAMConfig{
			IPv4Address: ip,
		},
	}

	endpointsConfig := make(map[string]*network.EndpointSettings)
	endpointsConfig[d.networkId] = &endPoint

	dockCont, err := d.dockerClient.ContainerCreate(ctx, &config, &hostConfig, &network.NetworkingConfig{
		EndpointsConfig: endpointsConfig,
	}, nil, containerName)
	if err != nil {
		return "", nil, err
	}

	err = d.dockerClient.ContainerStart(ctx, dockCont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", nil, err
	}

	return dockCont.ID, dockCont.Warnings, err
}

func (d *Docker) ContainerEvents(eve chan<- ContainerEventData) ContainerEventData {
	ctx := context.Background()

	// Create a filter for container create, die, and start events
	filter := filters.NewArgs()
	filter.Add("type", "container")
	filter.Add("event", "create")
	filter.Add("event", "die")
	filter.Add("event", "start")

	// Start listening to Docker events
	eventChan, errChan := d.dockerClient.Events(ctx, types.EventsOptions{Filters: filter})

	fmt.Println("Listening for container events...")

	for {
		select {
		case event := <-eventChan:
			switch event.Action {

			case "create":
				// fmt.Printf("Container %s created\n", event.Actor.ID)
				eve <- handleContainerEvent(event)

			case "die":
				// fmt.Printf("Container %s died\n", event.Actor.ID)
				eve <- handleContainerEvent(event)

			case "start":
				// fmt.Printf("Container %s started\n", event.Actor.ID)
				eve <- handleContainerEvent(event)
			}
		case err := <-errChan:
			// Handle errors
			log.Println("Error while listening to events:", err)
		}
	}
}

func handleContainerEvent(event events.Message) ContainerEventData {
	return ContainerEventData{
		ID:       event.Actor.ID,
		Name:     event.Actor.Attributes["name"],
		Image:    event.Actor.Attributes["image"],
		Status:   event.Status,
		ExitCode: event.Actor.Attributes["exitCode"],
	}
}
