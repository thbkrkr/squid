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
	containers, err := dockerStatus()
	if err != nil {
		handleError(c, err)
		return
	}

	composes, err := listComposes()
	if err != nil {
		handleError(c, err)
		return
	}

	services := mergeDockerStatusAndCompose(containers, composes)
	c.JSON(200, services)
}

func GetFullStatus() ([]Service, error) {
	containers, err := dockerStatus()
	if err != nil {
		return nil, err
	}

	composes, err := listComposes()
	if err != nil {
		return nil, err
	}

	services := mergeDockerStatusAndCompose(containers, composes)
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

type services map[string]map[string]interface{}

type Compose struct {
	Services services `json:"services"`
}

var composesDir = "./compose"

func listComposeFiles() ([]string, error) {
	composeFiles := []string{}

	err := filepath.Walk(composesDir, func(path string, f os.FileInfo, err error) error {
		if strings.Contains(path, ".yml") {
			composeFiles = append(composeFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return composeFiles, err
}

func listComposes() ([]Compose, error) {
	composes := []Compose{}

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

func yaml2json(file string) (*Compose, error) {
	in, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var compose Compose
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

func mergeDockerStatusAndCompose(containers []types.Container, composes []Compose) Services {
	services := Services{}

	// Add all containers listed in docker ps -a
	for _, container := range containers {
		services = append(services, Service{
			Image:      container.Image,
			Name:       strings.Replace(container.Names[0], "/", "", -1),
			FullStatus: container.Status,
			Status:     "Z",
			Definition: []string{},
		})
	}

	// Update containers that are declared in a docker compose file

	missingServices := []Service{}

	for _, compose := range composes {
		for name, composeService := range compose.Services {

			containerName := composeService["container_name"]
			if containerName != nil {
				name = containerName.(string)
			}
			image := composeService["image"].(string)

			isInDockerPs := false
			for i, s := range services {
				if s.Image == image && (s.Name == name || strings.Contains(s.Name, "_"+name+"_")) {
					isInDockerPs = true
					// Keep the first word of the full status
					services[i].Status = strings.Split(s.FullStatus, " ")[0]
					services[i].Definition = composeService
				}
			}

			if !isInDockerPs {
				missingServices = append(missingServices, Service{
					Image:      image,
					Name:       name,
					FullStatus: "Absent",
					Status:     "Absent",
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
