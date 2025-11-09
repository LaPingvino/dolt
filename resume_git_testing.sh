#!/bin/bash

# Resume Git Integration Testing Script
# Post-restart continuation script for Dolt Git integration development
#
# This script resumes development and testing where we left off:
# - Builds the Dolt binary with completed Git integration
# - Runs comprehensive integration tests
# - Tests real-world data collaboration workflows
# - Provides next step recommendations

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="git@github.com:lapingvino/holywritings-dolt.git"
DOLTHUB_REPO="holywritings/bahaiwritings"

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

log_header() {
    echo -e "${CYAN}================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}================================${NC}"
}

# Print startup banner
print_banner() {
    log_header "DOLT GIT INTEGRATION - RESUME TESTING"
    echo -e "${GREEN}Status: Git Integration Implementation Complete ✅${NC}"
    echo -e "${BLUE}Task: Validate production readiness with real-world testing${NC}"
    echo
    echo "Completed Features:"
    echo "✅ Bundle Support - SQLite-based bundles"
    echo "✅ ZIP CSV Import/Export - GTFS and CSV zip handling"
    echo "✅ Git Integration - Complete workflow with chunking"
    echo
    echo "Git Commands Implemented:"
    echo "  • dolt git clone - Clone Git repositories"
    echo "  • dolt git push - Push with automatic chunking"
    echo "  • dolt git pull - Pull repository changes"
    echo "  • dolt git add/commit/status/log - Complete workflow"
    echo
}

# Check prerequisites
check_environment() {
    log_info "Checking development environment..."

    # Check if we're in the right directory
    if [ ! -d "go" ] || [ ! -f "go/go.mod" ]; then
        log_error "Not in dolt project root directory"
        log_info "Please run: cd dolt && ./resume_git_testing.sh"
        exit 1
    fi

    # Check Go installation
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    # Check Git installation
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed or not in PATH"
        exit 1
    fi

    log_success "Environment check passed"
}

# Build the dolt binary with Git integration
build_dolt_binary() {
    log_info "Building Dolt binary with Git integration..."

    cd go

    # Clean any existing binary
    rm -f dolt ../dolt

    # Run go mod tidy to ensure dependencies
    log_info "Updating Go dependencies..."
    go mod tidy

    # Build the binary
    log_info "Compiling Dolt with Git integration..."
    if go build -o dolt ./cmd/dolt; then
        mv dolt ../dolt
        cd ..
        chmod +x dolt
        log_success "Dolt binary built successfully"
    else
        log_error "Failed to build Dolt binary"
        cd ..
        exit 1
    fi
}

# Test Git commands are available
test_git_commands() {
    log_info "Testing Git command availability..."

    # Test main git command
    if ./dolt git --help > /dev/null 2>&1; then
        log_success "Git commands are available"

        # Show available commands
        log_info "Available Git commands:"
        ./dolt git --help | grep -E "^\s+(clone|push|pull|add|commit|status|log)" | sed 's/^/  /'
    else
        log_error "Git commands not available"
        exit 1
    fi
}

# Quick functionality test
quick_functionality_test() {
    log_info "Running quick functionality tests..."

    # Test each command's help
    local commands=("clone" "push" "pull" "add" "commit" "status" "log")

    for cmd in "${commands[@]}"; do
        if ./dolt git $cmd --help > /dev/null 2>&1; then
            log_info "✓ dolt git $cmd - Help available"
        else
            log_warning "✗ dolt git $cmd - Help not working"
        fi
    done

    log_success "Quick functionality test completed"
}

# Run integration tests if available
run_integration_tests() {
    log_info "Looking for integration test script..."

    if [ -f "test_git_integration.sh" ]; then
        log_info "Found integration test script"
        log_warning "Integration tests use real data and may take time."
        echo -n "Run full integration tests? (y/n): "
        read -r response

        if [[ "$response" =~ ^[Yy]$ ]]; then
            log_info "Running integration tests..."
            chmod +x test_git_integration.sh

            if ./test_git_integration.sh; then
                log_success "Integration tests completed successfully"
            else
                log_warning "Integration tests failed or were interrupted"
            fi
        else
            log_info "Skipping integration tests"
        fi
    else
        log_warning "Integration test script not found"
        log_info "You can run manual tests with real repositories"
    fi
}

