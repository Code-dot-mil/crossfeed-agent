package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
	"strings"
	"encoding/json"

	"github.com/beanstalkd/go-beanstalk"
	"github.com/alessio/shellescape"
)

// Background process to consume and initiate jobs from queue
func initSpawner(arguments []string) {

	currentDir, err := os.Getwd()
	handleError(err)

	log.SetPrefix("[spawner] ")
	client, err := beanstalk.Dial("tcp", config.BeanstalkHost)
	handleError(err)
	log.Println("Spawner initiated. Waiting for next job.")
	for {
		id, body, err := client.Reserve(time.Duration(config.BeanstalkPollRate) * time.Second)
		if err != nil {
			if !strings.Contains(err.Error(), "timeout") { // Don't print if it's a timeout
				log.Println("Error reserving job: " + err.Error())
			}
			continue
		}

		err = client.Delete(id)
		if err != nil {
			log.Println(fmt.Sprintf("Error deleting job: %d, %s", id, err.Error()))
			continue
		}

		command := string(body)
		if strings.HasPrefix(command, "{") { // is json
			var dat map[string]interface{}
		    if err := json.Unmarshal(body, &dat); err != nil {
		        log.Println("Error parsing json: " + err.Error())
		        continue
		    }
		    cmd, exists := dat["payload"]
		    if !exists {
		        log.Println("Invalid input provided: " + string(body))
		        continue
		    }
		    command = cmd.(string)
		}

		log.Println(fmt.Sprintf("Spawning job: %d with command %s",  id, command))

		args := strings.Split(command, " ")
		cmd := args[0]
		allowedCommands := []string {"scanPorts", "fetchHosts", "subjack"}
		if !sliceContains(allowedCommands, cmd) {
			log.Println("Could not parse command: " + cmd)
			continue
		}
		for i := range args[1:] {
			args[i+1] = shellescape.Quote(args[i+1])
		}

		_, err = exec.Command(currentDir + "/crossfeed-agent", args...).Output()
		if err != nil {
			log.Println("Executing job failed: " + err.Error())
		}

		log.Println(fmt.Sprintf("Finished job: %d", id))
		log.Println("Waiting for next job.")
	}
}

// Enqueues a job on the job queue
func enqueueJob(args []string) {
	log.SetPrefix("[enqueue] ")
	client, err := beanstalk.Dial("tcp", config.BeanstalkHost)
	handleError(err)
	command := strings.Join(args, " ")
	var priority uint32 = 1
	delay := time.Duration(0)
	ttr := 60 * time.Minute
	id, err := client.Put([]byte(command), priority, delay, ttr)
	handleError(err)
	log.Println(fmt.Sprintf("Successfully enqueued command %s with job id %d.", command, id))
}
