package main

import (
	"database/sql"
	"log"
	"fmt"
	"os"
	"os/exec"
	"bufio"
	"github.com/joho/sqltocsv"
)

func subjack(args []string) {
	log.SetPrefix("[subjack] ")
	initSubjack(args)
}

func initSubjack(args []string) {
	db, err := sql.Open("postgres", psqlInfo)
	handleError(err)
	defer db.Close()
	err = db.Ping()
	handleError(err)

	// Fetch all hosts, for now restrict to live hosts (80/443)
	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM "Domains" WHERE ports LIKE '%80%' OR ports LIKE '%443%';`)
	err = row.Scan(&count)
	handleError(err)

	query := `SELECT name FROM "Domains" WHERE ports LIKE '%80%' OR ports LIKE '%443%';`
	rows, err := db.Query(query)
	handleError(err)
	csvConverter := sqltocsv.New(rows)
	csvConverter.WriteHeaders = false

	var hostsPath string = "output/subjack/hosts.txt"
	err = csvConverter.WriteFile(hostsPath)
	handleError(err)

	log.Println(fmt.Sprintf("Beginning subjack scan on %d domains.", count))

	var resultsPath string = "output/subjack/results.txt"
	file, err := os.Create(resultsPath)
	handleError(err)
	file.Close()

	_, err = exec.Command("subjack", "-w", hostsPath, "-o", resultsPath, "-ssl", "-a").Output()
	handleError(err)

	file, err = os.Open(resultsPath)
	handleError(err)

	scanner := bufio.NewScanner(file)
	// var hostsArray []string
	found := false
	for scanner.Scan() {
		found = true
		log.Println("Found vulnerable subdomain: " + scanner.Text())
	}
	file.Close()

	if !found {
		log.Println("No vulnerable subdomains were found.")
	}

	err = os.Remove(resultsPath)
	handleError(err)

}
