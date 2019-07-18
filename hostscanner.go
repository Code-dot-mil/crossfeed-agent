package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"os/exec"
	"fmt"
	"strings"

	"crossfeed-agent/webanalyze"
)

const(
	hostsPath = "output/hostscanner/hosts.txt"
	pathsPath = "output/hostscanner/paths.txt"
	outPath   = "output/hostscanner/megoutput/"
	wappalyzeAppsPath = "output/hostscanner/apps.json"
)

func fetchHosts(args []string) {
	log.SetPrefix("[hostscanner] ")
	if len(args) < 1 {
		log.Fatal("Please provide the config file or a single path.")
	}
	switch args[0] {
	case "initWappalyzer":
		initWappalyzer()
	case "scan":
		initHostScan(args[1])
	case "parseResults":
		parseScanResults()
	default:
		fmt.Println("Subcommand not found: " + args[0])
	}
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
	// handleError(err)
	// stderr, err := cmd.StderrPipe()
	// handleError(err)
	err = cmd.Start()
	handleError(err)

	cur := 0
	in := bufio.NewScanner(stdout)
	for in.Scan() {
		cur++
		log.Println(fmt.Sprintf("(%d/%d) %s", cur, count, in.Text()))
	}

	// outIn := bufio.NewScanner(stdout)
	// errIn := bufio.NewScanner(stderr)
	// cur := 0
	// for {
	// 	cont := false
	// 	if errIn.Scan() {
	// 		cur++
	// 		log.Println(fmt.Sprintf("(%d/%d) %s", cur, count, errIn.Text()))
	// 		cont = true
	// 	}
	// 	if outIn.Scan() {
	//     	cur++
	// 		log.Println(fmt.Sprintf("(%d/%d) %s", cur, count, outIn.Text()))
	//     	cont = true
	// 	}
	// 	if !cont {
	// 		break
	// 	}
	// }

	err = in.Err()
	handleError(err)


	log.Println(fmt.Sprintf("Finished host scan for %d paths on %d domains", len(paths), count))

}

func initWappalyzer() {
	err := webanalyze.DownloadFile(webanalyze.WappalyzerURL, wappalyzeAppsPath)
	handleError(err)
	log.Println("app definition file updated from ", webanalyze.WappalyzerURL)
}

func parseScanResults() {

	db, err := sql.Open("postgres", psqlInfo)
	handleError(err)
	defer db.Close()
	err = db.Ping()
	handleError(err)

	workers := 4
	crawlCount := 0
	searchSubdomain := false
	file, err := os.Open("output/hostscanner/megoutput/index")
	handleError(err)
	defer file.Close()

	results, err := webanalyze.Init(workers, file, wappalyzeAppsPath, crawlCount, searchSubdomain)
	handleError(err)

	log.Printf("Scanning with %v workers.", workers)

	var hostsArray []string
	var servicesArray []string

	for result := range results {
		if result.Error != nil {
			log.Printf("[-] Error for %v: %v", result.Host, result.Error)
			continue
		}

		if len(result.Matches) == 0 {
			continue
		}

		hostName := strings.Replace(strings.Split(result.Host, "//")[1], "/", "", -1)
		hostsArray = append(hostsArray, fmt.Sprintf("'%s'", hostName))

		results := ""
		for i, a := range result.Matches {

			var categories []string

			for _, cid := range a.App.Cats {
				categories = append(categories, webanalyze.AppDefs.Cats[string(cid)].Name)
			}

			if i != 0 {
				results += ", "
			}

			results += fmt.Sprintf("%v %v (%v)", a.AppName, a.Version, strings.Join(categories, " "))
		}

		results = strings.Replace(results, "'", "", -1)
		servicesArray = append(servicesArray, fmt.Sprintf("'%s'", results))
	}


	log.Println(fmt.Sprintf("Uploading %d found services to db...", len(hostsArray)))

	query := `UPDATE "Domains" SET services = data_table.services
				FROM (SELECT unnest(array[` + strings.Join(hostsArray[:], ",") + `]) as name, unnest(array[` + strings.Join(servicesArray[:], ",") + `]) as services)
				as data_table where "Domains".name = data_table.name AND strpos("Domains".services, data_table.services) = 0;`

	_, err = db.Exec(query)
	handleError(err)

	log.Println("Done parsing scan results")


}
