package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/thbkrkr/squid/controllers"
)

var (
	creds = flag.String("creds", "ba:zinga", "Basic auth credentials (username:password)")

	collector = flag.String("join", "https://squid.blurb.space", "Squid server URL")
	period    = flag.Int("p", 10, "Interval to report status in seconds")

	host = flag.String("h", "", "Hostname")

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

	credsParts := strings.Split(*creds, ":")
	username := credsParts[0]
	password := credsParts[1]

	if *collector != "" {
		go controllers.SendServicesStatus(*collector, username, password, *period, *host)
	}

	api("squid", username, password,
		func(r *gin.Engine) {
			r.GET("/get", controllers.GetAgent)
		}, func(r *gin.RouterGroup) {
			r.POST("/nodes/status/:host", controllers.CollectStatus)
			r.GET("/nodes/status", controllers.Statuses)
			r.GET("/compose/status", controllers.GetStatus)
			r.GET("/compose/up", controllers.ComposeUp)
			r.GET("/executions", controllers.ComposeUpHistory)
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
			"buildDate": buildDate,
			"gitCommit": gitCommit,
			"name":      name,
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
		"buildDate": buildDate,
		"gitCommit": gitCommit,
		"name":      name,
		"port":      4242,
	}).Info("Start")

	r.Run(":4242")
}
