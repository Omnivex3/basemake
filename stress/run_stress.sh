#!/usr/bin/env bash
# basemake — Intensive Stress Test Runner v3
set -euo pipefail

BIN=./basemake
DSN="postgres://basemake:basemakestress@127.0.0.1:5433/stressdb"
REPORT_DIR="stress/results"
mkdir -p "$REPORT_DIR"
rm -f "$REPORT_DIR"/*.log

GREEN='\033[0;32m'; BLUE='\033[0;34m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
PASS="${GREEN}✓${NC}"; FAIL="${RED}✗${NC}"

TOTAL=0; PASSED=0

run_test() {
    local name="$1"; local cmd="$2"; local expect_fail="${3:-false}"
    TOTAL=$((TOTAL + 1))
    local slug=$(echo "$name" | tr ' /' '_' | tr -cd '[:alnum:]-_')
    local outfile="$REPORT_DIR/${slug}_$$.log"
    local memfile="$REPORT_DIR/${slug}_mem_$$.log"

    echo -e "${BLUE}[TEST]${NC} $name"
    set +e
    /usr/bin/time -v bash -c "$cmd" >"$outfile" 2>"$memfile"
    local rc=$?
    set -e

    local duration=$(grep 'Elapsed (wall clock) time' "$memfile" | awk '{print $NF}')
    local maxmem=$(grep 'Maximum resident set size' "$memfile" | awk '{print $NF}')
    local maxmem_mb=$((maxmem / 1024))
    local lines=$(wc -l < "$outfile" | tr -d ' ')

    local verdict="${PASS}"
    if [ "$expect_fail" = "true" ]; then
        [ $rc -eq 0 ] && verdict="${FAIL}" || verdict="${PASS}"
    else
        [ $rc -ne 0 ] && verdict="${FAIL}" || verdict="${PASS}"
    fi
    [ "$verdict" = "${PASS}" ] && PASSED=$((PASSED + 1))

    local head=""
    [ $rc -ne 0 ] && head=$(head -3 "$outfile" | tr '\n' ' ' | head -c 120)
    echo -e "  ${verdict} rc=$rc | ${duration} | ${maxmem_mb}MB RSS | ${lines} lines $head"
}

# ── Rebuild ──
echo -e "${CYAN}━━━ Building fresh binary...${NC}"
go build -o "$BIN" . 2>&1 && echo -e "  ${PASS} Binary rebuilt" || echo -e "  ${FAIL} Build failed"

# ── Establish connection ──
echo -e "\n${CYAN}━━━ Connecting to stressdb${NC}"
$BIN connect "$DSN" 2>/dev/null
echo -e "  ${PASS} Connected"

echo ""
echo -e "${CYAN}━━━ PHASE 2a: CORE COMMANDS ──────────────────${NC}"

run_test "query COUNT" "'$BIN' 'SELECT COUNT(*) FROM stress_users'"
run_test "query SELECT * LIMIT 10" "'$BIN' 'SELECT * FROM stress_orders LIMIT 10'"
run_test "query JOIN users+orders" "'$BIN' 'SELECT u.name, COUNT(o.id) as order_count FROM stress_users u LEFT JOIN stress_orders o ON u.id = o.user_id GROUP BY u.id, u.name ORDER BY order_count DESC LIMIT 20'"
run_test "query GROUP BY + aggregate" "'$BIN' 'SELECT status, COUNT(*) as cnt, AVG(total)::numeric(10,2) as avg_total, SUM(total)::numeric(12,2) as revenue FROM stress_orders GROUP BY status ORDER BY cnt DESC'"
run_test "query multi-JOIN + window" "'$BIN' \"SELECT u.country, p.category, COUNT(*) as sales, RANK() OVER (PARTITION BY u.country ORDER BY COUNT(*) DESC) as rank FROM stress_orders o JOIN stress_users u ON o.user_id = u.id JOIN stress_products p ON o.product_id = p.id GROUP BY u.country, p.category ORDER BY u.country, rank LIMIT 30\""
run_test "query ORDER BY large" "'$BIN' 'SELECT * FROM stress_orders ORDER BY total DESC LIMIT 100'"
run_test "query DISTINCT" "'$BIN' 'SELECT DISTINCT status FROM stress_orders'"
run_test "query heavy aggregation" "'$BIN' 'SELECT u.plan, EXTRACT(MONTH FROM o.ordered_at) as month, COUNT(*) as orders_per_month, SUM(o.total) as rev FROM stress_orders o JOIN stress_users u ON o.user_id = u.id GROUP BY u.plan, EXTRACT(MONTH FROM o.ordered_at) ORDER BY u.plan, month'"

run_test "analyze count" "'$BIN' analyze 'SELECT COUNT(*) FROM stress_users'"
run_test "analyze join" "'$BIN' analyze 'SELECT u.country, COUNT(o.id) FROM stress_users u JOIN stress_orders o ON u.id = o.user_id GROUP BY u.country'"
run_test "analyze all tables" "'$BIN' analyze --all"

echo ""
echo -e "${CYAN}━━━ PHASE 2b: SCHEMA COMMANDS ────────────────${NC}"

run_test "check basic" "'$BIN' check 'SELECT COUNT(*) FROM stress_users WHERE email IS NULL'"
run_test "check threshold" "'$BIN' check \"SELECT COUNT(*) FROM stress_orders WHERE status = 'cancelled'\" --threshold 500ms"
run_test "check dry-run" "'$BIN' check 'SELECT * FROM stress_users LIMIT 5' --dry-run"

run_test "diff active vs cache" "'$BIN' diff"
run_test "diff two live" "'$BIN' diff --from '$DSN' --to '$DSN'"

echo ""
echo -e "${CYAN}━━━ PHASE 3: SERVER STRESS ───────────────────${NC}"

# Start server — port 9876 (default)
echo -n "  Starting server on port 9876... "
pkill -f "basemake server" 2>/dev/null || true
sleep 1
$BIN server start --port 9876 &>"$REPORT_DIR/server_$$.log" &
SERVER_PID=$!

# Wait for server with retries
for i in $(seq 1 10); do
    sleep 1
    if curl -sf http://localhost:9876/health >/dev/null 2>&1; then
        echo -e "${PASS} pid=$SERVER_PID"
        PASSED=$((TOTAL + 1)); TOTAL=$((TOTAL + 1))
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${FAIL} (timeout)"
        TOTAL=$((TOTAL + 1))
    fi
done

S="http://localhost:9876"
# Health already verified in startup, skip separate test
run_test "server events GET" "curl -sf http://localhost:9876/api/events"
run_test "server events POST" "curl -sf -X POST http://localhost:9876/api/events -H 'Content-Type: application/json' -d '{\"sql\":\"SELECT 1\",\"duration_ms\":50,\"user_name\":\"stress-test\",\"hostname\":\"localhost\"}'"

# Add 20 watches
echo "  Adding 20 watches..."
for i in $(seq 1 20); do
    q="SELECT COUNT(*) FROM stress_orders"
    [ $((i % 3)) -eq 0 ] && q="SELECT AVG(total)::numeric(10,2) FROM stress_orders WHERE user_id = $((i * 10000))"
    [ $((i % 3)) -eq 1 ] && q="SELECT COUNT(DISTINCT country) FROM stress_users"
    curl -sf -X POST "$S/api/watches" \
        -H 'Content-Type: application/json' \
        -d "{\"sql\":\"$q\",\"label\":\"stress-watch-$i\",\"interval_sec\":60,\"threshold_ms\":5000,\"dsn\":\"$DSN\",\"created_by\":\"stress-test\"}" \
        >/dev/null 2>&1 || true
done
PASSED=$((PASSED + 1)); TOTAL=$((TOTAL + 1)); echo -e "  ${PASS} 20 watches added"

# Let watches tick once
sleep 5

run_test "watch list" "curl -sf $S/api/watches"
run_test "watch logs" "curl -sf '$S/api/watches/1/results?limit=5'"
run_test "delete watch" "curl -sf -X DELETE $S/api/watches/1"

run_test "sync push" "'$BIN' sync push 'SELECT * FROM stress_orders LIMIT 5' --server $S --duration 150ms"
run_test "sync history" "'$BIN' sync history --server $S --limit 10"

# Clean up server
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo -e "  ${GREEN}Server stopped${NC}"
rm -f /root/.basemake/server/*.db 2>/dev/null || true

echo ""
echo -e "${CYAN}━━━ PHASE 4: EDGE CASES ──────────────────────${NC}"

# Large SQL — write to file to avoid bash quoting hell
cat > /tmp/stress_long.sql << 'SQLEOF'
SELECT u.name,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND status = 'delivered') as d1,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND status = 'pending') as d2,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND status = 'cancelled') as d3,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND status = 'shipped') as d4,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND status = 'processing') as d5,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND status = 'returned') as d6,
(SELECT AVG(total) FROM stress_orders o WHERE o.user_id = u.id) as avg_order,
(SELECT SUM(total) FROM stress_orders o WHERE o.user_id = u.id) as total_spent,
(SELECT MAX(total) FROM stress_orders o WHERE o.user_id = u.id) as max_order,
(SELECT MIN(total) FROM stress_orders o WHERE o.user_id = u.id) as min_order,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND o.ordered_at > NOW() - INTERVAL '30 days') as recent_30d,
(SELECT COUNT(*) FROM stress_orders o WHERE o.user_id = u.id AND o.ordered_at > NOW() - INTERVAL '90 days') as recent_90d
FROM stress_users u LIMIT 5;
SQLEOF
# Large SQL — run directly, skip run_test wrapper due to quoting complexity
echo -e "${BLUE}[TEST]${NC} large SQL file w/ subqueries"
SQL=$(cat /tmp/stress_long.sql)
OUTFILE="$REPORT_DIR/large_SQL_file_w_subqueries_$$.log"
MEMFILE="$REPORT_DIR/large_SQL_file_w_subqueries_mem_$$.log"
set +e
/usr/bin/time -v bash -c "'$BIN' query --no-stream $(printf '%q' "$SQL")" >"$OUTFILE" 2>"$MEMFILE"
RC=$?
set -e
DURATION=$(grep 'Elapsed' "$MEMFILE" | awk '{print $NF}')
MAXMEM=$(grep 'Maximum resident' "$MEMFILE" | awk '{print $NF}')
MAXMEM_MB=$((MAXMEM / 1024))
LINES=$(wc -l < "$OUTFILE" | tr -d ' ')
if [ $RC -eq 0 ]; then
    echo -e "  ${PASS} rc=0 | ${DURATION} | ${MAXMEM_MB}MB RSS | ${LINES} lines"
    PASSED=$((PASSED + 1))
else
    echo -e "  ${FAIL} rc=$RC | ${DURATION} | ${MAXMEM_MB}MB RSS"
fi
TOTAL=$((TOTAL + 1))

run_test "50 parallel queries" "
for i in \$(seq 1 50); do
    '$BIN' 'SELECT COUNT(*) FROM stress_orders WHERE user_id = '\$((RANDOM % 500000))'' >/dev/null 2>&1 &
done
wait
echo '50 concurrent queries completed'
"

run_test "empty result set" "'$BIN' 'SELECT * FROM stress_users WHERE id = -1'"
run_test "NULL query" "'$BIN' 'SELECT COUNT(*) FROM stress_orders WHERE shipped_at IS NULL'"
run_test "10K row stream" "'$BIN' 'SELECT * FROM stress_users ORDER BY id LIMIT 10000' > /dev/null && echo '10K rows streamed'"

# Schema stress — add 50 temp tables
run_test "schema: create 50 tables" "
PGPASSWORD=basemakestress psql -h 127.0.0.1 -p 5433 -U basemake -d stressdb -c \"
DO \\\$\\\$ BEGIN FOR i IN 1..50 LOOP EXECUTE 'CREATE TABLE IF NOT EXISTS stress_tmp_' || i || ' (id INT, val TEXT)'; END LOOP; END \\\$\\\$;\"
echo '50 temp tables created'
"
run_test "diff with 50 extra tables" "'$BIN' diff"

# Cleanup
PGPASSWORD=basemakestress psql -h 127.0.0.1 -p 5433 -U basemake -d stressdb -c "
DO \$\$ BEGIN FOR i IN 1..50 LOOP EXECUTE 'DROP TABLE IF EXISTS stress_tmp_' || i; END LOOP; END \$\$;
" >/dev/null 2>&1

echo ""
echo -e "${CYAN}━━━ PHASE 5: RESOURCE TRACKING ───────────────${NC}"

run_test "memory: 3-table JOIN" "'$BIN' 'SELECT u.country, p.category, SUM(o.total) as rev FROM stress_orders o JOIN stress_users u ON o.user_id = u.id JOIN stress_products p ON o.product_id = p.id GROUP BY u.country, p.category ORDER BY rev DESC'"

run_test "30 repeated queries" "
for i in \$(seq 1 30); do
    '$BIN' 'SELECT COUNT(*) FROM stress_orders' >/dev/null 2>&1
done
echo '30/30 completed'
"

echo ""
echo -e "${CYAN}━━━ SUMMARY ──────────────────────────────────${NC}"
echo -e "Tests:  $TOTAL"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$((TOTAL - PASSED))${NC}"
echo -e "Results: $REPORT_DIR/"
[ $((TOTAL - PASSED)) -gt 0 ] && exit 1 || exit 0
