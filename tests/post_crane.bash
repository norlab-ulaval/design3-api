#!/bin/bash

# Script to send a POST request to the crane endpoint
# Usage: ./post_crane.bash <crane_id>

if [ -z "$1" ]; then
    echo "Error: Crane ID is required"
    echo "Usage: $0 <crane_id>"
    echo "Example: $0 7"
    exit 1
fi

CRANE_ID=$1
BASE_URL="http://designiii.ca"

# Number of tokens on the balance
# You can modify this value as needed
NB_TOKENS=3

echo "Sending POST request to /cranes/${CRANE_ID}"

echo "Response:"
/usr/bin/curl -X POST "${BASE_URL}/cranes/${CRANE_ID}" \
    -H "Content-Type: application/json" \
    -d "{\"nb_tokens\": ${NB_TOKENS}}"
