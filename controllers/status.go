package controllers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

var (
	defaultHeaders = map[string]string{"User-Agent": "squid-1.0"}
)

func init() {
	os.Setenv("JSON", "yes")
}

func GetStatus(c *gin.Context) {
	services, err := getServices()
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, services)
}

func getServices() ([]Service, error) {
	containers, err := dockerStatus()
	if err != nil {
		return nil, err
	}

	composes, err := listComposes()
	if err != nil {
		return nil, err
	}

	services := mergeDockerStatusAndComposes(containers, composes)
	return services, nil
}

func getComposePlan(c *gin.Context) {
	composes, err := listComposes()
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, composes)
}

func getDockerStatus(c *gin.Context) {
	containers, err := dockerStatus()
	if err != nil {
		c.JSON(500, err.Error())
		return
	}

	c.JSON(200, containers)
}

// ---------

var dockerClient *client.Client

func dockerStatus() ([]types.Container, error) {
	if dockerClient == nil {
		c, err := client.NewClient("unix:///var/run/docker.sock", "v1.22", nil, defaultHeaders)
		if err != nil {
			return nil, err
		}
		dockerClient = c
	}

	options := types.ContainerListOptions{All: true}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	return containers, nil
}

// ---------

type RawCompose struct {
	Services RawServices `json:"services"`
}

type RawServices map[string]map[string]interface{}

var composesDir = "./compose"

func listComposeFiles() ([]string, error) {
	composeFiles := []string{}

	err := filepath.Walk(composesDir, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yml") {
			composeFiles = append(composeFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return composeFiles, err
}

func listComposes() ([]RawCompose, error) {
	composes := []RawCompose{}

	composeFiles, err := listComposeFiles()
	if err != nil {
		return nil, err
	}

	for _, composeFile := range composeFiles {
		compose, err := yaml2json(composeFile)
		if err != nil {
			return nil, err
		}
		composes = append(composes, *compose)
	}

	return composes, nil
}

func yaml2json(file string) (*RawCompose, error) {
	in, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var compose RawCompose
	err = yaml.Unmarshal(in, &compose)
	if err != nil {
		return nil, err
	}

	return &compose, nil
}

type Service struct {
	Image      string      `json:"image"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	FullStatus string      `json:"fullStatus"`
	Definition interface{} `json:"definition"`
}

// A services slice is sortable

type Services []Service

func (s Services) Len() int {
	return len(s)
}

func (s Services) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Services) Less(i, j int) bool {
	return s[i].Status < s[j].Status
}

// status: Up, Ex

func mergeDockerStatusAndComposes(containers []types.Container, composes []RawCompose) Services {
	services := Services{}

	// Transform all containers listed in docker ps -a in services
	for _, container := range containers {
		services = append(services, Service{
			Image:      container.Image,
			Name:       strings.Replace(container.Names[0], "/", "", -1),
			FullStatus: container.Status,
			Status:     "_NotDeclared",
			Definition: []string{},
		})
	}

	// Update service with its compose file declaration

	missingServices := []Service{}

	for _, compose := range composes {
		for name, composeService := range compose.Services {
			// Name is the container_name if defined or the key of the service
			containerName := composeService["container_name"]
			if containerName != nil {
				name = containerName.(string)
			}
			image := composeService["image"].(string)

			isInDockerPs := false
			for i, s := range services {
				// The service match the compose declaration if image and name matches
				if s.Image == image && (s.Name == name || strings.Contains(s.Name, "_"+name+"_")) {
					isInDockerPs = true
					// Keep the first word of the full status as status
					services[i].Status = strings.Split(s.FullStatus, " ")[0]
					services[i].Definition = composeService
				}
			}

			// Handle containers not started
			if !isInDockerPs {
				missingServices = append(missingServices, Service{
					Image:      image,
					Name:       name,
					FullStatus: "Not started",
					Status:     "NotStarted",
					Definition: composeService,
				})
			}
		}
	}

	services = append(services, missingServices...)

	sort.Sort(services)

	return services
}

func handleError(c *gin.Context, err error) {
	c.JSON(500, err.Error())
}
