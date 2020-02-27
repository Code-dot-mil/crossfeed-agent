package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/lib/pq"
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
		http.MethodGet,
		nil,
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
					http.MethodGet,
					nil,
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

	log.Println("Sorting...")
	_, err := exec.Command("sort", "-o", hostsPath, hostsPath).Output()
	handleError(err)

	for index, scan := range scans {
		port := scan.Port
		downloadUrl := scan.DownloadUrl

		var sonarPath string = "output/portscan/sonar/" + port + ".txt"
		var outpath string = "output/portscan/" + port + "-" + scanID + ".txt"

		// Download Sonar data for url
		log.Println("Starting port scan for port " + port + " using " + downloadUrl)
		_, err := exec.Command("/bin/sh", "scripts/prepare_files.sh", downloadUrl, port).Output()
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

		query := "SELECT ip FROM \"Domains\" WHERE ports LIKE '%%" + port + "%%';"
		rows, err := db.Query(query)
		handleError(err)

		existingPorts := make(map[string]bool)
		var ip string
		for rows.Next() {
			err := rows.Scan(&ip)
			handleError(err)
			existingPorts[ip] = true
		}
		rows.Close()

		file, err := os.Open(outpath)
		handleError(err)

		scanner := bufio.NewScanner(file)
		var ipsArray []string
		var portsArray []string
		var alertOutput []string
		for scanner.Scan() {
			var ip = scanner.Text()
			if _, exists := existingPorts[ip]; exists {
				continue
			}
			ipsArray = append(ipsArray, ip)
			portsArray = append(portsArray, port)
			alertOutput = append(alertOutput, ip)
		}
		file.Close()

		if len(ipsArray) > 0 {
			log.Println(fmt.Sprintf("Uploading %d found open ports to db...", len(ipsArray)))

			updateTaskOutput(fmt.Sprintf("Scan Ports (%s)", port), fmt.Sprintf("Port %s is now open for:\n%s", port, strings.Join(alertOutput, "\n")), 3)

			query = `UPDATE "Domains" SET ports = CASE WHEN "Domains".ports IS NOT NULL AND "Domains".ports <> '' THEN "Domains".ports || ',' || data_table.ports ELSE data_table.ports END
						FROM (SELECT unnest($1::text[]) as ip, unnest($2::text[]) as ports)
						as data_table where "Domains".ip = data_table.ip AND ("Domains".ports IS NULL OR strpos("Domains".ports, data_table.ports) = 0);`

			_, err = db.Exec(query, pq.Array(ipsArray), pq.Array(portsArray))
			handleError(err)
		} else {
			log.Println("No new ports found.")
		}

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
