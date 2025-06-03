#!/bin/bash

# Database inspection script for testing cron results

DATABASE_URL="postgres://iddaa:iddaa123@localhost:5432/iddaa_core?sslmode=disable"

echo "=== Database Inspection ==="
echo ""

echo "1. Checking competitions table..."
docker compose exec postgres psql -U iddaa -d iddaa_core -c "
SELECT 
    COUNT(*) as total_competitions,
    COUNT(DISTINCT sport_id) as unique_sports,
    COUNT(DISTINCT country_code) as unique_countries
FROM competitions;"

echo ""
echo "2. Latest competitions by sport..."
docker compose exec postgres psql -U iddaa -d iddaa_core -c "
SELECT 
    sport_id,
    country_code,
    COUNT(*) as count,
    MAX(updated_at) as last_updated
FROM competitions 
GROUP BY sport_id, country_code 
ORDER BY sport_id, country_code
LIMIT 10;"

echo ""
echo "3. Checking app_config table..."
docker compose exec postgres psql -U iddaa -d iddaa_core -c "
SELECT 
    platform,
    sportoto_program_name,
    payin_end_date,
    created_at,
    updated_at
FROM app_config 
ORDER BY updated_at DESC;"

echo ""
echo "4. Sample competition data..."
docker compose exec postgres psql -U iddaa -d iddaa_core -c "
SELECT 
    iddaa_id,
    sport_id,
    country_code,
    short_name,
    full_name,
    updated_at
FROM competitions 
ORDER BY updated_at DESC 
LIMIT 5;"

echo ""
echo "=== Inspection Complete ==="