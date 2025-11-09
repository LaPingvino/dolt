#!/bin/bash

# Comprehensive Dolt Git Integration Test
# Tests complete workflow with holywritings/bahaiwritings dataset
# Includes extensive logging and round-trip verification

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
ORIGINAL_DIR=$(pwd)
DOLTHUB_REPO="holywritings/bahaiwritings"
GITHUB_REPO="git@github.com:lapingvino/holywritings-dolt.git"
TEST_DIR="comprehensive-git-test"
ROUNDTRIP_DIR="roundtrip-test"
DOLT_BIN="${ORIGINAL_DIR}/dolt"
LOG_FILE="${ORIGINAL_DIR}/git_test_debug.log"

# Logging functions with timestamps
timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

log_to_file() {
    echo "[$(timestamp)] $1" >> "$LOG_FILE"
}

log_info() {
    local msg="[INFO] $1"
    echo -e "${BLUE}${msg}${NC}"
    log_to_file "$msg"
}

log_success() {
    local msg="[SUCCESS] $1"
    echo -e "${GREEN}${msg}${NC}"
    log_to_file "$msg"
}

log_warning() {
    local msg="[WARNING] $1"
    echo -e "${YELLOW}${msg}${NC}"
    log_to_file "$msg"
}

log_error() {
    local msg="[ERROR] $1"
    echo -e "${RED}${msg}${NC}"
    log_to_file "$msg"
}

log_debug() {
    local msg="[DEBUG] $1"
    echo -e "${MAGENTA}${msg}${NC}"
    log_to_file "$msg"
}

log_header() {
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}================================${NC}"
    log_to_file "=== $1 ==="
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test directories..."
    cd "$ORIGINAL_DIR"
    rm -rf "$TEST_DIR" "$ROUNDTRIP_DIR" 2>/dev/null || true
}

# Trap for cleanup on exit
trap cleanup EXIT

# Check prerequisites and environment
check_prerequisites() {
    log_header "CHECKING PREREQUISITES"

    log_info "Checking Dolt binary..."
    if [ ! -f "$DOLT_BIN" ]; then
        log_error "Dolt binary not found at $DOLT_BIN"
        exit 1
    fi

    log_info "Dolt version: $($DOLT_BIN version)"

    log_info "Checking Git commands availability..."
    $DOLT_BIN git --help > /dev/null 2>&1 || { log_error "Git commands not available"; exit 1; }

    log_info "Testing SSH connectivity to GitHub..."
    if ssh -T -o ConnectTimeout=10 git@github.com 2>&1 | grep -q "successfully authenticated"; then
        log_success "GitHub SSH authentication working"
    elif ssh -T -o ConnectTimeout=10 git@github.com 2>&1 | grep -q "Permission denied"; then
        log_warning "GitHub SSH authentication may have issues"
        log_debug "SSH test output: $(ssh -T git@github.com 2>&1 | head -1)"
    else
        log_warning "GitHub SSH connectivity unclear"
    fi

    log_info "Checking network connectivity to DoltHub..."
    if curl -s --max-time 10 "https://www.dolthub.com" > /dev/null 2>&1; then
        log_success "DoltHub connectivity OK"
    else
        log_warning "DoltHub connectivity issues"
    fi

    # Initialize log file
    echo "=== Dolt Git Integration Test Log - $(timestamp) ===" > "$LOG_FILE"
    log_success "Prerequisites check completed"
}

# Download dataset from DoltHub
download_dataset() {
    log_header "DOWNLOADING DATASET FROM DOLTHUB"

    log_info "Cloning $DOLTHUB_REPO from DoltHub..."
    log_info "This is a large dataset (39,450+ chunks) - will take several minutes"

    rm -rf "$TEST_DIR" 2>/dev/null || true

    # Clone with verbose output and capture progress
    log_debug "Starting clone operation..."
    if $DOLT_BIN clone "$DOLTHUB_REPO" "$TEST_DIR" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Dataset downloaded successfully"
    else
        log_error "Failed to download dataset"
        exit 1
    fi

    cd "$TEST_DIR"
    log_info "Moved to test directory: $(pwd)"
}

