package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/thbkrkr/squid/controllers"
)

var (
	host = flag.String("host", "", "Hostname")

	collector = flag.String("c", "", "Squid server URL")
	period    = flag.Int("p", 10, "Interval to report health in seconds")

	token = flag.String("Token", "ba.zinga", "Token")

	buildDate = "dev"
	gitCommit = "dev"
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

	tokens := strings.Split(*token, ".")
	username := tokens[0]
	password := tokens[1]

	if *collector != "" {
		go sendStatus(username, password)
	}

	api("squid", username, password,
		func(r *gin.Engine) {
			r.GET("/get", controllers.GetAgent)
		}, func(r *gin.RouterGroup) {
			r.POST("/nodes/status/:host", controllers.CollectStatus)
			r.GET("/nodes/status", controllers.Statuses)
			r.GET("/compose/status", controllers.GetStatus)
			r.GET("/compose/up", controllers.GetComposeUp)
			r.GET("/executions", controllers.GetExecutions)
		})
}

func api(name string, username string, password string, f func(r *gin.Engine), g func(r *gin.RouterGroup)) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	r.Static("/s", "./views/")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/s")
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":      name,
			"buildDate": buildDate,
			"gitCommit": gitCommit,
			"ok":        "true",
			"status":    200,
		})
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.JSON(200, nil)
	})

	f(r)

	a := r.Group("/api", gin.BasicAuth(gin.Accounts{
		username: password,
	}))

	g(a)

	logrus.WithFields(logrus.Fields{
		"name":      name,
		"port":      4242,
		"buildDate": buildDate,
		"gitCommit": gitCommit,
	}).Info("Start")

	r.Run(":4242")
}

func sendStatus(username string, password string) {
	duration := time.Duration(*period) * time.Second

	for {
		services, err := controllers.GetFullStatus()
		if err != nil {
			logrus.WithError(err).Error("Fail to get status")
		}

		err = send(username, password, services)
		if err != nil {
			logrus.WithError(err).Error("Fail to send status")
		}

		time.Sleep(duration)
	}
}

func send(username string, password string, services []controllers.Service) error {
	json, err := json.Marshal(services)
	if err != nil {
		return err
	}

	url := *collector + "/api/nodes/status/" + *host

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
