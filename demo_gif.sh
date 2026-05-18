#!/bin/bash
# Demo GIF script — Ghost + Mastodon + Budgets
clear
sleep 0.3

echo "╔══════════════════════════════════════════════╗"
echo "║  basemake — AI-powered database CLI          ║"
echo "║  NL queries · Perf analysis · Policy as code ║"
echo "╚══════════════════════════════════════════════╝"
echo ""

sleep 0.5

# ═══════════════ Ghost ═══════════════
echo "─── Ghost CMS (4 tables, 200 posts) ───"
echo "$ basemake connect"
sleep 0.2
./basemake connect "postgres://postgres:***@localhost:5433/ghost_demo" 2>&1 | head -6
echo ""

sleep 1
echo "$ basemake \"show me the 5 most recent published posts with their authors\""
sleep 0.2
./basemake "show me the 5 most recent published posts with their authors" --no-stream 2>&1
echo ""

sleep 1
echo "$ basemake query \"SELECT t.name, COUNT(pt.post_id) as c FROM tags t JOIN posts_tags pt ON pt.tag_id = t.id GROUP BY t.name ORDER BY c DESC\""
sleep 0.2
./basemake query "SELECT t.name, COUNT(pt.post_id) as c FROM tags t JOIN posts_tags pt ON pt.tag_id = t.id GROUP BY t.name ORDER BY c DESC" --no-stream 2>&1
echo ""

# ═══════════════ Mastodon ═══════════════
sleep 1
echo ""
echo "─── Mastodon (4 tables, 500 statuses) ───"
echo "$ basemake connect"
sleep 0.2
./basemake connect "postgres://postgres:***@localhost:5433/mastodon_demo" 2>&1 | head -6
echo ""

sleep 1
echo "$ basemake \"top 10 accounts by follower count\""
sleep 0.2
./basemake "top 10 accounts by follower count" --no-stream 2>&1
echo ""

sleep 1
echo "$ basemake analyze \"SELECT a.username, COUNT(s.id) FROM accounts a JOIN statuses s ON s.account_id = a.id GROUP BY a.username ORDER BY COUNT(s.id) DESC LIMIT 10\""
sleep 0.2
./basemake analyze "SELECT a.username, COUNT(s.id) FROM accounts a JOIN statuses s ON s.account_id = a.id GROUP BY a.username ORDER BY COUNT(s.id) DESC LIMIT 10" 2>&1
echo ""

# ═══════════════ Budgets ═══════════════
sleep 1
echo ""
echo "─── Database policy as code ───"
echo "$ basemake budget list"
sleep 0.2
./basemake budget list 2>&1
echo ""

sleep 1
echo "$ basemake check \"SELECT * FROM statuses\""
sleep 0.2
./basemake check "SELECT * FROM statuses WHERE favourites_count > 10" 2>&1
echo "Exit: $?"
echo ""

sleep 1
echo "$ basemake check \"SELECT count(*) FROM accounts\""
sleep 0.2
./basemake check "SELECT count(*) FROM accounts" 2>&1
echo "Exit: $?"
echo ""

sleep 1
echo "$ basemake check \"UPDATE accounts SET bio = 'updated'\" --dry-run"
sleep 0.2
./basemake check "UPDATE accounts SET bio = 'updated'" --dry-run 2>&1
echo ""

# ═══════════════ Finish ═══════════════
sleep 1
echo ""
echo "✅ Done."
