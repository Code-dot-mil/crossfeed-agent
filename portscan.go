package main

import (
	"database/sql"
	"fmt"
	"log"
	"github.com/joho/sqltocsv"
	"os/exec"
	"os"
	"strings"
	"bufio"
)

type Domain struct {
    name string
    ip  string
    ports string
}

func scanPorts(arguments map[string]interface{}) {
	if !hasKey(arguments, "<subcommand>") {
		log.Fatal("Please provide a subcommand. (either exportHosts or scan)")
	}
	switch arguments["<subcommand>"].(string) {
	case "scan":
		if !hasKey(arguments, "--ports") {
			log.Fatal("Please indicate the ports")
		}
		initPortScan(arguments["--ports"].(string))
	default:
		fmt.Println("Command not found: scanPorts" + arguments["<subcommand>"].(string))
	}
}


func initPortScan(ports string) {
	var path string = "output/portscan/ips-" + getTimestamp(false) + ".txt"
	var port string = ports

	db, err := sql.Open("postgres", psqlInfo)
	handleError(err)
	defer db.Close()
	err = db.Ping()
	handleError(err)

	fmt.Println("Exporting hosts to file...")

	query := "SELECT ip FROM \"Domains\" WHERE ip IS NOT NULL AND ip <> '';"
	rows, err := db.Query(query)
	handleError(err)
	csvConverter := sqltocsv.New(rows)
	csvConverter.WriteHeaders = false
	err = csvConverter.WriteFile(path)
	handleError(err)
	fmt.Println("Sorting...")
	_, err = exec.Command("sort", "-o", path, path).Output()
	handleError(err)

	var sonar string = "output/portscan/sonar/sonar-" + port + ".txt"
	var outpath string = "output/portscan/" + port + "-" + getTimestamp(false) + ".txt"
	cmd := exec.Command("comm", "-12", sonar, path)
	out, err := os.Create(outpath)
    handleError(err)
    defer out.Close()
    cmd.Stdout = out

    err = cmd.Start();
    handleError(err)
    cmd.Wait()
	fmt.Println("Successfully exported! See " + outpath)

	file, err := os.Open(outpath)
    handleError(err)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var ipsArray []string
    var portsArray []string
    for scanner.Scan() {
    	ipsArray = append(ipsArray, fmt.Sprintf("'%s'", scanner.Text()))
    	portsArray = append(portsArray, fmt.Sprintf("'%s'", ports))
    }

	fmt.Println("Uploading to db...")

	// Please excuse this horrific UPSERT query for the time being, there's no easy way to do it in go.
	query = "UPDATE \"Domains\" SET ports = \"Domains\".ports || ',' || data_table.ports FROM (SELECT unnest(array[" + strings.Join(ipsArray[:], ",") + "]) as ip, unnest(array[" + strings.Join(portsArray[:], ",") + "]) as ports) as data_table where \"Domains\".ip = data_table.ip AND strpos(\"Domains\".ports, data_table.ports) = 0;"

	_, err = db.Exec(query)
	handleError(err)

	fmt.Println("Done!")

}

