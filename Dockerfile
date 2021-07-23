FROM scratch
COPY ds-go-miner /
ENTRYPOINT [ "/ds-go-miner" ]
