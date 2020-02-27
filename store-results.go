package main

import (
	"log"
)

func storeResults(args []string) {
	log.SetPrefix("[store-results] ")
	if len(args) < 1 {
		log.Fatal("Please provide the key of results to store")
	}
}
