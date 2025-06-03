#!/bin/bash
# Script to clear events data for fresh sync

DATABASE_URL="${DATABASE_URL:-postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable}"

echo "Clearing events data..."

# Use a here document to run multiple SQL commands
psql "$DATABASE_URL" << EOF
-- Clear in correct order due to foreign key constraints
DELETE FROM odds;
DELETE FROM events;
DELETE FROM teams;
DELETE FROM market_types;

-- Show counts
SELECT 'Teams: ' || COUNT(*) FROM teams;
SELECT 'Events: ' || COUNT(*) FROM events;
SELECT 'Odds: ' || COUNT(*) FROM odds;
EOF

echo "Events data cleared!"