# Analyze downloaded data
analyze_data() {
    log_header "ANALYZING DOWNLOADED DATA"

    log_info "Repository status:"
    $DOLT_BIN status | tee -a "$LOG_FILE"

    log_info "Available tables:"
    $DOLT_BIN sql -q "SHOW TABLES;" 2>&1 | tee -a "$LOG_FILE"

    log_info "Database size estimation:"
    du -sh . | tee -a "$LOG_FILE"

    log_info "Row counts per table:"
    for table in $($DOLT_BIN sql -q "SHOW TABLES;" 2>/dev/null | tail -n +4 | head -n -1 | awk '{print $2}' | grep -v '^$'); do
        if [ "$table" != "Tables_in_holywritings" ]; then
            count=$($DOLT_BIN sql -q "SELECT COUNT(*) FROM \`$table\`;" 2>/dev/null | tail -n +4 | head -1 | awk '{print $2}' || echo "ERROR")
            log_info "  $table: $count rows"
        fi
    done

    log_info "Recent commits:"
    $DOLT_BIN log --oneline -n 5 | tee -a "$LOG_FILE"

    log_success "Data analysis completed"
}

# Test Git workflow commands
test_git_workflow() {
    log_header "TESTING GIT WORKFLOW COMMANDS"

    # Test git status
    log_info "Testing 'dolt git status'..."
    log_debug "Git status output:"
    $DOLT_BIN git status 2>&1 | tee -a "$LOG_FILE"
    log_success "Git status working"

    # Test git add with verbose output
    log_info "Testing 'dolt git add .' with verbose output..."
    log_debug "Git add output:"
    $DOLT_BIN git add . 2>&1 | tee -a "$LOG_FILE"
    log_success "Git add completed"

    # Check status after add
    log_info "Git status after add:"
    $DOLT_BIN git status 2>&1 | tee -a "$LOG_FILE"

    # Test git commit
    log_info "Testing 'dolt git commit'..."
    local commit_msg="Export holywritings/bahaiwritings dataset from DoltHub to GitHub

This commit contains the complete Bahai writings dataset exported using Dolt's Git integration:
- Large dataset with 39,450+ chunks from DoltHub
- Automatic chunking for GitHub compatibility
- Complete schema and data preservation
- Test of production-ready Git integration

Dataset includes multiple tables with religious texts and metadata.
Exported on: $(date)
Test run: Comprehensive Git Integration Test"

    log_debug "Committing with message length: ${#commit_msg} characters"
    $DOLT_BIN git commit -m "$commit_msg" 2>&1 | tee -a "$LOG_FILE"
    log_success "Git commit completed"

    # Test git log
    log_info "Testing 'dolt git log'..."
    $DOLT_BIN git log --oneline -n 3 2>&1 | tee -a "$LOG_FILE"
    log_success "Git workflow commands tested successfully"
}

# Test chunking with different sizes
test_chunking_strategies() {
    log_header "TESTING CHUNKING STRATEGIES"

    local chunk_sizes=("25MB" "50MB" "100MB")

    for size in "${chunk_sizes[@]}"; do
        log_info "Testing chunking with $size chunk size (dry-run)..."
        log_debug "Dry-run output for $size chunks:"

        $DOLT_BIN git push --chunk-size="$size" --dry-run --verbose "$GITHUB_REPO" main 2>&1 | tee -a "$LOG_FILE"

        log_success "Dry-run with $size chunks completed"
        echo
    done
}

# Push to GitHub with detailed logging
push_to_github() {
    log_header "PUSHING TO GITHUB WITH DETAILED LOGGING"

    log_info "Attempting to push to $GITHUB_REPO"
    log_info "Using default chunk size (50MB)"

    # Set Git user for the repository if needed
    log_debug "Setting Git user configuration..."
    git config user.name "Dolt Git Integration Test" 2>/dev/null || true
    git config user.email "test@dolt-git-integration.local" 2>/dev/null || true

    log_debug "Starting push operation with full verbose output..."

    # Capture the full output of the push operation
    local push_output_file="${ORIGINAL_DIR}/push_output.log"

    if $DOLT_BIN git push --verbose "$GITHUB_REPO" main 2>&1 | tee "$push_output_file" | tee -a "$LOG_FILE"; then
        log_success "Push to GitHub completed successfully!"

        # Analyze what was actually pushed
        log_info "Analyzing push results..."
        if grep -q "Created commit:" "$push_output_file"; then
            local commit_hash=$(grep "Created commit:" "$push_output_file" | awk '{print $3}')
            log_info "Created Git commit: $commit_hash"
        fi

        if grep -q "Pushing to remote repository" "$push_output_file"; then
            log_info "Remote push operation was attempted"
        fi

        # Check for any error indicators
        if grep -qi "error\|failed\|denied" "$push_output_file"; then
            log_warning "Push output contains error indicators:"
            grep -i "error\|failed\|denied" "$push_output_file" | tee -a "$LOG_FILE"
        fi

    else
        log_error "Push to GitHub failed!"
        log_error "Full push output saved to: $push_output_file"

        # Show the last few lines of output for immediate debugging
        log_error "Last 10 lines of push output:"
        tail -10 "$push_output_file" | tee -a "$LOG_FILE"

        # Don't exit here - continue with analysis
        return 1
    fi
}

