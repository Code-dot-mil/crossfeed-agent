package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/joho/sqltocsv"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Domain struct {
	name  string
	ip    string
	ports string
}

func scanPorts(args []string) {
	log.SetPrefix("[portscan] ")
	if len(args) < 1 {
		log.Fatal("Please provide the project sonar url")
	}
	initPortScan(strings.Split(args[0], ","))
}

func initPortScan(urls []string) {
	db, err := sql.Open("postgres", psqlInfo)
	handleError(err)
	defer db.Close()
	err = db.Ping()
	handleError(err)

	log.Println("Exporting hosts to file...")

	// Fetch all ip addresses from db and sort
	query := `SELECT ip FROM "Domains" WHERE ip IS NOT NULL AND ip <> '';`
	rows, err := db.Query(query)
	handleError(err)
	csvConverter := sqltocsv.New(rows)
	csvConverter.WriteHeaders = false

	var hostsPath string = "output/portscan/ips-" + getTimestamp(false) + ".txt"
	err = csvConverter.WriteFile(hostsPath)
	handleError(err)
	log.Println("Sorting...")
	_, err = exec.Command("sort", "-o", hostsPath, hostsPath).Output()
	handleError(err)

	for _, url := range urls {
		split := strings.Split(url, "_")
		port := split[len(split) - 1]

		// Download Sonar data for url
		log.Println("Starting port scan for port " + port + " using " + url)
		_, err := exec.Command("/bin/sh", "prepare_files.sh", url, port).Output()
		handleError(err)

		var sonarPath string = "output/portscan/sonar/" + port + ".txt"
		var outpath string = "output/portscan/" + port + "-" + getTimestamp(false) + ".txt"
		cmd := exec.Command("comm", "-12", sonarPath, hostsPath)
		out, err := os.Create(outpath)
		handleError(err)
		defer out.Close()
		cmd.Stdout = out

		err = cmd.Start()
		handleError(err)
		cmd.Wait()
		log.Println("Files successfully compared! See " + outpath)

		file, err := os.Open(outpath)
		handleError(err)

		scanner := bufio.NewScanner(file)
		var ipsArray []string
		var portsArray []string
		for scanner.Scan() {
			ipsArray = append(ipsArray, fmt.Sprintf("'%s'", scanner.Text()))
			portsArray = append(portsArray, fmt.Sprintf("'%s'", port))
		}
		file.Close()

		log.Println(fmt.Sprintf("Uploading %d found open ports to db...", len(ipsArray)))

		// Please excuse this horrific UPSERT query for the time being, there's no easy way to do it in go.
		query = `UPDATE "Domains" SET ports = CASE WHEN "Domains".ports IS NOT NULL AND "Domains".ports <> '' THEN "Domains".ports || ',' || data_table.ports ELSE data_table.ports END
					FROM (SELECT unnest(array[` + strings.Join(ipsArray[:], ",") + `]) as ip, unnest(array[` + strings.Join(portsArray[:], ",") + `]) as ports)
					as data_table where "Domains".ip = data_table.ip AND strpos("Domains".ports, data_table.ports) = 0;`

		_, err = db.Exec(query)
		handleError(err)

		err = os.Remove(sonarPath)
		handleError(err)

		err = os.Remove(outpath)
		handleError(err)

		log.Println("Done scanning ports for port " + port + " using " + url)
	}

	log.Println("Finished port scan for " + strings.Join(urls,","))

	err = os.Remove(hostsPath)
	handleError(err)

}
