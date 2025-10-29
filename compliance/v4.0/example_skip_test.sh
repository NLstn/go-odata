#!/bin/bash

# OData v4 Compliance Test: Example Skip Test
# Demonstrates the use of skip_test function for unimplemented features
# Spec: https://docs.oasis-open.org/odata/odata/v4.0/

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../test_framework.sh"

echo "======================================"
echo "OData v4 Compliance Test"
echo "Section: Example Skip Test"
echo "======================================"
echo ""
echo "Description: This test demonstrates the skip_test functionality"
echo "             for marking tests as skipped when features are not implemented."
echo ""
echo "Spec Reference: https://docs.oasis-open.org/odata/odata/v4.0/"
echo ""

# Test 1: A passing test
test_passing() {
    return 0
}

run_test "This test should pass" test_passing

# Test 2: A skipped test - demonstrating the feature
skip_test "Delta token support" "Delta token feature is not yet implemented in go-odata"

# Test 3: Another passing test
test_another_passing() {
    return 0
}

run_test "Another passing test" test_another_passing

# Test 4: Another skipped test
skip_test "Stream property support" "Stream properties are not yet fully supported"

print_summary
