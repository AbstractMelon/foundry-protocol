#!/bin/bash

# Start the decaf server
go run ./server/decaf/main.go &

# Start the velocity server
node ./server/velocity/src/index.js &

# Start the gateway
go run ./gateway/main.go &

wait
