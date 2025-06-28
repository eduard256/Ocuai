#!/bin/bash

# PostgreSQL configuration
POSTGRES_DB="ocuai"
POSTGRES_USER="ocuai"
POSTGRES_PASSWORD="ocuai123"
POSTGRES_PORT="5432"
POSTGRES_DATA_DIR="./data/postgresql"

# Function to check if PostgreSQL is installed
check_postgres_installation() {
    if ! command -v initdb >/dev/null 2>&1; then
        echo "Error: PostgreSQL is not installed or not in PATH"
        echo "Please install PostgreSQL first:"
        echo "  Arch Linux: sudo pacman -S postgresql"
        echo "  Ubuntu/Debian: sudo apt install postgresql postgresql-contrib"
        echo "  CentOS/RHEL: sudo dnf install postgresql postgresql-server"
        echo ""
        echo "After installation, you may need to initialize the system database:"
        echo "  sudo -u postgres initdb --locale=C.UTF-8 --encoding=UTF8 -D /var/lib/postgres/data"
        echo "  sudo systemctl enable --now postgresql"
        return 1
    fi
    return 0
}

# Function to check if PostgreSQL is running
is_postgres_running() {
    # First check if the data directory exists and is initialized
    if [ ! -f "$POSTGRES_DATA_DIR/PG_VERSION" ]; then
        return 1
    fi
    
    if command -v pg_ctl >/dev/null 2>&1; then
        pg_ctl status -D "$POSTGRES_DATA_DIR" >/dev/null 2>&1
        return $?
    else
        # Check if postgres process is running on our port
        lsof -ti:$POSTGRES_PORT >/dev/null 2>&1
        return $?
    fi
}

# Function to start PostgreSQL
start_postgres() {
    echo "Starting PostgreSQL database..."
    
    # Check if PostgreSQL is installed
    if ! check_postgres_installation; then
        return 1
    fi
    
    # Create data directory if it doesn't exist
    mkdir -p "$POSTGRES_DATA_DIR"
    
    # Check if database cluster is initialized
    if [ ! -f "$POSTGRES_DATA_DIR/PG_VERSION" ]; then
        echo "Initializing PostgreSQL database cluster..."
        
        # Initialize database if initdb is available
        if command -v initdb >/dev/null 2>&1; then
            initdb -D "$POSTGRES_DATA_DIR" --auth-local=trust --auth-host=trust
            if [ $? -ne 0 ]; then
                echo "Failed to initialize PostgreSQL database"
                return 1
            fi
            echo "PostgreSQL database cluster initialized successfully"
        else
            echo "Warning: initdb not found. Make sure PostgreSQL is properly installed."
            echo "Trying to start PostgreSQL service..."
            sudo systemctl start postgresql 2>/dev/null || true
            return $?
        fi
    fi
    
    # Check if PostgreSQL is already running
    if is_postgres_running; then
        echo "PostgreSQL is already running"
        return 0
    fi
    
    # Start PostgreSQL server
    if command -v pg_ctl >/dev/null 2>&1; then
        # Ensure logs directory exists
        mkdir -p "./data/logs"
        
        echo "Starting PostgreSQL server..."
        pg_ctl start -D "$POSTGRES_DATA_DIR" -l "./data/logs/postgresql.log" -o "-p $POSTGRES_PORT -k /tmp"
        if [ $? -ne 0 ]; then
            echo "Failed to start PostgreSQL server"
            echo "Check logs at ./data/logs/postgresql.log for details"
            return 1
        fi
        
        # Wait for PostgreSQL to start
        sleep 5
        
                 # Create database and user if they don't exist
         if command -v createuser >/dev/null 2>&1 && command -v createdb >/dev/null 2>&1; then
             echo "Creating PostgreSQL user and database..."
             # Use TCP connection or socket in /tmp
             createuser -h localhost -p $POSTGRES_PORT -s "$POSTGRES_USER" 2>/dev/null || \
             PGHOST=/tmp createuser -p $POSTGRES_PORT -s "$POSTGRES_USER" 2>/dev/null || true
             
             createdb -h localhost -p $POSTGRES_PORT -O "$POSTGRES_USER" "$POSTGRES_DB" 2>/dev/null || \
             PGHOST=/tmp createdb -p $POSTGRES_PORT -O "$POSTGRES_USER" "$POSTGRES_DB" 2>/dev/null || true
             
             echo "PostgreSQL database and user created/verified"
         else
             echo "Warning: createuser or createdb not found"
         fi
    else
        echo "Warning: pg_ctl not found. Trying to start PostgreSQL service..."
        sudo systemctl start postgresql 2>/dev/null || true
    fi
    
    # Verify PostgreSQL is running
    if is_postgres_running; then
        echo "PostgreSQL started successfully"
        return 0
    else
        echo "Failed to start PostgreSQL"
        return 1
    fi
}

