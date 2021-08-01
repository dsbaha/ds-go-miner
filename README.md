# ds-go-miner
```
Duino Coin Miner in Golang
```
# Compile Static Binary
```
CGO_ENABLED=0 go build
```
# Create Docker File
```
docker build -t ds-go-miner .
```
# Run Docker Container
```
docker run -ti --rm -e MINERNAME="dsminer" ds-go-miner:latest
```
# Environment Variable Options
```console
DUCOSERVER Server Address and Port
MINERNAME  Miner Name
HOSTNAME   Rig ID
DIFF       Difficulty LOW/MEDIUM/NET
ALGO       xxhash/ducos1a
```
# Runtime Options
```console
  -algo string
        Algorithm select xxhash/ducos1a, environment variable ALGO
  -debug
        console log send/receive messages.
  -diff string
        Difficulty LOW/MEDIUM/NET, environment variable DIFF
  -id string
        Rig ID, environment variable HOSTNAME (default "332b33abe2a8")
  -name string
        Miner Name, enviromnet variable MINERNAME
  -quiet
        Turn off Console Logging
  -server string
        Server Address and Port, environment variable DUCOSERVER
  -skip
        Skip the first 'Difficulty' Hash Range
  -threads int
        Number of Threads to Run (default 1)
```
