#!/bin/bash

docker run --env-file ./.env -p 80:5000 -d --restart always --name design3-api design3-api
