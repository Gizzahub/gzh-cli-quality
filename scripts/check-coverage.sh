#!/bin/bash
# Check test coverage and enforce minimum thresholds
# Usage: ./scripts/check-coverage.sh [--min PERCENTAGE]

set -e

# Default minimum coverage threshold
MIN_COVERAGE=40.0

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --min)
            MIN_COVERAGE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [--min PERCENTAGE]"
            echo ""
            echo "Options:"
            echo "  --min PERCENTAGE    Set minimum coverage threshold (default: 40.0)"
            echo "  --help, -h          Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "=================================="
echo "🧪 Running tests with coverage..."
echo "=================================="
echo ""

# Run tests with coverage
if ! go test ./... -coverprofile=coverage.out -covermode=atomic; then
    echo ""
    echo "❌ Tests failed"
    exit 1
fi

echo ""
echo "=================================="
echo "📊 Coverage Analysis"
echo "=================================="
echo ""

# Get total coverage
COVERAGE_LINE=$(go tool cover -func=coverage.out | grep total)
COVERAGE=$(echo "$COVERAGE_LINE" | awk '{print $3}' | sed 's/%//')

echo "Total Coverage: $COVERAGE%"
echo "Minimum Required: $MIN_COVERAGE%"
echo ""

# Check threshold
if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
    echo "❌ FAIL: Coverage $COVERAGE% is below minimum threshold of $MIN_COVERAGE%"
    echo ""
    echo "To improve coverage:"
    echo "  1. Run: go tool cover -html=coverage.out"
    echo "  2. Identify uncovered code (red sections)"
    echo "  3. Write tests for critical paths"
    echo ""
    exit 1
fi

echo "✅ PASS: Coverage $COVERAGE% meets minimum threshold of $MIN_COVERAGE%"

# Check if it exceeds recommended threshold
if (( $(echo "$COVERAGE >= 50.0" | bc -l) )); then
    echo "🎉 EXCELLENT: Coverage exceeds recommended threshold of 50%"
fi

echo ""
echo "=================================="
echo "📦 Coverage by Package"
echo "=================================="
echo ""

# Display package-level coverage
go tool cover -func=coverage.out | \
    grep -E "github.com/Gizzahub/gzh-cli-quality/(config|detector|executor|git|report|tools)/" | \
    awk '{print $1 "\t" $3}' | \
    sed 's|github.com/Gizzahub/gzh-cli-quality/||' | \
    sort -t/ -k1,1 | \
    while IFS=$'\t' read -r package coverage; do
        # Extract package name (first part before /)
        pkg_name=$(echo "$package" | cut -d/ -f1)

        # Color code based on coverage
        cov_num=$(echo "$coverage" | sed 's/%//')
        if (( $(echo "$cov_num >= 80.0" | bc -l) )); then
            printf "✅ %-20s %s\n" "$pkg_name" "$coverage"
        elif (( $(echo "$cov_num >= 60.0" | bc -l) )); then
            printf "⚠️  %-20s %s\n" "$pkg_name" "$coverage"
        else
            printf "❌ %-20s %s\n" "$pkg_name" "$coverage"
        fi
    done | sort -u

echo ""
echo "=================================="
echo "📝 Coverage Report Generated"
echo "=================================="
echo ""
echo "Files created:"
echo "  - coverage.out (profile data)"
echo ""
echo "To view detailed HTML report:"
echo "  go tool cover -html=coverage.out -o coverage.html"
echo "  open coverage.html  # macOS"
echo "  xdg-open coverage.html  # Linux"
echo ""

# Optional: Generate HTML report automatically
if command -v xdg-open &> /dev/null || command -v open &> /dev/null; then
    read -p "Generate and open HTML report now? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        go tool cover -html=coverage.out -o coverage.html
        if command -v xdg-open &> /dev/null; then
            xdg-open coverage.html &
        elif command -v open &> /dev/null; then
            open coverage.html
        fi
        echo "✅ HTML report opened in browser"
    fi
fi

echo ""
echo "✅ Coverage check completed successfully"
exit 0
