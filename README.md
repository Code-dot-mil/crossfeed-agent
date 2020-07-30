This repository has been archived. Please see https://github.com/deptofdefense/Crossfeed for the new, redesigned version of Crossfeed.

# Crossfeed Agent

Backend scanning infrastructure for Crossfeed, written in go.

Installation:

1. `./init.sh`

2. `go build`

Modules created:

- Port scanner. Uses Rapid7's [Project Sonar](https://www.rapid7.com/research/project-sonar/) database of internet scans to passively find open ports.
- Host scanner, using [meg](https://github.com/tomnomnom/meg) to fetch many paths from many hosts and fingerprint using [Wappalyzer](https://github.com/haccer/subjack/tree/master/subjack)
- Subdomain takeover scanner, using [subjack](https://github.com/haccer/subjack/), to detect improperly configured domains

To be created:

- Subdomain scanner using amass
- and more

Usage:

1. Run `./crossfeed-agent spawner` to wait for incoming requests from web

If you need to run requests manually, run `./crossfeed-agent [command] [args]`, e.g. `./crossfeed-agent scan-hosts /` to scan all live hosts for the root directory.
