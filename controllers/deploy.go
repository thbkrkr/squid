package controllers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	executions = []*cmdResult{}
	mE         sync.RWMutex
)

type cmdResult struct {
	Date   int64                  `json:"date"`
	Cmd    map[string]interface{} `json:"cmd"`
	Result []string               `json:"result"`
}

func GetComposeUp(c *gin.Context) {
	now := time.Now().UnixNano()

	composeFiles, err := listComposeFiles()
	if err != nil {
		handleError(c, err)
		return
	}

	nbComposes := len(composeFiles)
	results := make([]*cmdResult, nbComposes)

	var wg sync.WaitGroup
	wg.Add(nbComposes)

	for index, composeFile := range composeFiles {
		go func(i int, compose string) {
			defer wg.Done()

			// Exec docker-compose up using doo
			fmt.Printf("doo -q dc %s up -d\n", compose)
			cmd := exec.Command("doo", "-q", "dc", compose, "up", "-d")

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

			results[i] = &cmdResult{
				Date:   now,
				Cmd:    data,
				Result: lines[:len(lines)-2],
			}
		}(index, composeFile)
	}

	wg.Wait()

	m.Lock()
	defer m.Unlock()
	executions = results

	c.JSON(200, results)
}

func GetExecutions(c *gin.Context) {
	m.RLock()
	defer m.RUnlock()

	c.JSON(200, executions)
}