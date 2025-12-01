#!/bin/bash
# Benchmark script: Measure baseline performance for caching system design
# Usage: ./scripts/benchmark-baseline.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BENCHMARK_DIR="$PROJECT_ROOT/benchmarks"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$BENCHMARK_DIR/baseline-$TIMESTAMP.txt"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

mkdir -p "$BENCHMARK_DIR"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  gz-quality Performance Baseline Benchmark${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Report will be saved to: $REPORT_FILE"
echo ""

# Ensure binary is built
if [ ! -f "$PROJECT_ROOT/build/gzh-quality" ]; then
    echo -e "${YELLOW}Building binary...${NC}"
    cd "$PROJECT_ROOT"
    make build
fi

BINARY="$PROJECT_ROOT/build/gzh-quality"

# Create report header
cat > "$REPORT_FILE" << EOF
gz-quality Performance Baseline Benchmark
==========================================

Date: $(date)
Version: $(git describe --tags --always 2>/dev/null || echo "unknown")
Commit: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
Platform: $(uname -s) $(uname -m)
CPU: $(sysctl -n machdep.cpu.brand_string 2>/dev/null || grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs)
Memory: $(sysctl -n hw.memsize 2>/dev/null || grep MemTotal /proc/meminfo | awk '{print $2 " KB"}')
Go Version: $(go version)

Project Statistics
------------------
EOF

# Count files by language
echo "" >> "$REPORT_FILE"
echo "File counts:" >> "$REPORT_FILE"
find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" | wc -l | xargs echo "  Go files:" >> "$REPORT_FILE"
find . -name "*.py" ! -path "./vendor/*" ! -path "./.git/*" ! -path "./.venv/*" | wc -l | xargs echo "  Python files:" >> "$REPORT_FILE"
find . -name "*.ts" -o -name "*.tsx" ! -path "./node_modules/*" ! -path "./.git/*" | wc -l | xargs echo "  TypeScript files:" >> "$REPORT_FILE"
find . -name "*.js" -o -name "*.jsx" ! -path "./node_modules/*" ! -path "./.git/*" | wc -l | xargs echo "  JavaScript files:" >> "$REPORT_FILE"

# Total lines of code
echo "" >> "$REPORT_FILE"
find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" -exec wc -l {} + | tail -1 | awk '{print "  Total Go lines: " $1}' >> "$REPORT_FILE"

cat >> "$REPORT_FILE" << EOF

Benchmark Scenarios
-------------------

EOF

echo -e "${GREEN}Scenario 1: Full check (all files, no cache)${NC}"
echo ""

# Scenario 1: Full check
echo "Scenario 1: Full check (all files)" >> "$REPORT_FILE"
echo "Command: gz-quality run" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

TIME_OUTPUT=$(mktemp)
{ time "$BINARY" run 2>&1; } 2> "$TIME_OUTPUT" || true

REAL_TIME=$(grep "^real" "$TIME_OUTPUT" | awk '{print $2}')
echo "  Execution time: $REAL_TIME" >> "$REPORT_FILE"
echo "  Exit code: $?" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo -e "  Time: ${YELLOW}$REAL_TIME${NC}"
echo ""

# Scenario 2: Staged files only
echo -e "${GREEN}Scenario 2: Staged files check${NC}"
echo ""

echo "Scenario 2: Staged files check" >> "$REPORT_FILE"
echo "Command: gz-quality run --staged" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Create a staged file
TEST_FILE="$PROJECT_ROOT/tmp/test-benchmark.go"
mkdir -p "$PROJECT_ROOT/tmp"
cat > "$TEST_FILE" << 'GOFILE'
package main

import "fmt"

func main() {
    fmt.Println("benchmark test")
}
GOFILE

git add "$TEST_FILE" 2>/dev/null || true

TIME_OUTPUT=$(mktemp)
{ time "$BINARY" run --staged 2>&1; } 2> "$TIME_OUTPUT" || true

REAL_TIME=$(grep "^real" "$TIME_OUTPUT" | awk '{print $2}')
echo "  Execution time: $REAL_TIME" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo -e "  Time: ${YELLOW}$REAL_TIME${NC}"
echo ""

# Cleanup
git reset "$TEST_FILE" 2>/dev/null || true
rm -f "$TEST_FILE"

# Scenario 3: Check only (no modifications)
echo -e "${GREEN}Scenario 3: Check only (no file modifications)${NC}"
echo ""

echo "Scenario 3: Check only (no modifications)" >> "$REPORT_FILE"
echo "Command: gz-quality check" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

TIME_OUTPUT=$(mktemp)
{ time "$BINARY" check 2>&1; } 2> "$TIME_OUTPUT" || true

REAL_TIME=$(grep "^real" "$TIME_OUTPUT" | awk '{print $2}')
echo "  Execution time: $REAL_TIME" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo -e "  Time: ${YELLOW}$REAL_TIME${NC}"
echo ""

# Scenario 4: Specific tool only (golangci-lint)
echo -e "${GREEN}Scenario 4: Single tool (golangci-lint)${NC}"
echo ""

echo "Scenario 4: Single tool execution" >> "$REPORT_FILE"
echo "Command: gz-quality tool golangci-lint" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

TIME_OUTPUT=$(mktemp)
{ time "$BINARY" tool golangci-lint 2>&1; } 2> "$TIME_OUTPUT" || true

REAL_TIME=$(grep "^real" "$TIME_OUTPUT" | awk '{print $2}')
echo "  Execution time: $REAL_TIME" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo -e "  Time: ${YELLOW}$REAL_TIME${NC}"
echo ""

# Scenario 5: Parallel execution with different worker counts
echo -e "${GREEN}Scenario 5: Worker count comparison${NC}"
echo ""

echo "Scenario 5: Worker count impact" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

for WORKERS in 1 2 4 8; do
    echo "  Workers: $WORKERS" >> "$REPORT_FILE"
    echo "  Command: gz-quality run --workers $WORKERS" >> "$REPORT_FILE"

    TIME_OUTPUT=$(mktemp)
    { time "$BINARY" run --workers "$WORKERS" 2>&1; } 2> "$TIME_OUTPUT" || true

    REAL_TIME=$(grep "^real" "$TIME_OUTPUT" | awk '{print $2}')
    echo "    Execution time: $REAL_TIME" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"

    echo -e "    $WORKERS workers: ${YELLOW}$REAL_TIME${NC}"
done
echo ""

# Tool availability
cat >> "$REPORT_FILE" << EOF

Tool Availability
-----------------

EOF

echo -e "${GREEN}Tool versions:${NC}"
"$BINARY" version >> "$REPORT_FILE" 2>&1 || true

# Summary
cat >> "$REPORT_FILE" << EOF

Performance Metrics Summary
---------------------------

Key Findings:
1. Full check baseline established
2. Worker count optimal range identified
3. Tool-specific performance measured
4. Baseline for cache improvement comparison

Next Steps:
- Implement caching system (see docs/developer/CACHING.md)
- Re-run benchmark with cache enabled
- Target: 50-80% improvement for repeated checks
- Target: < 1s for cache lookup overhead

EOF

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Benchmark complete!${NC}"
echo ""
echo "Report saved to: $REPORT_FILE"
echo ""
echo "To view the report:"
echo "  cat $REPORT_FILE"
echo ""
echo "To compare with future benchmarks:"
echo "  diff $REPORT_FILE benchmarks/baseline-<new-timestamp>.txt"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
