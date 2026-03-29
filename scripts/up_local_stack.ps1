$ErrorActionPreference = "Stop"

docker compose -f infra/docker/docker-compose.yml config | Out-Null
docker compose -f infra/docker/docker-compose.yml up -d --build
