#!/bin/bash

# Change to the script's directory
cd "$(dirname "$0")"

# Set the path to the SQLite datab relative to the script location
DATABASE_PATH="social-network/backend/database.db"

# Function to apply a single migration file
apply_migration() {
    local migration_file=$1
    echo "Applying migration: $migration_file"
    sqlite3 $DATABASE_PATH < $migration_file
    if [ $? -ne 0 ]; then
        echo "Error applying migration: $migration_file"
        exit 1
    fi
}

# Check if datab file exists
if [ ! -f $DATABASE_PATH ]; then
    echo "Database does not exist at: $DATABASE_PATH"
    exit 1
fi

# Find all .sql files in the current directory and apply them
for migration_file in *.sql; do
    apply_migration "$migration_file"
done

echo "All migrations applied successfully."
