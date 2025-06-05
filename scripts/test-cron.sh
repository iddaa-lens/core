#!/bin/bash

# Test script for running individual cron jobs
set -e

echo "Starting services..."
docker compose up -d postgres

echo "Waiting for database to be ready..."
until docker compose exec postgres pg_isready -U iddaa -d iddaa_core; do
  echo "Database is unavailable - sleeping"
  sleep 2
done

echo "Running migrations..."
docker compose run --rm cron migrate -path migrations -database "postgres://iddaa:iddaa123@postgres:5432/iddaa_core?sslmode=disable" up

echo "Building cron service..."
docker compose build cron

echo "Testing competition sync job..."
docker compose run --rm cron ./cron --job=sports --once

echo "Testing config sync job..."
docker compose run --rm cron ./cron --job=events --once

echo "Done!"