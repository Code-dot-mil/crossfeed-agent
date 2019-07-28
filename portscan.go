package main

import (
	"bufio"
	"fmt"
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

type SonarStudy struct {
	Name          string
	Uniqid        string
	Created_at    string
	Sonarfile_set []string
}

type DownloadUrl struct {
	Url string
}

type ScanInfo struct {
	Port        string
	DownloadUrl string
}

func scanPorts(args []string) {
	log.SetPrefix("[portscan] ")
	if len(args) < 1 {
		log.Fatal("Please provide the ports to scan")
	}
	scanLatestResults(strings.Split(args[0], ","), args[1])
}

func scanLatestResults(ports []string, taskID string) {
	var availableFiles SonarStudy
	fetchExternalAPI("https://us.api.insight.rapid7.com/opendata/studies/sonar.tcp/",
		map[string]string{
			"X-Api-Key": config.SONAR_API_KEY,
		},
		&availableFiles)

	scans := make([]ScanInfo, len(ports))

	for i, port := range ports {
		for _, file := range availableFiles.Sonarfile_set {
			if strings.HasSuffix(file, fmt.Sprintf("_%s.csv.gz", port)) {
				var downloadUrl DownloadUrl
				fetchExternalAPI("https://us.api.insight.rapid7.com/opendata/studies/sonar.tcp/"+file+"/download/",
					map[string]string{
						"X-Api-Key": config.SONAR_API_KEY,
					},
					&downloadUrl)
				scans[i] = ScanInfo{
					Port:        port,
					DownloadUrl: downloadUrl.Url,
				}
				break
			}
		}
	}

	initPortScan(scans, taskID)
}

func initPortScan(scans []ScanInfo, taskID string) {
	var (
		scanID    = getTimestamp(true)
		hostsPath = "output/portscan/ips-" + scanID + ".txt"
	)

	log.Println("Exporting hosts to file...")

	// Fetch all ip addresses from db and sort
	query := `SELECT ip FROM "Domains" WHERE ip IS NOT NULL AND ip <> '';`
	writeQueryToFile(query, hostsPath)

	query = `SELECT name, ports FROM "Domains" WHERE ports LIKE '%80%' OR ports LIKE '%443%';`
	rows, err := db.Query(query)
	handleError(err)

	existingPorts := make(map[string][]string)
	var name string
	var ports string
	for rows.Next() {
		err := rows.Scan(&name, &ports)
		handleError(err)
		existingPorts[name] = strings.Split(strings.Replace(ports, " ", "", -1), ",")
	}
	rows.Close()

	log.Println("Sorting...")
	_, err = exec.Command("sort", "-o", hostsPath, hostsPath).Output()
	handleError(err)

	for index, scan := range scans {
		port := scan.Port
		downloadUrl := scan.DownloadUrl

		var sonarPath string = "output/portscan/sonar/" + port + ".txt"
		var outpath string = "output/portscan/" + port + "-" + scanID + ".txt"

		// Download Sonar data for url
		log.Println("Starting port scan for port " + port + " using " + downloadUrl)
		_, err := exec.Command("/bin/sh", "prepare_files.sh", downloadUrl, port).Output()
		handleError(err)

		// Compare lines in both sorted files
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

		log.Println(fmt.Sprintf("Done scanning ports for port %s using %s", port, downloadUrl))
		updateTaskPercentage(taskID, 100*(index+1)/len(scans))
	}

	log.Println(fmt.Sprintf("Finished port scan for %d ports.", len(scans)))

	err = os.Remove(hostsPath)
	handleError(err)

}
