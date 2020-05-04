#!/bin/bash



# Start the server in background
./server &

# Run the tests
go test ./tests
