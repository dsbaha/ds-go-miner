# ds-go-miner
Duino Coin Miner in Golang

# Compile Static Binary
CGO_ENABLED=0 go build

# Create Docker File
docker build -t ds-go-miner .

# Run Docker Container
docker run -ti --rm -e MINERNAME="dsminer" ds-go-miner:latest

# Environment Variable Options
DUCOSERVER Server Address and Port
MINERNAME  Miner Name
HOSTNAME   Rig ID
DIFF       Difficulty LOW/MEDIUM/NET
ALGO       xxhash/ducos1a
