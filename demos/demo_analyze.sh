#!/bin/bash
# Demo: basemake analyze — see slow queries in real time
export PAGER=cat
clear
sleep 0.3

echo "╔══════════════════════════════════════════════╗"
echo "║  basemake analyze                            ║"
echo "║  See exactly how each query executes         ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
sleep 0.5

echo "$ basemake analyze \"SELECT u.username, t.title, c.name AS category FROM topics t JOIN users u ON u.id = t.user_id JOIN categories c ON c.id = t.category_id ORDER BY t.views DESC LIMIT 20\""
sleep 0.3
basemake analyze "SELECT u.username, t.title, c.name AS category FROM topics t JOIN users u ON u.id = t.user_id JOIN categories c ON c.id = t.category_id ORDER BY t.views DESC LIMIT 20" 2>&1
echo ""
sleep 1

echo "$ basemake analyze \"SELECT username, email, trust_level, last_seen_at FROM users WHERE trust_level >= 3\""
sleep 0.3
basemake analyze "SELECT username, email, trust_level, last_seen_at FROM users WHERE trust_level >= 3" 2>&1
echo ""
sleep 0.5

echo "$ basemake analyze \"SELECT name, description, topics_count FROM categories ORDER BY topics_count DESC\""
sleep 0.3
basemake analyze "SELECT name, description, topics_count FROM categories ORDER BY topics_count DESC" 2>&1
echo ""
sleep 0.5

echo "✅ basemake analyze — query performance, instantly."
