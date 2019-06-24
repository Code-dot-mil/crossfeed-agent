package main

import (
	"fmt"
	"os"
	"io"
	"log"
	"github.com/docopt/docopt-go"
	"github.com/tkanos/gonfig"
	_ "github.com/lib/pq"
)

type Configuration struct {
    Host              string
    Port              string
    User              string
    Password          string
    Dbname            string
    LogPath           string
    Debug             bool
    BeanstalkHost     string
    BeanstalkPollRate int
}

var config Configuration
var psqlInfo string

func main() {
	config = Configuration{}

	err := gonfig.GetConf("config.json", &config)
	handleError(err)
	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.Host, config.Port, config.User, config.Password, config.Dbname)

	if !config.Debug {
		logPath := config.LogPath + getMonth() + ".txt"
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		handleError(err)
		defer f.Close()
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

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
		case "scanPorts":
			scanPorts(getArgs(arguments))
		case "fetchHosts":
			fetchHosts(getArgs(arguments))
		case "spawner":
			initSpawner(getArgs(arguments))
		case "enqueue":
			enqueueJob(getArgs(arguments))
		default:
			fmt.Println("Command not found: " + arguments["<command>"].(string))
		}
	}

}
