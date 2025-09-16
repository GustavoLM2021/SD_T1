#!/bin/bash

# Basic use: ./simple_dimex.sh (give chmod +x simple_dimex.sh first) or bash simple_dimex.sh

echo "=== DIMEX Process Manager ==="
echo


ADDRESSES="127.0.0.1:5000 127.0.0.1:6001 127.0.0.1:7002"

start_dimex() {
    echo "Starting 3 procs..."
    echo "ctrl+c to stop all procs"
    echo
    
   
    rm -f mxOUT.txt
    
    
    go run useDIMEX-f.go 0 $ADDRESSES --s &
    PID1=$!

    go run useDIMEX-f.go 1 $ADDRESSES &
    PID2=$!
    
    go run useDIMEX-f.go 2 $ADDRESSES &
    PID3=$!
    
    echo "Procs in effect:"
    echo "  Proc 0: PID $PID1"
    echo "  Proc 1: PID $PID2" 
    echo "  Proc 2: PID $PID3"
    echo
    
    
    wait $PID1 $PID2 $PID3
}

# stop all procs
cleanup() {
    echo
    echo "Stoping all procs..."
    pkill -f "useDIMEX-f.go"
    echo "Procs Stopped."
    exit 0
}

# ctrl+c 
trap cleanup SIGINT SIGTERM

# start procs
start_dimex