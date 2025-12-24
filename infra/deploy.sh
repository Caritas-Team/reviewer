#!/bin/bash

set -e

run_compose() { docker compose -p dev -f docker-compose.yaml -f infra/docker-compose.prod.yaml "$@"; }

# build containers with build sections in compose
run_compose build -q

# start project
run_compose up -d --quiet-pull

# show containers status
run_compose ps

# Clear old backedn images
docker image prune -f