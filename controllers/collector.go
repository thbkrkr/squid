package controllers

import (
	"sync"

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
