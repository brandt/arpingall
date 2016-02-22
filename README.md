# arpingall

Sends an ARP for every IP on every interface to the interface's default gateway.


## Requirements

- Linux
- `arping` installed


## About

Just a simple wrapper around `arping`.

Built because one of our provider's switches would occasionaly get amnesia.  As a temporary workaround, we ran this as a cron job.


## Known Issues

- Only IPv4 supported.


## Authors

- J. Brandt Buckley
