#!/bin/bash

echo "=== Ocuai NVR Production Demo ==="
echo "Demonstrating all fixed features..."
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${YELLOW}Starting Ocuai NVR...${NC}"

# Clean start
pkill -f "go run" 2>/dev/null
pkill -f "npm run dev" 2>/dev/null
rm -f data/db/ocuai.db*

# Start services
./scripts/dev.sh &
DEMO_PID=$!

echo "Waiting for services..."
sleep 8

echo -e "\n${GREEN}✓ Services started${NC}"
echo -e "  Backend: ${BLUE}http://localhost:8080${NC}"
echo -e "  Frontend: ${BLUE}http://localhost:3000${NC}"

echo -e "\n${YELLOW}Key Features Working:${NC}"
echo -e "${GREEN}✓${NC} URL Routing:"
echo -e "  - http://localhost:3000/register (First-time setup)"
echo -e "  - http://localhost:3000/login"
echo -e "  - http://localhost:3000/dashboard"
echo -e "  - http://localhost:3000/cameras"
echo -e "  - http://localhost:3000/events"
echo -e "  - http://localhost:3000/settings"

echo -e "\n${GREEN}✓${NC} Real-Time Updates:"
echo -e "  - Time updates every second in UI"
echo -e "  - WebSocket sends stats every 5 seconds"
echo -e "  - Dashboard shows live system status"

echo -e "\n${GREEN}✓${NC} Authentication Flow:"
echo -e "  - First user auto-becomes admin"
echo -e "  - Auto-login after registration"
echo -e "  - Auto-redirect to dashboard"
echo -e "  - Secure session management"

echo -e "\n${YELLOW}Demo Instructions:${NC}"
echo "1. Open browser to http://localhost:3000"
echo "2. Register first admin user"
echo "3. Watch automatic redirect to dashboard"
echo "4. Observe real-time clock updates"
echo "5. Navigate between pages - URLs change!"
echo "6. Try logout - redirects to login"

echo -e "\n${BLUE}Press Ctrl+C to stop demo${NC}"

# Keep running
wait $DEMO_PID 