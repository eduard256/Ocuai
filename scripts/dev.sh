#!/bin/bash

# Create necessary directories if they don't exist
mkdir -p data/db data/videos models

# Export environment variables
export OCUAI_DATABASE_PATH="./data/db/ocuai.db"
export OCUAI_VIDEO_PATH="./data/videos"
export OCUAI_PORT="8080"

# Function to stop background processes on exit
cleanup() {
    echo "Stopping all processes..."
    kill $(jobs -p) 2>/dev/null
}

# Set up cleanup on script exit
trap cleanup EXIT

# Get the project root directory
PROJECT_ROOT=$(pwd)

# Start frontend
echo "Starting frontend..."
cd web && npm run dev &

# Wait a bit for frontend to start
sleep 2

# Start backend
echo "Starting backend..."
cd "$PROJECT_ROOT"
go run cmd/ocuai/main.go

# Wait for all background processes
wait 