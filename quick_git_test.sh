#!/bin/bash

# Quick Git Integration Test
# Tests Dolt Git integration with small sample data to verify functionality

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ORIGINAL_DIR=$(pwd)
GITHUB_REPO="git@github.com:lapingvino/holywritings-dolt.git"
TEST_DIR="quick-git-test"
DOLT_BIN="${ORIGINAL_DIR}/dolt"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test directory..."
    cd "$ORIGINAL_DIR"
    rm -rf "$TEST_DIR" 2>/dev/null || true
}

# Trap for cleanup on exit
trap cleanup EXIT

# Main test function
main() {
    log_info "Starting Quick Git Integration Test"
    log_info "Creating small sample dataset to verify Git integration works"
    echo

    # Create test directory and initialize Dolt repo
    log_info "Step 1: Creating test Dolt repository..."
    rm -rf "$TEST_DIR" 2>/dev/null || true
    mkdir "$TEST_DIR"
    cd "$TEST_DIR"

    $DOLT_BIN init
    log_success "Dolt repository initialized"

    # Create sample tables with small data
    log_info "Step 2: Creating sample tables with data..."

    # Create employees table
    $DOLT_BIN sql -q "CREATE TABLE employees (
        id INT PRIMARY KEY,
        name VARCHAR(100),
        department VARCHAR(50),
        salary INT,
        hire_date DATE
    );"

    # Create departments table
    $DOLT_BIN sql -q "CREATE TABLE departments (
        id INT PRIMARY KEY,
        name VARCHAR(50),
        budget INT,
        manager VARCHAR(100)
    );"

    # Insert sample data
    $DOLT_BIN sql -q "INSERT INTO employees VALUES
        (1, 'Alice Smith', 'Engineering', 95000, '2023-01-15'),
        (2, 'Bob Jones', 'Marketing', 70000, '2023-03-20'),
        (3, 'Carol White', 'Sales', 65000, '2023-02-10'),
        (4, 'David Brown', 'Engineering', 88000, '2023-01-25'),
        (5, 'Eve Wilson', 'HR', 75000, '2023-04-01');"

    $DOLT_BIN sql -q "INSERT INTO departments VALUES
        (1, 'Engineering', 500000, 'Alice Smith'),
        (2, 'Marketing', 200000, 'Bob Jones'),
        (3, 'Sales', 300000, 'Carol White'),
        (4, 'HR', 150000, 'Eve Wilson');"

    log_success "Sample data created"

    # Commit the data to Dolt
    log_info "Step 3: Committing data to Dolt..."
    $DOLT_BIN add .
    $DOLT_BIN commit -m "Add sample employee and department data"
    log_success "Data committed to Dolt"

    # Show what we have
    log_info "Step 4: Verifying data..."
    echo "Employees table:"
    $DOLT_BIN sql -q "SELECT * FROM employees;"
    echo
    echo "Departments table:"
    $DOLT_BIN sql -q "SELECT * FROM departments;"
    echo

    # Test Git integration commands
    log_info "Step 5: Testing Git integration commands..."

    # Test git status
    log_info "Testing 'dolt git status'..."
    $DOLT_BIN git status
    log_success "Git status working"

    # Test git add
    log_info "Testing 'dolt git add'..."
    $DOLT_BIN git add .
    log_success "Git add working"

    # Test git status after add
    log_info "Git status after add:"
    $DOLT_BIN git status

    # Test git commit
    log_info "Testing 'dolt git commit'..."
    $DOLT_BIN git commit -m "Quick test: Export sample employee and department data

This is a quick test of Dolt Git integration with small sample data:
- 5 employees across 4 departments
- Tests chunking, CSV export, and Git workflow
- Validates core Git integration functionality
- Data includes: employees and departments tables"

    log_success "Git commit working"

    # Test dry-run push
    log_info "Step 6: Testing dry-run push..."
    $DOLT_BIN git push --dry-run --verbose "$GITHUB_REPO" main
    log_success "Dry-run push completed"

    # Test with different chunk sizes
    log_info "Step 7: Testing different chunk sizes..."
    log_info "Testing 25MB chunks (dry-run):"
    $DOLT_BIN git push --chunk-size=25MB --dry-run --verbose "$GITHUB_REPO" main

    log_info "Testing 100MB chunks (dry-run):"
    $DOLT_BIN git push --chunk-size=100MB --dry-run --verbose "$GITHUB_REPO" main
    log_success "Chunk size testing completed"

    # Actual push to GitHub
    log_info "Step 8: Pushing to GitHub..."
    log_info "Proceeding with actual push to GitHub repository..."
    $DOLT_BIN git push --verbose "$GITHUB_REPO" main
    log_success "Successfully pushed to GitHub!"

    # Test git log
    log_info "Step 9: Testing git log..."
    $DOLT_BIN git log --oneline -n 5
    log_success "Git log working"

    # Final summary
    echo
    log_success "Quick Git Integration Test completed successfully!"
    echo
    log_info "Summary of tested functionality:"
    log_info "✓ Dolt repository initialization"
    log_info "✓ Sample data creation (2 tables, 9 total rows)"
    log_info "✓ Git workflow: status, add, commit, log"
    log_info "✓ Chunking with different sizes (dry-run)"
    log_info "✓ Actual push to GitHub repository"
    echo
    log_info "Check the results at: https://github.com/lapingvino/holywritings-dolt"
    log_info "The repository should now contain:"
    log_info "  - .dolt-metadata/ with repository info"
    log_info "  - data/ directory with employees.csv and departments.csv"
    log_info "  - README.md with human-readable information"
    echo
}

# Run the test
main "$@"