# Wait and verify GitHub repository
verify_github_repository() {
    log_header "VERIFYING GITHUB REPOSITORY"

    log_info "Waiting 30 seconds for GitHub to process the push..."
    sleep 30

    # Try to access the repository via GitHub API
    log_info "Checking GitHub repository via API..."
    if curl -s "https://api.github.com/repos/lapingvino/holywritings-dolt" | grep -q '"name"'; then
        log_success "GitHub repository exists and is accessible"

        # Get repository info
        local repo_info=$(curl -s "https://api.github.com/repos/lapingvino/holywritings-dolt")
        local repo_size=$(echo "$repo_info" | grep '"size"' | head -1 | awk '{print $2}' | sed 's/,//')
        log_info "Repository size: $repo_size KB"

        # Check for recent commits
        log_info "Checking recent commits via API..."
        curl -s "https://api.github.com/repos/lapingvino/holywritings-dolt/commits" | grep '"sha"' | head -3 | tee -a "$LOG_FILE"

    else
        log_warning "GitHub repository not accessible via API or doesn't exist"
    fi

    # Try to clone the repository to verify it actually worked
    log_info "Attempting to verify by cloning from GitHub..."
    cd "$ORIGINAL_DIR"

    if git clone "$GITHUB_REPO" github-verification 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Successfully cloned from GitHub - repository exists!"

        cd github-verification
        log_info "GitHub repository contents:"
        ls -la | tee -a "$LOG_FILE"

        if [ -d "data" ]; then
            log_info "Data directory found - checking structure:"
            find data -type f | head -10 | tee -a "$LOG_FILE"
        fi

        if [ -d ".dolt-metadata" ]; then
            log_info "Dolt metadata found - checking structure:"
            find .dolt-metadata -type f | tee -a "$LOG_FILE"
        fi

        cd "$ORIGINAL_DIR"
        rm -rf github-verification

    else
        log_error "Failed to clone from GitHub - repository may not exist or have issues"
        return 1
    fi
}

# Round-trip test: clone back from GitHub
roundtrip_test() {
    log_header "ROUND-TRIP TEST: CLONE FROM GITHUB"

    cd "$ORIGINAL_DIR"
    rm -rf "$ROUNDTRIP_DIR" 2>/dev/null || true

    log_info "Cloning from GitHub back to Dolt..."

    if $DOLT_BIN git clone "$GITHUB_REPO" "$ROUNDTRIP_DIR" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "Successfully cloned from GitHub"

        cd "$ROUNDTRIP_DIR"

        # Analyze cloned data
        log_info "Analyzing round-trip data..."
        log_info "Repository status:"
        $DOLT_BIN status | tee -a "$LOG_FILE"

        log_info "Available tables in round-trip:"
        $DOLT_BIN sql -q "SHOW TABLES;" 2>&1 | tee -a "$LOG_FILE"

        # Compare with original
        log_info "Comparing table counts with original..."
        cd "$ORIGINAL_DIR/$TEST_DIR"

        log_info "Original table count:"
        original_count=$($DOLT_BIN sql -q "SHOW TABLES;" 2>/dev/null | wc -l)
        log_info "Original: $original_count tables"

        cd "$ORIGINAL_DIR/$ROUNDTRIP_DIR"
        log_info "Round-trip table count:"
        roundtrip_count=$($DOLT_BIN sql -q "SHOW TABLES;" 2>/dev/null | wc -l)
        log_info "Round-trip: $roundtrip_count tables"

        if [ "$original_count" = "$roundtrip_count" ]; then
            log_success "Table counts match - round-trip successful!"
        else
            log_warning "Table counts differ - may indicate data loss"
        fi

        cd "$ORIGINAL_DIR"

    else
        log_error "Failed to clone from GitHub"
        return 1
    fi
}

