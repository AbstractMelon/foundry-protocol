#!/bin/bash

# Start the velocity server
go run ./server/main.go &

# Start the gateway
go run ./gateway/main.go &

wait
