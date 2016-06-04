package controllers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

var (
	defaultHeaders = map[string]string{"User-Agent": "squid-api-1.0"}
)

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

func GetComposePlan(c *gin.Context) {
	composes, err := listComposeFiles()
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, composes)
}

type cmdResult struct {
	Cmd    map[string]interface{} `json:"cmd"`
	Result []string               `json:"result"`
}

func GetComposeUp(c *gin.Context) {
	results := []cmdResult{}

	composeFiles, err := listComposeFiles()
	if err != nil {
		handleError(c, err)
		return
	}

	for _, composeFile := range composeFiles {
		// Exec docker-compose up using doo
		cmd := exec.Command("doo", "dc", composeFile, "up", "-d")

		stdout, err := cmd.CombinedOutput()
		if err != nil {
			handleError(c, err)
			return
		}

		lines := strings.Split(string(stdout), "\n")
		// Forget the empty last line and
		// unmarshal the penultimate line in json
		in := lines[len(lines)-2]
		var data map[string]interface{}
		err = json.Unmarshal([]byte(in), &data)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}

		results = append(results, cmdResult{
			Cmd:    data,
			Result: lines[:len(lines)-2],
		})
	}

	c.JSON(200, results)
}

func GetDockerStatus(c *gin.Context) {
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

type compose struct {
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

func listComposes() ([]compose, error) {
	composes := []compose{}

	composeFiles, err := listComposeFiles()
	if err != nil {
		return nil, err
	}

	for _, composeFile := range composeFiles {
		compose, err := yaml2json(composeFile)
		if err != nil {
			return nil, err
		}
		composes = append(composes, compose)
	}

	return composes, nil
}

func yaml2json(file string) (compose, error) {
	in, err := ioutil.ReadFile(file)
	if err != nil {
		return compose{}, err
	}

	var obj compose
	err = yaml.Unmarshal(in, &obj)
	if err != nil {
		return compose{}, err
	}

	return obj, nil
}

type service struct {
	Image      string      `json:"image"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	FullStatus string      `json:"fullStatus"`
	Definition interface{} `json:"definition"`
}

func mergeDockerStatusAndCompose(containers []types.Container, composes []compose) []service {
	services := []service{}

	// Add all containers listed in docker ps -a
	for _, container := range containers {
		services = append(services, service{
			Image:      container.Image,
			Name:       strings.Replace(container.Names[0], "/", "", -1),
			FullStatus: container.Status,
			Status:     "unknown",
			Definition: []string{},
		})
	}

	// Update containers that are declared in a docker compose file

	missingServices := []service{}

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
					services[i].Status = strings.Split(s.FullStatus, " ")[0]
					services[i].Definition = composeService
				}
			}

			if !isInDockerPs {
				missingServices = append(missingServices, service{
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

	return services
}

func handleError(c *gin.Context, err error) {
	c.JSON(500, err.Error())
}