# Function to stop PostgreSQL
stop_postgres() {
    echo "Stopping PostgreSQL database..."
    
    if ! is_postgres_running; then
        echo "PostgreSQL is not running"
        return 0
    fi
    
    if command -v pg_ctl >/dev/null 2>&1; then
        pg_ctl stop -D "$POSTGRES_DATA_DIR" -m fast
    else
        # Find and kill postgres processes
        local postgres_pids=$(lsof -ti:$POSTGRES_PORT 2>/dev/null)
        if [ ! -z "$postgres_pids" ]; then
            echo "Stopping PostgreSQL processes: $postgres_pids"
            echo "$postgres_pids" | xargs -r kill -TERM 2>/dev/null
            sleep 3
            
            # Force kill if still running
            local remaining_pids=$(lsof -ti:$POSTGRES_PORT 2>/dev/null)
            if [ ! -z "$remaining_pids" ]; then
                echo "Force killing PostgreSQL processes: $remaining_pids"
                echo "$remaining_pids" | xargs -r kill -9 2>/dev/null
            fi
        fi
    fi
    
    echo "PostgreSQL stopped"
}

# Function to kill processes on specific ports with more aggressive checking
kill_processes_on_ports() {
    local ports=("8080" "3000" "1984" "8554" "8555" "5432")
    
    echo "Checking for existing processes on ports: ${ports[*]}"
    
    for port in "${ports[@]}"; do
        echo "Checking port $port..."
        
        # More aggressive port checking - try multiple methods
        local pids=""
        
        # Method 1: lsof (most reliable)
        pids=$(lsof -ti:$port 2>/dev/null | tr '\n' ' ')
        
        # Method 2: fuser if available
        if [ -z "$pids" ] && command -v fuser >/dev/null 2>&1; then
            pids=$(fuser $port/tcp 2>/dev/null | tr '\n' ' ')
        fi
        
        # Method 3: netstat if lsof fails
        if [ -z "$pids" ] && command -v netstat >/dev/null 2>&1; then
            pids=$(netstat -tlnp 2>/dev/null | grep ":$port " | awk '{print $7}' | cut -d'/' -f1 | grep -v '-' | tr '\n' ' ')
        fi
        
        # Method 4: ss if others fail
        if [ -z "$pids" ] && command -v ss >/dev/null 2>&1; then
            pids=$(ss -tlnp 2>/dev/null | grep ":$port " | awk '{print $6}' | cut -d'=' -f2 | cut -d',' -f1 | tr '\n' ' ')
        fi
        
        if [ ! -z "$pids" ]; then
            echo "Found processes on port $port: $pids"
            echo "Killing processes on port $port..."
            
            # Try to kill gracefully first
            for pid in $pids; do
                if [ "$pid" != "" ] && [ "$pid" != "-" ] && [ "$pid" -gt 0 ] 2>/dev/null; then
                    kill "$pid" 2>/dev/null && echo "Sent TERM signal to PID $pid"
                fi
            done
            
            sleep 3
            
            # Check if processes are still running and force kill if necessary
            local remaining_pids=""
            remaining_pids=$(lsof -ti:$port 2>/dev/null | tr '\n' ' ')
            
            if [ ! -z "$remaining_pids" ]; then
                echo "Force killing remaining processes on port $port: $remaining_pids"
                for pid in $remaining_pids; do
                    if [ "$pid" != "" ] && [ "$pid" != "-" ] && [ "$pid" -gt 0 ] 2>/dev/null; then
                        kill -9 "$pid" 2>/dev/null && echo "Sent KILL signal to PID $pid"
                    fi
                done
                
                # Extra aggressive cleanup using fuser if available
                if command -v fuser >/dev/null 2>&1; then
                    echo "Using fuser to force kill processes on port $port"
                    fuser -k $port/tcp 2>/dev/null || true
                fi
                
                # Final verification with multiple attempts
                for attempt in {1..3}; do
                    sleep 2
                    local final_check=$(lsof -ti:$port 2>/dev/null)
                    if [ -z "$final_check" ]; then
                        echo "Port $port is now free"
                        break
                    elif [ $attempt -eq 3 ]; then
                        echo "Warning: Port $port still has processes after $attempt attempts: $final_check"
                    fi
                done
            else
                echo "Port $port is now free"
            fi
        else
            echo "Port $port is free"
        fi
    done
    
    echo "Port cleanup completed"
    echo ""
}

