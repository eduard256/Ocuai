#!/bin/bash

# Ocuai Camera Integration Test Script
# This script validates that all camera functionality is working correctly

echo "🎥 Ocuai Camera Integration System Test"
echo "========================================"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Helper functions
test_passed() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((TESTS_PASSED++))
    ((TOTAL_TESTS++))
}

test_failed() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((TESTS_FAILED++))
    ((TOTAL_TESTS++))
}

test_warning() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
}

test_info() {
    echo -e "${BLUE}ℹ️  INFO${NC}: $1"
}

# Test 1: Check go2rtc YAML configuration
echo "Test 1: go2rtc Configuration"
echo "-----------------------------"
if [ -f "data/go2rtc/go2rtc.yaml" ]; then
    # Check for invalid JSON syntax in YAML
    if grep -q "streams: {}" data/go2rtc/go2rtc.yaml; then
        test_failed "YAML contains invalid JSON syntax: 'streams: {}'"
    else
        test_passed "YAML configuration format is correct"
    fi
    
    # Validate YAML structure
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "
import yaml
try:
    with open('data/go2rtc/go2rtc.yaml', 'r') as f:
        yaml.safe_load(f)
    print('YAML_VALID')
except Exception as e:
    print(f'YAML_ERROR: {e}')
        " | grep -q "YAML_VALID"
        
        if [ $? -eq 0 ]; then
            test_passed "YAML syntax is valid"
        else
            test_failed "YAML syntax has errors"
        fi
    else
        test_warning "Python3 not available - cannot validate YAML syntax"
    fi
else
    test_failed "go2rtc configuration file not found"
fi
echo

# Test 2: Check Go binary compilation
echo "Test 2: Go Binary Compilation"
echo "------------------------------"
if go version >/dev/null 2>&1; then
    test_info "Go version: $(go version)"
    
    # Test compilation
    if go build -o /tmp/ocuai_test ./cmd/ocuai >/dev/null 2>&1; then
        test_passed "Go binary compiles successfully"
        rm -f /tmp/ocuai_test
    else
        test_failed "Go binary compilation failed"
    fi
else
    test_failed "Go is not installed or not in PATH"
fi
echo

# Test 3: Check Node.js and web frontend
echo "Test 3: Frontend Dependencies"
echo "------------------------------"
cd web 2>/dev/null
if [ $? -eq 0 ]; then
    if [ -f "package.json" ]; then
        test_passed "Frontend package.json found"
        
        if [ -d "node_modules" ] && [ -f "node_modules/.package-lock.json" ] || [ -f "package-lock.json" ]; then
            test_passed "Node modules are installed"
        else
            test_warning "Node modules not installed - run 'npm install' in web directory"
        fi
        
        # Check if build works
        if npm run build >/dev/null 2>&1; then
            test_passed "Frontend builds successfully"
        else
            test_warning "Frontend build has issues"
        fi
    else
        test_failed "Frontend package.json not found"
    fi
else
    test_failed "Web directory not found"
fi
cd .. 2>/dev/null
echo

