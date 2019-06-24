package main

import (
	"database/sql"
	"log"
	"strings"
)

func fetchHosts(args []string) {
	log.SetPrefix("[hostscanner] ")
	// if len(args) < 1 {
	// 	log.Fatal("Please provide the project sonar url")
	// }
	initHostScan(args)
}

func initHostScan(args []string) {
	db, err := sql.Open("postgres", psqlInfo)
	handleError(err)
	defer db.Close()
	err = db.Ping()
	handleError(err)

	log.Println("Beginning host scan with args " + strings.Join(args, ", "))

}
