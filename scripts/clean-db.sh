#!/bin/bash

echo "🧹 Database Cleanup Script"
echo "=========================="
echo ""

# PostgreSQL configuration
POSTGRES_DATA_DIR="./data/postgresql"
POSTGRES_LOG_FILE="./data/logs/postgresql.log"

# Function to stop PostgreSQL
stop_postgres() {
    echo "Stopping PostgreSQL..."
    
    # Try pg_ctl first
    if [ -f "$POSTGRES_DATA_DIR/PG_VERSION" ] && command -v pg_ctl >/dev/null 2>&1; then
        pg_ctl stop -D "$POSTGRES_DATA_DIR" -m immediate 2>/dev/null || true
    fi
    
    # Kill any remaining postgres processes
    pkill -f postgres 2>/dev/null || true
    
    # Kill processes on port 5432
    local postgres_pids=$(lsof -ti:5432 2>/dev/null)
    if [ ! -z "$postgres_pids" ]; then
        echo "Killing PostgreSQL processes: $postgres_pids"
        echo "$postgres_pids" | xargs -r kill -9 2>/dev/null || true
    fi
    
    echo "PostgreSQL stopped"
}

# Function to show help
show_help() {
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  --full     Complete database reset (removes all data and recreates cluster)"
    echo "  --data     Clear only table data (keeps structure)"
    echo "  --tables   Drop all tables and recreate from migrations"
    echo "  --help     Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --full      # Complete reset (recommended)"
    echo "  $0 --data      # Keep tables, clear data only"
    echo "  $0 --tables    # Drop and recreate tables"
}

# Function for full reset
full_reset() {
    echo "🔥 Performing FULL database reset..."
    echo "This will completely remove all PostgreSQL data!"
    
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Operation cancelled."
        exit 0
    fi
    
    stop_postgres
    
    echo "Removing PostgreSQL data directory..."
    rm -rf "$POSTGRES_DATA_DIR"
    
    echo "Removing PostgreSQL logs..."
    rm -f "$POSTGRES_LOG_FILE"
    
    echo "✅ Full reset completed!"
    echo "Run './scripts/dev.sh' to reinitialize the database"
}

# Function to clear data only
clear_data() {
    echo "🧽 Clearing table data..."
    
    read -p "This will delete all data but keep table structure. Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Operation cancelled."
        exit 0
    fi
    
    # Check if PostgreSQL is running
    if ! lsof -ti:5432 >/dev/null 2>&1; then
        echo "PostgreSQL is not running. Starting it..."
        ./scripts/dev.sh >/dev/null 2>&1 &
        sleep 5
        if ! lsof -ti:5432 >/dev/null 2>&1; then
            echo "❌ Failed to start PostgreSQL"
            exit 1
        fi
    fi
    
    echo "Connecting to database and clearing data..."
    PGHOST=/tmp psql -p 5432 -d ocuai -U ocuai -c "
        TRUNCATE TABLE cameras CASCADE;
        -- Add more TRUNCATE statements here for other tables
    " 2>/dev/null || \
    psql -h localhost -p 5432 -d ocuai -U ocuai -c "
        TRUNCATE TABLE cameras CASCADE;
        -- Add more TRUNCATE statements here for other tables
    " 2>/dev/null || {
        echo "❌ Failed to clear data. Make sure PostgreSQL is running and accessible."
        exit 1
    }
    
    echo "✅ Data cleared successfully!"
}

# Function to drop and recreate tables
recreate_tables() {
    echo "🔄 Recreating tables from migrations..."
    
    read -p "This will drop all tables and recreate them. Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Operation cancelled."
        exit 0
    fi
    
    # Check if PostgreSQL is running
    if ! lsof -ti:5432 >/dev/null 2>&1; then
        echo "PostgreSQL is not running. Starting it..."
        ./scripts/dev.sh >/dev/null 2>&1 &
        sleep 5
        if ! lsof -ti:5432 >/dev/null 2>&1; then
            echo "❌ Failed to start PostgreSQL"
            exit 1
        fi
    fi
    
    echo "Dropping all tables..."
    PGHOST=/tmp psql -p 5432 -d ocuai -U ocuai -c "
        DROP SCHEMA public CASCADE;
        CREATE SCHEMA public;
        GRANT ALL ON SCHEMA public TO ocuai;
        GRANT ALL ON SCHEMA public TO public;
    " 2>/dev/null || \
    psql -h localhost -p 5432 -d ocuai -U ocuai -c "
        DROP SCHEMA public CASCADE;
        CREATE SCHEMA public;
        GRANT ALL ON SCHEMA public TO ocuai;
        GRANT ALL ON SCHEMA public TO public;
    " 2>/dev/null || {
        echo "❌ Failed to drop tables. Make sure PostgreSQL is running and accessible."
        exit 1
    }
    
    echo "Tables dropped. Restart your application to run migrations."
    echo "✅ Tables recreated successfully!"
}

# Main script logic
case "${1:-}" in
    --full)
        full_reset
        ;;
    --data)
        clear_data
        ;;
    --tables)
        recreate_tables
        ;;
    --help)
        show_help
        ;;
    "")
        echo "❌ No option specified."
        echo ""
        show_help
        exit 1
        ;;
    *)
        echo "❌ Unknown option: $1"
        echo ""
        show_help
        exit 1
        ;;
esac 