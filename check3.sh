#!/bin/bash
cd /opt/zhangyi

# Write login JSON inside container
docker compose exec -T zhangyi sh -c 'printf "{\"username\":\"admin\",\"password\":\"admin123\"}" > /tmp/l.json'

# Login
TOKEN=$(docker compose exec -T zhangyi curl -s -X POST http://localhost:8080/api/auth/login -H 'Content-Type: application/json' -d @/tmp/l.json)
echo "LOGIN RESPONSE: $TOKEN"

# Extract token
JWT=$(echo "$TOKEN" | python3 -c "import sys,json; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)
echo "JWT: ${JWT:0:30}..."

if [ -z "$JWT" ]; then
  echo "FAILED TO GET TOKEN"
  exit 1
fi

echo "=== 利润表 ==="
docker compose exec -T zhangyi curl -s -H "Authorization: Bearer $JWT" 'http://localhost:8080/api/books/3/reports/income-statement?period=2026-06'

echo ""
echo "=== 费用统计 ==="
docker compose exec -T zhangyi curl -s -H "Authorization: Bearer $JWT" 'http://localhost:8080/api/books/3/reports/expense?period=2026-06'

echo ""
echo "=== 资产负债表 ==="
docker compose exec -T zhangyi curl -s -H "Authorization: Bearer $JWT" 'http://localhost:8080/api/books/3/reports/balance-sheet?period=2026-06'
