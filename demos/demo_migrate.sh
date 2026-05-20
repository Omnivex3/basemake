#!/bin/bash
# Demo: basemake check --migrate — predict migration impact
export PAGER=cat
clear
sleep 0.3

echo "╔══════════════════════════════════════════════╗"
echo "║  basemake check --migrate                    ║"
echo "║  Know what breaks before you run the change  ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
sleep 0.5

# Write a sample migration file
cat > /tmp/demo_migration.sql << 'SQLEOF'
-- Remove unused index, add covering index
DROP INDEX IF EXISTS idx_topics_views;
CREATE INDEX idx_topics_views_covering ON topics (category_id, views, created_at);
SQLEOF

echo "$ cat migrations/cleanup.sql"
cat /tmp/demo_migration.sql
echo ""
sleep 0.5

echo "$ basemake check --migrate migrations/cleanup.sql"
sleep 0.3
basemake check --migrate /tmp/demo_migration.sql 2>&1
echo ""
sleep 1

echo "$ basemake check \"SELECT * FROM topics ORDER BY views DESC LIMIT 10\" --dry-run"
sleep 0.3
basemake check "SELECT * FROM topics ORDER BY views DESC LIMIT 10" --dry-run 2>&1
echo ""
sleep 0.5

echo "✅ basemake check --migrate — your profile history protects every deploy."
