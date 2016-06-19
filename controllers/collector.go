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

func GetAgent(c *gin.Context) {
	getAgentScript := `docker pull krkr/squid
docker rm -f squid-agent 2> /dev/null || true
docker run -d \
  --name squid-agent \
  --hostname=$(hostname) \
  -p 4242 \
  -v $(pwd)/compose:/app/compose \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e DOMAIN=${DOMAIN} -e ONS_ZONE=${ONS_ZONE} \
  --restart=always \
  krkr/squid -c http://localhost:4242
`

	c.String(200, getAgentScript)
}
