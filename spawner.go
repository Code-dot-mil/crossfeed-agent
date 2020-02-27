package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Job struct {
	Command string
	Payload string
}

// Background process to consume and initiate jobs from queue
func initSpawner(arguments []string) {
	currentDir, err := os.Getwd()
	handleError(err)

	log.SetPrefix("[spawner] ")

	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.AWS_REGION),
		Credentials: credentials.NewStaticCredentials(config.AWS_ACCESS_KEY_ID, config.AWS_SECRET_ACCESS_KEY, ""),
		Endpoint:    aws.String(config.SQS_ENDPOINT),
	})

	svc := sqs.New(awsSession)

	log.Println("Spawner initiated. Waiting for next job.")
	for {
		// Receive job from queue
		result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			AttributeNames: []*string{
				aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
			},
			MessageAttributeNames: []*string{
				aws.String(sqs.QueueAttributeNameAll),
			},
			QueueUrl:            &config.SQS_URL,
			MaxNumberOfMessages: aws.Int64(1),
			VisibilityTimeout:   aws.Int64(20), // 20 seconds
			WaitTimeSeconds:     aws.Int64(20),
		})

		if err != nil {
			fmt.Println("Error reserving job: " + err.Error())
			continue
		} else if len(result.Messages) == 0 {
			continue
		}

		message := result.Messages[0]
		id := *message.MessageId

		// Delete job from queue
		_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      &config.SQS_URL,
			ReceiptHandle: message.ReceiptHandle,
		})

		if err != nil {
			log.Println(fmt.Sprintf("Error deleting job: %d, %s", id, err.Error()))
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(*message.Body), &job); err != nil {
			log.Println("Error parsing json: " + err.Error())
			continue
		}

		command := job.Command

		log.Println(fmt.Sprintf("Spawning job: %s with command %s", id, command))

		taskID := initStatusTracker(command)

		args := strings.Split(command, " ")
		cmd := args[0]
		allowedCommands := []string{"scan-ports", "scan-hosts", "subjack", "bitdiscovery", "store-results"}
		if !sliceContains(allowedCommands, cmd) {
			log.Println("Command not found: " + cmd)
			continue
		}

		if job.Payload != "" {
			args = append(args, job.Payload)
		}

		for i := range args[1:] {
			args[i+1] = shellescape.Quote(args[i+1])
		}

		args = append(args, fmt.Sprintf("%d", taskID)) // the taskID is always the last argument

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.SPAWNER_TIMEOUT_LENGTH)*time.Minute)
		defer cancel()

		_, err = exec.CommandContext(ctx, currentDir+"/crossfeed-agent", args...).Output()

		if ctx.Err() == context.DeadlineExceeded { // If command timed out, still process results
			log.Println(fmt.Sprintf("Command %s timed out after %d minutes, continuing.", command, config.SPAWNER_TIMEOUT_LENGTH))
		}

		if err != nil {
			log.Println("Executing job failed: " + err.Error())
			updateTaskStatus(taskID, "failed")
		} else {
			updateTaskStatus(taskID, "finished")
		}

		log.Println(fmt.Sprintf("Finished job: %s", id))
		log.Println("Waiting for next job.")
	}
}
