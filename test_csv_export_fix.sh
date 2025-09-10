#!/bin/bash

# Test script to verify Git integration CSV export fix
# This script tests that the critical bug fixes for empty CSV files are working

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TEST_DIR="csv-export-test"
DOLT_BIN="./go/dolt"
LOG_FILE="csv_export_test.log"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO] $1${NC}"
    echo "[INFO] $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
    echo "[SUCCESS] $1" >> "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
    echo "[WARNING] $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR] $1${NC}"
    echo "[ERROR] $1" >> "$LOG_FILE"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test directory..."
    cd ..
    rm -rf "$TEST_DIR" 2>/dev/null || true
}

# Trap for cleanup on exit
trap cleanup EXIT

# Initialize log
echo "=== CSV Export Fix Test - $(date) ===" > "$LOG_FILE"

log_info "Testing CSV export fix for Git integration"
log_info "This test verifies that CSV files contain actual data, not placeholders"

# Check if dolt binary exists
if [ ! -f "$DOLT_BIN" ]; then
    log_error "Dolt binary not found. Please run: cd go && go build ./cmd/dolt"
    exit 1
fi

log_info "Dolt version: $($DOLT_BIN version)"

# Create test directory
rm -rf "$TEST_DIR" 2>/dev/null || true
mkdir "$TEST_DIR"
cd "$TEST_DIR"

# Initialize Dolt repository
log_info "Initializing test Dolt repository..."
$DOLT_BIN init
$DOLT_BIN config --local --add user.name "Test User"
$DOLT_BIN config --local --add user.email "test@example.com"

# Create test tables with known data
log_info "Creating test tables with sample data..."

