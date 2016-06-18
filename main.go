package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/thbkrkr/squid/controllers"
)

var (
	collector = flag.String("c", "http://localhost:4242", "Collector URL")
	host      = flag.String("host", "", "Hostname")
)

func main() {
	flag.Parse()

	if *host == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "default"
		}
		*host = hostname
	}

	go postStatus()

	api("squid", func(r *gin.Engine) {
		r.POST("/nodes/status/:host", controllers.CollectStatus)
		r.GET("/nodes/status", controllers.Statuses)

		r.GET("/compose/status", controllers.GetStatus)
		r.GET("/compose/up", controllers.GetComposeUp)
		r.GET("/compose/plan", controllers.GetComposePlan)

		r.GET("/executions", controllers.GetExecutions)

		r.GET("/docker/status", controllers.GetDockerStatus)
	})
}

func api(name string, f func(r *gin.Engine)) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	r.Static("/s", "./views/")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/s")
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":   name,
			"ok":     "true",
			"status": 200,
		})
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.JSON(200, nil)
	})

	f(r)

	logrus.WithField("port", 4242).WithField("name", name).Info("Start")
	r.Run(":4242")
}

func postStatus() {
	duration := time.Duration(10) * time.Second

	for {
		services, err := controllers.GetFullStatus()
		if err != nil {
			logrus.WithError(err).Error("Fail to get status")
		}

		err = send(services)
		if err != nil {
			logrus.WithError(err).Error("Fail to send status")
		}

		time.Sleep(duration)
	}
}

func send(services []controllers.Service) error {
	json, err := json.Marshal(services)
	if err != nil {
		return err
	}

	url := *collector + "/nodes/status/" + *host

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	//req.Header.Set("Content-Type", "")
	/*if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}*/

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	//fmt.Printf("status=%v\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}

	return nil
}