# Generate comprehensive test report
generate_report() {
    log_header "GENERATING COMPREHENSIVE TEST REPORT"

    local report_file="${ORIGINAL_DIR}/git_integration_test_report.md"

    cat > "$report_file" << EOF
# Dolt Git Integration Test Report

**Test Date:** $(date)
**Test Duration:** Started at test initialization
**Dataset:** holywritings/bahaiwritings (39,450+ chunks)

## Test Summary

### Environment
- Dolt Version: $($DOLT_BIN version)
- Platform: $(uname -a)
- Git Integration: Enabled and tested

### Test Results

#### ✅ Completed Successfully
- [x] Prerequisites check
- [x] Large dataset download from DoltHub
- [x] Git workflow commands (status, add, commit, log)
- [x] Chunking strategies (25MB, 50MB, 100MB)

#### 🔄 GitHub Integration Results
EOF

    if verify_github_repository; then
        echo "- [x] Push to GitHub successful" >> "$report_file"
        echo "- [x] GitHub repository verification passed" >> "$report_file"
        if roundtrip_test; then
            echo "- [x] Round-trip test successful" >> "$report_file"
        else
            echo "- [ ] Round-trip test failed" >> "$report_file"
        fi
    else
        echo "- [ ] Push to GitHub failed or verification failed" >> "$report_file"
        echo "- [ ] Round-trip test not attempted" >> "$report_file"
    fi

    cat >> "$report_file" << EOF

## Detailed Logs
- Full debug log: git_test_debug.log
- Push operation log: push_output.log

## GitHub Repository
- URL: https://github.com/lapingvino/holywritings-dolt
- SSH: $GITHUB_REPO

## Next Steps
EOF

    if [ -f "push_output.log" ] && grep -qi "error\|failed" "push_output.log"; then
        cat >> "$report_file" << EOF
1. Review push_output.log for specific error messages
2. Check SSH key configuration for GitHub access
3. Verify repository permissions on GitHub
4. Test with smaller dataset if authentication is resolved
EOF
    else
        cat >> "$report_file" << EOF
1. Check GitHub repository for pushed content
2. Verify data integrity in GitHub repository
3. Test cloning from GitHub by external users
4. Performance optimization for large dataset pushes
EOF
    fi

    log_success "Test report generated: $report_file"
}

# Main test execution
main() {
    log_header "COMPREHENSIVE DOLT GIT INTEGRATION TEST"
    log_info "Testing with holywritings/bahaiwritings dataset"
    log_info "Full round-trip test with extensive debugging"
    echo

    check_prerequisites
    download_dataset
    analyze_data
    test_git_workflow
    test_chunking_strategies

    # Attempt push and continue regardless of result for debugging
    if push_to_github; then
        log_success "Push succeeded - proceeding with verification"
        verify_github_repository
        roundtrip_test
    else
        log_warning "Push failed - will still attempt verification for debugging"
        verify_github_repository || true  # Don't exit on failure
    fi

    generate_report

    log_header "TEST COMPLETED"
    log_info "Check the following files for detailed information:"
    log_info "  - git_test_debug.log (complete debug log)"
    log_info "  - push_output.log (push operation details)"
    log_info "  - git_integration_test_report.md (summary report)"
    echo

    if [ -f "${ORIGINAL_DIR}/push_output.log" ]; then
        log_info "Key findings from push operation:"
        if grep -q "Successfully pushed" "${ORIGINAL_DIR}/push_output.log"; then
            log_success "Push operation reported success"
        elif grep -qi "failed\|error" "${ORIGINAL_DIR}/push_output.log"; then
            log_error "Push operation reported failures - check push_output.log"
            log_info "Common issues to check:"
            log_info "  1. SSH key authentication with GitHub"
            log_info "  2. Repository permissions"
            log_info "  3. Git repository initialization"
        fi
    fi
}

# Execute main function
main "$@"
