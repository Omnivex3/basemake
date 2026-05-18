#!/bin/bash
# Short demo script for basemake GIF recording - v3 with basemake check
clear
sleep 0.3

echo "╔══════════════════════════════════════════════╗"
echo "║  basemake — AI-powered database CLI          ║"
echo "║  NL queries · Performance analysis · CI gate ║"
echo "╚══════════════════════════════════════════════╝"
echo ""

sleep 0.5

# 1. Connect
echo "$ basemake connect postgres://postgres:***@localhost:5433/demodb"
sleep 0.2
./basemake connect "postgres://postgres:postgres@localhost:5433/demodb" 2>&1 | head -5
echo ""

# 2. NL query
sleep 1
echo "$ basemake \"show me users who signed up last week\""
sleep 0.2
./basemake "show me users who signed up last week" --no-stream 2>&1
echo ""

# 3. Top products with explain
sleep 1
echo "$ basemake \"top 5 products by revenue\" --explain"
sleep 0.2
./basemake "top 5 products by revenue" --explain --no-stream 2>&1
echo ""

# 4. Performance analysis
sleep 1
echo "$ basemake analyze \"SELECT * FROM orders WHERE status='delivered'\""
sleep 0.2
./basemake analyze "SELECT * FROM orders WHERE status='delivered'" 2>&1
echo ""

# 5. CI/CD gate — basemake check (fast query)
sleep 1
echo "$ basemake check 'SELECT count(*) FROM users' --threshold 1s"
sleep 0.2
./basemake check "SELECT count(*) FROM users" --threshold 1s 2>&1
echo ""

# 6. CI/CD gate — basemake check dangerous (seq scan)
sleep 1
echo "$ basemake check 'SELECT * FROM orders' --threshold 5ms"
sleep 0.2
./basemake check "SELECT * FROM orders WHERE total > 100" --threshold 5ms 2>&1
echo ""

# 7. Dry-run mode for migration safety
sleep 1
echo "$ basemake check 'UPDATE accounts SET balance = 0' --dry-run"
sleep 0.2
./basemake check "UPDATE accounts SET balance = 0" --dry-run 2>&1
echo ""

# 8. Fin
sleep 1
echo ""
echo "✅ All local. All private. All yours."
echo "   github.com/DynamicKarabo/basemake"
echo ""
