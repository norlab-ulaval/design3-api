#!/bin/bash

# Script to send a POST request to the vehicle endpoint
# Usage: ./post_vehicle.bash <vehicle_id>

if [ -z "$1" ]; then
    echo "Error: Vehicle ID is required"
    echo "Usage: $0 <vehicle_id>"
    echo "Example: $0 7"
    exit 1
fi

VEHICLE_ID=$1
BASE_URL="http://designiii.ca"

# Example path: from ZD7 through A1 to ZC1
# You can modify this path as needed
PATH='["ZD7", "A1", "ZC1"]'

echo "Sending POST request to /vehicles/${VEHICLE_ID}"

echo "Response:"
/usr/bin/curl -X POST "${BASE_URL}/vehicles/${VEHICLE_ID}" \
    -H "Content-Type: application/json" \
    -d "{\"path\": ${PATH}}"
