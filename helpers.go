package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"github.com/joho/sqltocsv"
	"net/http"
	"time"
)

// Helper method to log fatally if an error occurs
func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Returns the current year-month pair, e.g. 2006-01
func getMonth() string {
	t := time.Now()
	return fmt.Sprintf("%s", t.Format("2006-01"))
}

const layout = "2006-01-02"

// Returns the current yyyy-mm-dd time, optionally with a unix timestamp appended
func getTimestamp(detailed bool) string {
	t := time.Now()
	if detailed {
		return fmt.Sprintf("%s-%d", t.Format(layout), t.Unix())
	} else {
		return fmt.Sprintf("%s", t.Format(layout))
	}
}

// Helper method to check if a map has a given key
func hasKey(arguments map[string]interface{}, key string) bool {
	_, exists := arguments[key]
	return exists && (arguments[key] != nil)
}

// Parses arguments from docopt's args
func getArgs(arguments map[string]interface{}) []string {
	if !hasKey(arguments, "<args>") {
		log.Fatal("Please provide args.")
	}
	return arguments["<args>"].([]string)
}

// Helper method to check if a slice contains a given string
func sliceContains(slice []string, str string) bool {
	for i := range slice {
		if slice[i] == str {
			return true
		}
	}
	return false
}

// Performs an http request to the given url with specified headers
// Results are returned in result, which should match the JSON schema
func fetchExternalAPI(url string, headers map[string]string, result interface{}) {
	client := http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	handleError(err)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res, err := client.Do(req)
	handleError(err)

	body, err := ioutil.ReadAll(res.Body)
	handleError(err)

	err = json.Unmarshal(body, &result)
	handleError(err)
}

func writeQueryToFile(query string, file string) {
	rows, err := db.Query(query)
	handleError(err)
	csvConverter := sqltocsv.New(rows)
	csvConverter.WriteHeaders = false

	err = csvConverter.WriteFile(file)
	handleError(err)
}

// Begin tracking the task's status
func initStatusTracker(command string) int {
	var id int
	query := `INSERT INTO "TaskStatuses" (command, status, percentage) VALUES ($1, 'running', 0) RETURNING id`
	err := db.QueryRow(query, command).Scan(&id)
	handleError(err)
	return id
}

var lastPercentages = make(map[string]int)

// Updates the database with the current task percentage
// Will only update if the percentage has changed
func updateTaskPercentage(id string, percentage int) {
	if lastPercentages[id] == percentage {
		return
	}
	lastPercentages[id] = percentage
	query := `UPDATE "TaskStatuses" SET percentage = $1 WHERE id = $2`
	_, err := db.Exec(query, percentage, id)
	handleError(err)
}

// Updates the status of the task in the database
func updateTaskStatus(id int, status string) {
	query := `UPDATE "TaskStatuses" SET status = $1 WHERE id = $2`
	_, err := db.Exec(query, status, id)
	handleError(err)
}

// Updates alerts database and sends Slack notification
func updateTaskOutput(command string, text string, priority int) {
	query := `INSERT INTO "Alerts" (command, text, priority) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, command)
	handleError(err)
}
