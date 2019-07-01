package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"os/exec"
	"fmt"
	"strings"
)

func fetchHosts(args []string) {
	log.SetPrefix("[hostscanner] ")
	if len(args) < 1 {
		log.Fatal("Please provide the config file or a single path.")
	}
	initHostScan(args[0])
}

func initHostScan(input string) {
	var paths []string
	if strings.HasPrefix(input, "/") { // Treat as single path
		paths = append(paths, input)
	} else { // Find input config file

	}

	db, err := sql.Open("postgres", psqlInfo)
	handleError(err)
	defer db.Close()
	err = db.Ping()
	handleError(err)

	query := `SELECT name, ports FROM "Domains" WHERE ports LIKE '%80%' OR ports LIKE '%443%';`
	rows, err := db.Query(query)
	handleError(err)
	defer rows.Close()

	hostsPath := "output/hostscanner/hosts.txt"
	pathsPath := "output/hostscanner/paths.txt"
	outPath   := "output/hostscanner/megoutput/"
	f, err := os.Create(hostsPath)
    handleError(err)

	var name string
	var ports string
	var count int
	for rows.Next() {
		err := rows.Scan(&name, &ports)
		handleError(err)
		protocol := "http://"
		if strings.Contains(ports, "443") {
			protocol = "https://"
		}
		_, err = f.WriteString(fmt.Sprintf("%s%s\n", protocol, name))
		handleError(err)
		count++
	}

	f.Close()

	f, err = os.Create(pathsPath)
    handleError(err)

	for _, path := range paths {
		_, err = f.WriteString(path + "\n")
		handleError(err)
	}

	f.Close()


	// start the command after having set up the pipe

	log.Println(fmt.Sprintf("Beginning host scan for %d paths on %d domains", len(paths), count))

	cmd := exec.Command("meg", "-c", "50", "-v", pathsPath, hostsPath, outPath)
	stdout, err := cmd.StdoutPipe()
	handleError(err)
	stderr, err := cmd.StderrPipe()
	handleError(err)
	err = cmd.Start()
	handleError(err)

	outIn := bufio.NewScanner(stdout)
	errIn := bufio.NewScanner(stderr)
	cur := 0
	for {
		cont := false
		if errIn.Scan() {
			cur++
			log.Println(fmt.Sprintf("(%d/%d) %s", cur, count, errIn.Text()))
			cont = true
		}
		if outIn.Scan() {
	    	cur++
			log.Println(fmt.Sprintf("(%d/%d) %s", cur, count, outIn.Text()))
	    	cont = true
		}
		if !cont {
			break
		}
	}

	err = outIn.Err()
	handleError(err)


	log.Println(fmt.Sprintf("Finished host scan for %d paths on %d domains", len(paths), count))

}
