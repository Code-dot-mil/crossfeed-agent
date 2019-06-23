package main

import (
	"fmt"
	"time"
)

func handleError(err error) {
	if err != nil {
		panic(err)
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