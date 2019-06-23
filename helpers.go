package main

import (
	"fmt"
	"time"
	"log"
)

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const layout = "2006-01-02"

func getTimestamp(detailed bool) string {
	t := time.Now()
	if detailed {
		return fmt.Sprintf("%s-%d", t.Format(layout), t.Unix())
	} else {
		return fmt.Sprintf("%s", t.Format(layout))
	}
}

func hasKey(arguments map[string]interface{}, key string) bool {
	_, exists := arguments[key]
	return exists && (arguments[key] != nil)
}

func getArgs(arguments map[string]interface{}) []string {
	if !hasKey(arguments, "<args>") {
		log.Fatal("Please provide args.")
	}
	return arguments["<args>"].([]string)
}
