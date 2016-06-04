package main

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/thbkrkr/squid/controllers"
)

func main() {
	api("squid", func(r *gin.Engine) {
		r.GET("/compose/status", controllers.GetStatus)
		r.GET("/compose/up", controllers.GetComposeUp)
		r.GET("/compose/plan", controllers.GetComposePlan)
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
