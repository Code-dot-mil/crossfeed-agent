package main

import (
	"database/sql"
	"fmt"
	"github.com/docopt/docopt-go"
	_ "github.com/lib/pq"
	"github.com/tkanos/gonfig"
	"io"
	"log"
	"os"
	"path"
)

type Configuration struct {
	DB_HOST             string
	DB_PORT             string
	DB_USER             string
	DB_PASSWORD         string
	DB_NAME             string
	LOG_PATH            string
	DEBUG               bool
	BEANSTALK_HOST      string
	BEANSTALK_POLL_RATE int
	SONAR_API_KEY       string
}

var config Configuration
var psqlInfo string
var db *sql.DB

func main() {
	config = Configuration{}

	err := gonfig.GetConf("config.json", &config)
	handleError(err)
	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.DB_HOST, config.DB_PORT, config.DB_USER, config.DB_PASSWORD, config.DB_NAME)

	if !config.DEBUG {
		logPath := path.Join(config.LOG_PATH, getMonth()+".txt")
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		handleError(err)
		defer f.Close()
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

	db, err = sql.Open("postgres", psqlInfo)
	handleError(err)
	err = db.Ping()
	handleError(err)

	usage := `Crossfeed agent. Used to execute backend scans on a cron job. Scans are pushed to remote crossfeed database.

Examples:
crossfeed-agent scanPorts 2019-05-20-1558346873-https_get_443 443

Usage:
  crossfeed-agent <command> [<args>...]
  crossfeed-agent -h | --help
  crossfeed-agent --version

Options:
  -h --help     Show this screen.
  --version     Show version.`

	arguments, _ := docopt.ParseDoc(usage)
	if hasKey(arguments, "<command>") {
		switch arguments["<command>"].(string) {
		case "scan-ports":
			scanPorts(getArgs(arguments))
		case "scan-hosts":
			fetchHosts(getArgs(arguments))
		case "subjack":
			subjack(getArgs(arguments))
		case "spawner":
			initSpawner(getArgs(arguments))
		case "enqueue":
			enqueueJob(getArgs(arguments))
		default:
			fmt.Println("Command not found: " + arguments["<command>"].(string))
		}
	}

}
