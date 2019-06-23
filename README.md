# Crossfeed Agent

Backend scanning infrastructure for Crossfeed, written in go.

Installation:

1. `./init.sh`

2. `go build`

Modules created:

- Port scanner. Uses Rapid7's [Project Sonar](https://www.rapid7.com/research/project-sonar/) database of internet scans to passively find open ports.

Usage:

This will be put on a cron job eventually. For the time being:

1. `./prepare_files.sh 2019-05-20-1558382559-http_get_80 sonar-80`
2. `./crossfeed-agent run scanPorts scan -p 80`

This fetches all domains from the database, compares against the Sonar csv of scanned ports, and updates the database with new found ports.
