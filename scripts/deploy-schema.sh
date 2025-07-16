#!/bin/bash

# Deploy DataGolf analytics schema to Supabase database
# Usage: ./deploy-schema.sh

set -e

# Default database URL - update this to match your Supabase connection
DATABASE_URL=${DATABASE_URL:-"postgresql://postgres:postgres@10.182.159.197:5432/iralogix"}

echo "Deploying DataGolf analytics schema..."
echo "Database: $DATABASE_URL"

# Check if SQL file exists
if [ ! -f "deploy-missing-datagolf-tables.sql" ]; then
    echo "Error: deploy-missing-datagolf-tables.sql not found"
    exit 1
fi

# Try to execute using psql if available
if command -v psql &> /dev/null; then
    echo "Using psql to execute schema deployment..."
    psql "$DATABASE_URL" -f deploy-missing-datagolf-tables.sql
    echo "Schema deployment completed successfully!"
else
    echo "psql not found. Please install PostgreSQL client or use alternative method."
    echo "You can also execute the SQL file manually using your preferred database client."
    exit 1
fi