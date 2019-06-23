package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	_ "github.com/lib/pq"
)

var psqlInfo string = fmt.Sprintf("host=%s port=%d user=%s "+
	"password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)

func main() {
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
