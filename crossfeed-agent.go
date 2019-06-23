package main

import (
	"fmt"
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
}

var config Configuration

var psqlInfo string

func main() {
	config := Configuration{}

	err := gonfig.GetConf("config.json", &config)
	handleError(err)
	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", config.Host, config.Port, config.User, config.Password, config.Dbname)

	usage := `Crossfeed agent. Used to execute backend scans on a cron job. Scans are pushed to remote crossfeed database.

Usage:
  crossfeed-agent run <command> [<subcommand>] [--ports=<p>]
  crossfeed-agent -h | --help
  crossfeed-agent --version

Options:
  -h --help     Show this screen.
  -p --ports=<p>    port to scan
  --version     Show version.`

	arguments, _ := docopt.ParseDoc(usage)
	if arguments["run"].(bool) {
		switch arguments["<command>"].(string) {
		case "scanPorts":
			scanPorts(arguments)
		default:
			fmt.Println("Command not found: " + arguments["<command>"].(string))
		}
	}

}