# Kill any existing processes on our ports before starting
kill_processes_on_ports

# Create necessary directories if they don't exist
mkdir -p data/db data/videos data/logs models data/postgresql

# Export environment variables for PostgreSQL
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="$POSTGRES_PORT"
export POSTGRES_USER="$POSTGRES_USER"
export POSTGRES_PASSWORD="$POSTGRES_PASSWORD"
export POSTGRES_DB="$POSTGRES_DB"
export POSTGRES_SSLMODE="disable"

# Export other environment variables
export OCUAI_DATABASE_PATH="./data/db/ocuai.db"
export OCUAI_VIDEO_PATH="./data/videos"
export OCUAI_PORT="8080"

# Function to stop all processes on exit
cleanup() {
    echo ""
    echo "Cleaning up processes..."
    
    # Kill background jobs
    local job_pids=$(jobs -p)
    if [ ! -z "$job_pids" ]; then
        echo "Stopping background processes: $job_pids"
        kill $job_pids 2>/dev/null
        sleep 2
        
        # Force kill if still running
        local remaining_jobs=$(jobs -p)
        if [ ! -z "$remaining_jobs" ]; then
            echo "Force killing remaining jobs: $remaining_jobs"
            kill -9 $remaining_jobs 2>/dev/null
        fi
    fi
    
    # Stop PostgreSQL
    stop_postgres
    
    # Final port cleanup
    echo "Final port cleanup..."
    kill_processes_on_ports
    
    echo "Cleanup completed"
}

# Set up cleanup on script exit
trap cleanup EXIT INT TERM

# Start PostgreSQL database
start_postgres
if [ $? -ne 0 ]; then
    echo "Failed to start PostgreSQL. Exiting..."
    exit 1
fi

# Get the project root directory
PROJECT_ROOT=$(pwd)

# Start frontend
echo "Starting frontend..."
cd web && npm run dev &
FRONTEND_PID=$!

# Wait a bit for frontend to start and verify it's running
sleep 5
if ! kill -0 $FRONTEND_PID 2>/dev/null; then
    echo "Frontend failed to start"
    exit 1
fi

# Verify frontend is actually listening on port 3000
frontend_check=0
for i in {1..15}; do
    # Try multiple methods to check port 3000
    if lsof -ti:3000 >/dev/null 2>&1 || \
       netstat -tlnp 2>/dev/null | grep ":3000 " >/dev/null || \
       ss -tlnp 2>/dev/null | grep ":3000 " >/dev/null || \
       curl -s http://localhost:3000 >/dev/null 2>&1; then
        echo "Frontend is running on port 3000"
        frontend_check=1
        break
    fi
    echo "Waiting for frontend to start on port 3000... ($i/15)"
    sleep 3
done

if [ $frontend_check -eq 0 ]; then
    echo "Frontend is not responding on port 3000 after 45 seconds"
    echo "This might be normal if the frontend takes longer to start"
    echo "Continuing with backend startup..."
    # Don't exit, just warn and continue
fi

# Build and start backend with go2rtc auto-start
echo "Starting backend..."
cd "$PROJECT_ROOT"

# Build the PostgreSQL version with go2rtc integration
echo "Building backend with go2rtc integration..."
make -f Makefile.postgres build

# Start the built binary with all necessary environment variables
export PORT="8080"
export GO2RTC_PATH="./data/go2rtc/bin/go2rtc"
export GO2RTC_CONFIG="./data/go2rtc/go2rtc.yaml"

echo "🚀 Starting OcuAI with automatic go2rtc startup..."
./bin/ocuai-postgres

# Wait for all background processes
wait 