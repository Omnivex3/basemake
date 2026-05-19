#!/bin/bash
# Demo GIF script — shows connect, query, analyze, check
clear
sleep 0.3

echo "╔══════════════════════════════════════════════╗"
echo "║  basemake — AI-powered database CLI          ║"
echo "║  NL queries · Perf analysis · Policy as code ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
sleep 0.5

# ═══════════════ Connect ═══════════════
echo "─── Connecting to Ghost CMS (4 tables, 200 posts) ───"
echo "$ basemake connect postgres://postgres:***@localhost:5433/ghost_demo"
sleep 0.2
/tmp/basemake connect "postgres://postgres:demo@localhost:5433/ghost_demo" 2>&1 | head -7
echo ""
sleep 1

# ═══════════════ Raw SQL Query ═══════════════
echo "$ basemake query \"SELECT title, published_at FROM posts ORDER BY published_at DESC LIMIT 5\""
sleep 0.2
/tmp/basemake query "SELECT title, published_at FROM posts ORDER BY published_at DESC LIMIT 5" --no-stream 2>&1
echo ""
sleep 1

# ═══════════════ Analyze ═══════════════
echo "$ basemake analyze \"SELECT * FROM posts WHERE status = 'published'\""
sleep 0.2
/tmp/basemake analyze "SELECT * FROM posts WHERE status = 'published'" 2>&1
echo ""
sleep 1

# ═══════════════ Check (CI gate with warning) ═══════════════
echo "$ basemake check \"SELECT * FROM posts WHERE title LIKE '%guide%'\""
sleep 0.2
/tmp/basemake check "SELECT * FROM posts WHERE title LIKE '%guide%'" 2>&1
echo ""
sleep 0.5

echo "$ basemake check \"UPDATE posts SET title = 'hacked'\" --dry-run"
sleep 0.2
/tmp/basemake check "UPDATE posts SET title = 'hacked'" --dry-run 2>&1
echo ""

# ═══════════════ Connect to mastodon ═══════════════
sleep 1
echo ""
echo "─── Switching to Mastodon (4 tables, 500 statuses) ───"
echo "$ basemake connect postgres://postgres:***@localhost:5433/mastodon_demo"
sleep 0.2
/tmp/basemake connect "postgres://postgres:demo@localhost:5433/mastodon_demo" 2>&1 | head -7
echo ""
sleep 1

# ═══════════════ Analyze with join ═══════════════
echo "$ basemake analyze \"SELECT a.username, COUNT(s.id) AS c FROM accounts a JOIN statuses s ON s.account_id = a.id GROUP BY a.username ORDER BY c DESC LIMIT 5\""
sleep 0.2
/tmp/basemake analyze "SELECT a.username, COUNT(s.id) AS c FROM accounts a JOIN statuses s ON s.account_id = a.id GROUP BY a.username ORDER BY c DESC LIMIT 5" 2>&1
echo ""

# ═══════════════ Finish ═══════════════
sleep 1
echo ""
echo "✅ Done. basemake — all local. all private. all yours."
