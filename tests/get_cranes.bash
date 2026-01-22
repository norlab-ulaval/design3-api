#!/bin/bash

# Script to send a GET request to the cranes endpoint
# Usage: ./get_cranes.bash [crane_id]
# If crane_id is provided, gets info for that specific crane
# If no crane_id is provided, gets info for all cranes

BASE_URL="http://designiii.ca"

if [ -z "$1" ]; then
    # Get all cranes
    echo "Sending GET request to /cranes/"
    echo ""

    echo "Response:"
    /usr/bin/curl -X GET "${BASE_URL}/cranes/" \
        -H "Content-Type: application/json"
else
    # Get specific crane
    CRANE_ID=$1
    echo "Sending GET request to /cranes/${CRANE_ID}"
    echo ""

    echo "Response:"
    /usr/bin/curl -X GET "${BASE_URL}/cranes/${CRANE_ID}" \
        -H "Content-Type: application/json"
fi
