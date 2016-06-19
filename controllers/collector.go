package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

var (
	statuses = map[string][]Service{}
	m        sync.RWMutex
)

func CollectStatus(c *gin.Context) {
	m.Lock()
	defer m.Unlock()

	host := c.Param("host")

	var servicesForm []Service

	if err := c.BindJSON(&servicesForm); err != nil {
		handleError(c, err)
		return
	}

	statuses[host] = servicesForm

	c.JSON(200, true)
}

func Statuses(c *gin.Context) {
	m.RLock()
	defer m.RUnlock()

	c.JSON(200, statuses)
}

func GetAgent(c *gin.Context) {
	url := "https://squid.blurb.space"
	getAgentScript := `echo "Install squid...."
docker pull krkr/squid
docker rm -f squid-agent 2> /dev/null || true
docker run -d \
  --name squid-agent \
  --hostname=$(hostname) \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/compose:/app/compose \
  -p 4242:4242 \
  --restart=always \
  krkr/squid -join ` + url

	c.String(200, getAgentScript)
}

func SendServicesStatus(collector string, username string, password string, period int, host string) {
	duration := time.Duration(period) * time.Second

	for {
		services, err := GetFullStatus()
		if err != nil {
			logrus.WithError(err).Error("Fail to get status")
		}

		err = postStatus(collector, username, password, host, services)
		if err != nil {
			logrus.WithError(err).Error("Fail to send status")
		}

		time.Sleep(duration)
	}
}

func postStatus(collector string, username string, password string, host string, services []Service) error {
	json, err := json.Marshal(services)
	if err != nil {
		return err
	}

	url := collector + "/api/nodes/status/" + host

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}

	return nil
}