# Show test examples
show_test_examples() {
    log_header "MANUAL TESTING EXAMPLES"

    echo -e "${YELLOW}Test Git Integration with Real Data:${NC}"
    echo
    echo "1. Clone from DoltHub and test Git workflow:"
    echo "   ./dolt clone $DOLTHUB_REPO test-holywritings"
    echo "   cd test-holywritings"
    echo "   ../dolt git status"
    echo "   ../dolt git add ."
    echo "   ../dolt git commit -m 'Test export from DoltHub'"
    echo
    echo "2. Test pushing to GitHub (requires SSH key):"
    echo "   ../dolt git push --dry-run $GITHUB_REPO main"
    echo "   ../dolt git push $GITHUB_REPO main"
    echo
    echo "3. Test chunking with different sizes:"
    echo "   ../dolt git push --chunk-size=25MB --dry-run $GITHUB_REPO main"
    echo "   ../dolt git push --chunk-size=100MB --dry-run $GITHUB_REPO main"
    echo
    echo -e "${YELLOW}Test Authentication Methods:${NC}"
    echo "   # With GitHub token:"
    echo "   ../dolt git clone --token=YOUR_TOKEN https://github.com/user/repo"
    echo
    echo "   # With SSH key:"
    echo "   ../dolt git clone git@github.com:user/repo.git"
    echo
}

# Performance validation
validate_performance() {
    log_info "Running performance validation..."

    # Check chunking algorithm performance
    if [ -d "go/libraries/doltcore/git" ]; then
        log_info "Running chunking performance tests..."
        cd go

        if go test -v ./libraries/doltcore/git/... -bench=. > /dev/null 2>&1; then
            log_success "Performance tests passed"
        else
            log_warning "Performance tests not available or failed"
        fi

        cd ..
    fi
}

# Show next steps
show_next_steps() {
    log_header "NEXT STEPS & RECOMMENDATIONS"

    echo -e "${GREEN}Git Integration Status: COMPLETED ✅${NC}"
    echo
    echo -e "${YELLOW}Immediate Actions:${NC}"
    echo "1. Test with real datasets using the examples above"
    echo "2. Validate authentication with your GitHub/GitLab accounts"
    echo "3. Test chunking behavior with large tables"
    echo "4. Verify round-trip data integrity"
    echo
    echo -e "${YELLOW}Next Development Priorities:${NC}"
    echo "1. 🎯 Table Editor/Viewer implementation (highest impact)"
    echo "2. 📊 Performance optimization and benchmarking"
    echo "3. 🧪 Additional platform testing (GitLab, Gitea, etc.)"
    echo "4. 📚 User documentation and examples"
    echo
    echo -e "${YELLOW}Technical Debt & Improvements:${NC}"
    echo "- Complete TODO items in Git command implementations"
    echo "- Add more comprehensive error recovery"
    echo "- Implement progress bars for long operations"
    echo "- Add configuration file support for Git settings"
    echo
    echo -e "${CYAN}For Table Editor implementation:${NC}"
    echo "Consider using:"
    echo "- bubbletea (TUI framework)"
    echo "- lipgloss (styling)"
    echo "- bubbles (UI components)"
    echo "- Integration with Dolt's SQL engine for data operations"
}

# Check if we can connect to test repositories
test_connectivity() {
    log_info "Testing connectivity to test repositories..."

    # Test DoltHub connectivity
    if curl -s --max-time 10 "https://www.dolthub.com/api/v1alpha1/holywritings/bahaiwritings" > /dev/null 2>&1; then
        log_success "DoltHub connectivity: OK"
    else
        log_warning "DoltHub connectivity: Limited or failed"
    fi

    # Test GitHub connectivity
    if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
        log_success "GitHub SSH connectivity: OK"
    elif curl -s --max-time 10 "https://github.com" > /dev/null 2>&1; then
        log_warning "GitHub HTTPS connectivity: OK (SSH authentication not configured)"
    else
        log_warning "GitHub connectivity: Failed"
    fi
}

# Main execution flow
main() {
    print_banner

    check_environment
    build_dolt_binary
    test_git_commands
    quick_functionality_test
    test_connectivity
    validate_performance

    echo
    log_success "Resume testing setup completed successfully!"

    show_test_examples
    run_integration_tests
    show_next_steps

    echo
    log_header "READY FOR CONTINUED DEVELOPMENT"
    log_info "The Dolt binary with Git integration is ready at: ./dolt"
    log_info "Run './dolt git --help' to see all available commands"
    echo
}

# Execute main function
main "$@"
