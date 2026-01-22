#!/bin/bash

# Script to send a GET request to the vehicles endpoint
# Usage: ./get_vehicles.bash [vehicle_id]
# If vehicle_id is provided, gets info for that specific vehicle
# If no vehicle_id is provided, gets info for all vehicles

BASE_URL="http://designiii.ca"

if [ -z "$1" ]; then
    # Get all vehicles
    echo "Sending GET request to /vehicles/"
    echo ""

    echo "Response:"
    /usr/bin/curl -X GET "${BASE_URL}/vehicles/" \
        -H "Content-Type: application/json"
else
    # Get specific vehicle
    VEHICLE_ID=$1
    echo "Sending GET request to /vehicles/${VEHICLE_ID}"
    echo ""

    echo "Response:"
    /usr/bin/curl -X GET "${BASE_URL}/vehicles/${VEHICLE_ID}" \
        -H "Content-Type: application/json"
fi