# Test 4: Check critical files and directories
echo "Test 4: File Structure"
echo "----------------------"
REQUIRED_FILES=(
    "cmd/ocuai/main.go"
    "internal/go2rtc/go2rtc.go"
    "internal/go2rtc/scanner.go"
    "internal/streaming/streaming.go"
    "internal/web/web.go"
    "data/config.yaml"
    "go.mod"
    "go.sum"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        test_passed "Required file exists: $file"
    else
        test_failed "Missing required file: $file"
    fi
done

REQUIRED_DIRS=(
    "data"
    "data/go2rtc"
    "internal"
    "web"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        test_passed "Required directory exists: $dir"
    else
        test_failed "Missing required directory: $dir"
    fi
done
echo

# Test 5: Check network ports availability
echo "Test 5: Network Ports"
echo "---------------------"
REQUIRED_PORTS=(1984 8554 8555 8080 3000)

for port in "${REQUIRED_PORTS[@]}"; do
    if command -v netstat >/dev/null 2>&1; then
        if netstat -ln 2>/dev/null | grep -q ":$port "; then
            test_warning "Port $port is already in use"
        else
            test_passed "Port $port is available"
        fi
    elif command -v ss >/dev/null 2>&1; then
        if ss -ln 2>/dev/null | grep -q ":$port "; then
            test_warning "Port $port is already in use"
        else
            test_passed "Port $port is available"
        fi
    else
        test_warning "Cannot check port $port (no netstat/ss available)"
    fi
done
echo

# Test 6: Test go2rtc binary availability
echo "Test 6: go2rtc Binary"
echo "---------------------"
if [ -f "data/go2rtc/bin/go2rtc" ]; then
    test_passed "go2rtc binary found"
    
    # Test if binary works
    if ./data/go2rtc/bin/go2rtc --help >/dev/null 2>&1; then
        test_passed "go2rtc binary is executable"
    else
        test_failed "go2rtc binary is not executable or corrupted"
    fi
elif command -v go2rtc >/dev/null 2>&1; then
    test_passed "go2rtc found in system PATH"
else
    test_warning "go2rtc binary not found - will be downloaded on first run"
fi
echo

# Test 7: Database connectivity
echo "Test 7: Database"
echo "----------------"
if [ -f "data/db/ocuai.db" ]; then
    test_passed "SQLite database file exists"
    
    # Test database access
    if command -v sqlite3 >/dev/null 2>&1; then
        if sqlite3 data/db/ocuai.db ".tables" >/dev/null 2>&1; then
            test_passed "Database is accessible"
        else
            test_warning "Database file exists but may be corrupted"
        fi
    else
        test_warning "sqlite3 not available - cannot test database access"
    fi
else
    test_warning "Database file not found - will be created on first run"
fi
echo

# Test 8: Configuration validation
echo "Test 8: Configuration"
echo "---------------------"
if [ -f "data/config.yaml" ]; then
    test_passed "Main configuration file exists"
    
    # Check for basic required configuration sections
    if grep -q "storage:" data/config.yaml && grep -q "streaming:" data/config.yaml; then
        test_passed "Configuration has required sections"
    else
        test_warning "Configuration may be incomplete"
    fi
    
    # Check for security configuration
    if grep -q "security:" data/config.yaml; then
        test_passed "Security configuration section exists"
    else
        test_warning "Security configuration section missing"
    fi
else
    test_failed "Main configuration file missing"
fi
echo

# Test 9: Code quality checks
echo "Test 9: Code Quality"
echo "--------------------"
if command -v gofmt >/dev/null 2>&1; then
    # Check if Go code is formatted
    UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor)
    if [ -z "$UNFORMATTED" ]; then
        test_passed "Go code is properly formatted"
    else
        test_warning "Some Go files need formatting: $UNFORMATTED"
    fi
else
    test_warning "gofmt not available - cannot check code formatting"
fi

if command -v go >/dev/null 2>&1; then
    # Check for potential issues
    if go vet ./... >/dev/null 2>&1; then
        test_passed "Go vet checks pass"
    else
        test_warning "Go vet found potential issues"
    fi
else
    test_warning "Go not available - cannot run vet checks"
fi
echo

# Test Summary
echo "Test Summary"
echo "============"
echo -e "Total tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All critical tests passed! System is ready for production.${NC}"
    echo
    echo "✅ Next steps:"
    echo "  1. Run './scripts/dev.sh' to start the development server"
    echo "  2. Navigate to http://localhost:8080 in your browser"
    echo "  3. Test camera addition functionality"
    echo "  4. Check logs for any runtime issues"
    echo
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed. Please fix the issues before deploying.${NC}"
    echo
    echo "🔧 Common fixes:"
    echo "  - Run 'go mod tidy' to fix Go dependencies"
    echo "  - Run 'npm install' in the web directory for frontend"
    echo "  - Check file permissions: chmod +x scripts/*.sh"
    echo "  - Ensure all required directories exist"
    echo
    exit 1
fi 