#!/bin/bash

# OcuAI Development Script for PostgreSQL with Docker
# This script uses Docker PostgreSQL instead of local installation

set -e

echo "🚀 Starting OcuAI Development Environment (PostgreSQL + Go2rtc)"

# Configuration
POSTGRES_PORT=5432
POSTGRES_USER=ocuai
POSTGRES_PASSWORD=ocuai123
POSTGRES_DB=ocuai

# Function to check if port is in use
is_port_in_use() {
    local port=$1
    lsof -ti:$port >/dev/null 2>&1
}

# Function to kill processes on port
kill_port() {
    local port=$1
    local pids=$(lsof -ti:$port 2>/dev/null)
    if [ ! -z "$pids" ]; then
        echo "Killing processes on port $port: $pids"
        echo "$pids" | xargs -r kill -TERM 2>/dev/null
        sleep 2
        # Force kill if still running
        local remaining=$(lsof -ti:$port 2>/dev/null)
        if [ ! -z "$remaining" ]; then
            echo "$remaining" | xargs -r kill -9 2>/dev/null
        fi
    fi
}

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."
    
    # Kill background jobs
    local job_pids=$(jobs -p)
    if [ ! -z "$job_pids" ]; then
        echo "Stopping background processes..."
        kill $job_pids 2>/dev/null
        sleep 2
        # Force kill if still running
        local remaining_jobs=$(jobs -p)
        if [ ! -z "$remaining_jobs" ]; then
            kill -9 $remaining_jobs 2>/dev/null
        fi
    fi
    
    # Stop processes on our ports
    echo "Stopping services..."
    kill_port 8080  # Backend
    kill_port 3000  # Frontend
    kill_port 1984  # Go2rtc
    
    echo "✅ Cleanup completed"
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Check and kill existing processes on our ports
echo "🔍 Checking for existing processes..."
kill_port 8080
kill_port 3000
kill_port 1984

# Start PostgreSQL with Docker
echo "🐳 Starting PostgreSQL database..."
make -f Makefile.postgres docker-up

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if docker exec ocuai-postgres pg_isready -U $POSTGRES_USER -d $POSTGRES_DB >/dev/null 2>&1; then
        echo "✅ PostgreSQL is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ PostgreSQL failed to start after 30 seconds"
        exit 1
    fi
    sleep 1
done

# Build the application
echo "🏗️ Building application..."
make -f Makefile.postgres build

# Create necessary directories
mkdir -p data/logs data/videos data/go2rtc

# Export environment variables
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="$POSTGRES_PORT"
export POSTGRES_USER="$POSTGRES_USER"
export POSTGRES_PASSWORD="$POSTGRES_PASSWORD"
export POSTGRES_DB="$POSTGRES_DB"
export POSTGRES_SSLMODE="disable"
export PORT="8080"
export GO2RTC_PATH="./data/go2rtc/bin/go2rtc"
export GO2RTC_CONFIG="./data/go2rtc/go2rtc.yaml"

# Start frontend
echo "🎨 Starting frontend..."
cd web && npm run dev &
FRONTEND_PID=$!
cd ..

# Wait for frontend to start
echo "⏳ Waiting for frontend to start..."
for i in {1..20}; do
    if is_port_in_use 3000; then
        echo "✅ Frontend is running on http://localhost:3000"
        break
    fi
    if [ $i -eq 20 ]; then
        echo "⚠️ Frontend is taking longer than expected to start"
        echo "Continuing with backend startup..."
        break
    fi
    sleep 2
done

# Start backend with automatic go2rtc startup
echo "🚀 Starting backend with go2rtc auto-start..."
echo "📡 Go2rtc will be available at: http://10.0.1.2:1984/"
echo "🌐 Backend will be available at: http://localhost:8080/"
echo "🎯 Frontend will be available at: http://localhost:3000/"
echo ""
echo "Press Ctrl+C to stop all services"

./bin/ocuai-postgres

# Wait for background processes
wait 