#!/bin/bash
# Example script demonstrating ETag support in go-odata
# This script assumes the development server is running on localhost:8080

set -e

echo "========================================"
echo "ETag Support Example"
echo "========================================"
echo ""

# Step 1: Get an entity and retrieve its ETag
echo "Step 1: GET /Products(1) to retrieve ETag"
echo "----------------------------------------"
RESPONSE=$(curl -s -i 'http://localhost:8080/Products(1)')
ETAG=$(echo "$RESPONSE" | grep -i "^Etag:" | sed 's/Etag: //i' | tr -d '\r')
echo "Retrieved ETag: $ETAG"
echo ""

# Step 2: Update with correct ETag (should succeed)
echo "Step 2: PATCH /Products(1) with correct ETag"
echo "---------------------------------------------"
curl -s -i -X PATCH 'http://localhost:8080/Products(1)' \
  -H 'Content-Type: application/json' \
  -H "If-Match: $ETAG" \
  -d '{"Price": 899.99}' | head -5
echo ""
echo "✓ Update succeeded with matching ETag"
echo ""

# Step 3: Try to update with incorrect ETag (should fail)
echo "Step 3: PATCH /Products(1) with incorrect ETag"
echo "-----------------------------------------------"
curl -s -i -X PATCH 'http://localhost:8080/Products(1)' \
  -H 'Content-Type: application/json' \
  -H 'If-Match: W/"wrongetag"' \
  -d '{"Price": 799.99}' | head -10
echo ""
echo "✗ Update failed with non-matching ETag (412 Precondition Failed)"
echo ""

# Step 4: Update with wildcard (should succeed)
echo "Step 4: PATCH /Products(1) with wildcard If-Match: *"
echo "----------------------------------------------------"
curl -s -i -X PATCH 'http://localhost:8080/Products(1)' \
  -H 'Content-Type: application/json' \
  -H 'If-Match: *' \
  -d '{"Price": 949.99}' | head -5
echo ""
echo "✓ Update succeeded with wildcard If-Match"
echo ""

# Step 5: Create a new entity and get its ETag
echo "Step 5: POST /Products to create new entity"
echo "--------------------------------------------"
RESPONSE=$(curl -s -i -X POST 'http://localhost:8080/Products' \
  -H 'Content-Type: application/json' \
  -H 'Prefer: return=representation' \
  -d '{"Name": "New Product", "Price": 199.99, "Category": "Test", "Version": 1}')
NEW_ETAG=$(echo "$RESPONSE" | grep -i "^Etag:" | sed 's/Etag: //i' | tr -d '\r')
echo "Created new product with ETag: $NEW_ETAG"
echo ""

echo "========================================"
echo "ETag Demo Complete"
echo "========================================"