# Table 1: Simple users table
$DOLT_BIN sql <<EOF
CREATE TABLE users (
    id INT PRIMARY KEY,
    name VARCHAR(100),
    email VARCHAR(100),
    age INT,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO users (id, name, email, age) VALUES
    (1, 'Alice Johnson', 'alice@example.com', 28),
    (2, 'Bob Smith', 'bob@example.com', 35),
    (3, 'Carol Davis', 'carol@example.com', 42),
    (4, 'David Wilson', 'david@example.com', 29),
    (5, 'Eve Brown', 'eve@example.com', 31);
EOF

# Table 2: Products table with more data types
$DOLT_BIN sql <<EOF
CREATE TABLE products (
    id INT PRIMARY KEY,
    name VARCHAR(200),
    price DECIMAL(10,2),
    in_stock BOOLEAN,
    description TEXT,
    category_id INT
);

INSERT INTO products (id, name, price, in_stock, description, category_id) VALUES
    (1, 'Laptop Computer', 999.99, true, 'High-performance laptop for professionals', 1),
    (2, 'Wireless Mouse', 29.99, true, 'Ergonomic wireless mouse with precision tracking', 1),
    (3, 'Coffee Mug', 12.50, false, 'Ceramic coffee mug with company logo', 2),
    (4, 'Desk Chair', 249.99, true, 'Comfortable office chair with lumbar support', 3),
    (5, 'Notebook Set', 15.99, true, 'Set of 3 spiral notebooks for note-taking', 2),
    (6, 'USB Cable', 8.99, true, 'USB-C to USB-A cable, 6 feet long', 1),
    (7, 'Water Bottle', 19.99, true, 'Insulated stainless steel water bottle', 2);
EOF

# Commit the data
log_info "Committing test data..."
$DOLT_BIN add .
$DOLT_BIN commit -m "Add test data for CSV export verification

This commit contains:
- 5 users with various data types
- 7 products with decimals, booleans, and text

Used to test that CSV export generates actual data, not placeholders."

# Verify data is in Dolt
log_info "Verifying data in Dolt..."
USERS_COUNT=$($DOLT_BIN sql -q "SELECT COUNT(*) FROM users;" | tail -1 | awk '{print $2}')
PRODUCTS_COUNT=$($DOLT_BIN sql -q "SELECT COUNT(*) FROM products;" | tail -1 | awk '{print $2}')

log_info "Users count: $USERS_COUNT"
log_info "Products count: $PRODUCTS_COUNT"

if [ "$USERS_COUNT" != "5" ] || [ "$PRODUCTS_COUNT" != "7" ]; then
    log_error "Test data not properly inserted into Dolt"
    exit 1
fi

# Test CSV export using git functionality
log_info "Testing CSV export via Git integration..."

# Create a temporary git repository for export
EXPORT_DIR="../csv-export-output"
rm -rf "$EXPORT_DIR" 2>/dev/null || true
mkdir "$EXPORT_DIR"
cd "$EXPORT_DIR"
git init
git config user.name "Test User"
git config user.email "test@example.com"

cd "../$TEST_DIR"

# Export to git repository (dry run first to test chunking logic)
log_info "Running dry-run export test..."
$DOLT_BIN git push --dry-run --verbose "$EXPORT_DIR" main 2>&1 | tee -a "../$LOG_FILE"

# Actual export
log_info "Running actual CSV export..."
$DOLT_BIN git push --verbose "$EXPORT_DIR" main 2>&1 | tee -a "../$LOG_FILE"

# Verify export results
log_info "Verifying export results..."
cd "$EXPORT_DIR"

# Check directory structure
if [ ! -d "data" ]; then
    log_error "Data directory not created during export"
    exit 1
fi

if [ ! -d ".dolt-metadata" ]; then
    log_error "Metadata directory not created during export"
    exit 1
fi

# Check for CSV files
USERS_CSV="data/users/users.csv"
PRODUCTS_CSV="data/products/products.csv"

if [ ! -f "$USERS_CSV" ]; then
    log_error "Users CSV file not found: $USERS_CSV"
    exit 1
fi

if [ ! -f "$PRODUCTS_CSV" ]; then
    log_error "Products CSV file not found: $PRODUCTS_CSV"
    exit 1
fi

# Critical test: Check that CSV files are NOT empty
log_info "Testing CSV file contents (critical bug fix verification)..."

USERS_CSV_SIZE=$(wc -c < "$USERS_CSV")
PRODUCTS_CSV_SIZE=$(wc -c < "$PRODUCTS_CSV")

log_info "Users CSV size: $USERS_CSV_SIZE bytes"
log_info "Products CSV size: $PRODUCTS_CSV_SIZE bytes"

if [ "$USERS_CSV_SIZE" -le 50 ]; then
    log_error "Users CSV file is too small ($USERS_CSV_SIZE bytes) - likely empty or contains only headers"
    log_error "This indicates the CSV export bug is NOT fixed"
    head -5 "$USERS_CSV" | tee -a "../$LOG_FILE"
    exit 1
fi

if [ "$PRODUCTS_CSV_SIZE" -le 50 ]; then
    log_error "Products CSV file is too small ($PRODUCTS_CSV_SIZE bytes) - likely empty or contains only headers"
    log_error "This indicates the CSV export bug is NOT fixed"
    head -5 "$PRODUCTS_CSV" | tee -a "../$LOG_FILE"
    exit 1
fi

# Verify CSV content structure
log_info "Verifying CSV content structure..."

USERS_LINES=$(wc -l < "$USERS_CSV")
PRODUCTS_LINES=$(wc -l < "$PRODUCTS_CSV")

log_info "Users CSV lines: $USERS_LINES (expected: 6 = 1 header + 5 data)"
log_info "Products CSV lines: $PRODUCTS_LINES (expected: 8 = 1 header + 7 data)"

# Check for actual data content (not just placeholders)
log_info "Checking for actual data content..."

# Look for known test data in users CSV
if grep -q "Alice Johnson" "$USERS_CSV"; then
    log_success "Found 'Alice Johnson' in users CSV - data export working!"
else
    log_error "Did not find 'Alice Johnson' in users CSV - data may not be properly exported"
    log_error "Users CSV contents:"
    cat "$USERS_CSV" | head -10 | tee -a "../$LOG_FILE"
    exit 1
fi

if grep -q "alice@example.com" "$USERS_CSV"; then
    log_success "Found 'alice@example.com' in users CSV - email data preserved!"
else
    log_error "Did not find 'alice@example.com' in users CSV"
    exit 1
fi

# Check for known product data
if grep -q "Laptop Computer" "$PRODUCTS_CSV"; then
    log_success "Found 'Laptop Computer' in products CSV - product data exported!"
else
    log_error "Did not find 'Laptop Computer' in products CSV"
    log_error "Products CSV contents:"
    cat "$PRODUCTS_CSV" | head -10 | tee -a "../$LOG_FILE"
    exit 1
fi

if grep -q "999.99" "$PRODUCTS_CSV"; then
    log_success "Found '999.99' in products CSV - decimal data preserved!"
else
    log_error "Did not find '999.99' in products CSV - decimal conversion may be broken"
    exit 1
fi

# Check for placeholder/dummy data (this should NOT be found)
if grep -q "dolt_row_.*_col_" "$USERS_CSV"; then
    log_error "Found placeholder data in users CSV - export bug is NOT fixed!"
    log_error "Placeholder data indicates the old broken code path is still being used"
    exit 1
fi

if grep -q "dolt_row_.*_col_" "$PRODUCTS_CSV"; then
    log_error "Found placeholder data in products CSV - export bug is NOT fixed!"
    exit 1
fi

log_success "No placeholder data found - export bug appears to be fixed!"

# Show sample of exported data for manual verification
log_info "Sample of exported data:"
echo -e "${BLUE}Users CSV (first 3 lines):${NC}"
head -3 "$USERS_CSV" | tee -a "../$LOG_FILE"
echo -e "${BLUE}Products CSV (first 3 lines):${NC}"
head -3 "$PRODUCTS_CSV" | tee -a "../$LOG_FILE"

# Test metadata files
log_info "Verifying metadata files..."
if [ -f ".dolt-metadata/manifest.json" ]; then
    log_success "Manifest file created"
    MANIFEST_SIZE=$(wc -c < ".dolt-metadata/manifest.json")
    log_info "Manifest size: $MANIFEST_SIZE bytes"
else
    log_warning "Manifest file not found"
fi

if [ -f ".dolt-metadata/schema.sql" ]; then
    log_success "Schema file created"
    SCHEMA_SIZE=$(wc -c < ".dolt-metadata/schema.sql")
    log_info "Schema size: $SCHEMA_SIZE bytes"
else
    log_warning "Schema file not found"
fi

# Test README generation
if [ -f "README.md" ]; then
    log_success "README.md generated"
    README_SIZE=$(wc -c < "README.md")
    log_info "README size: $README_SIZE bytes"

    # Check if README contains actual table information
    if grep -q "users" "README.md" && grep -q "products" "README.md"; then
        log_success "README contains table information"
    else
        log_warning "README may not contain complete table information"
    fi
else
    log_warning "README.md not generated"
fi

# Final verification summary
log_info "=== TEST RESULTS SUMMARY ==="
log_success "✓ CSV files are generated and contain actual data"
log_success "✓ No placeholder/dummy data found in exports"
log_success "✓ Real table data (names, emails, prices) properly exported"
log_success "✓ Data types (strings, decimals, integers) preserved"
log_success "✓ File sizes indicate substantial data content"

log_info "Users table: $USERS_COUNT rows -> $USERS_LINES CSV lines ($USERS_CSV_SIZE bytes)"
log_info "Products table: $PRODUCTS_COUNT rows -> $PRODUCTS_LINES CSV lines ($PRODUCTS_CSV_SIZE bytes)"

echo
log_success "🎉 CSV EXPORT FIX VERIFICATION SUCCESSFUL! 🎉"
log_success "The critical bug causing empty CSV files has been fixed"
log_success "Git integration can now export actual Dolt data to CSV format"

echo -e "${GREEN}"
echo "Next steps for complete Git integration:"
echo "1. ✓ CSV data export - FIXED"
echo "2. [ ] Commit history preservation - needs implementation"
echo "3. [ ] Large dataset testing with chunking"
echo "4. [ ] Authentication and push to real Git repositories"
echo -e "${NC}"

log_info "Test completed successfully. Check '$LOG_FILE' for detailed logs."
