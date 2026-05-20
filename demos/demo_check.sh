#!/bin/bash
# Demo: basemake check — CI gate for query safety
export PAGER=cat
clear
sleep 0.3

echo "╔══════════════════════════════════════════════╗"
echo "║  basemake check                              ║"
echo "║  Block bad queries before they hit prod      ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
sleep 0.5

echo "$ basemake check \"SELECT * FROM topics WHERE title LIKE '%guide%'\""
sleep 0.3
basemake check "SELECT * FROM topics WHERE title LIKE '%guide%'" 2>&1
echo ""
sleep 1

echo "$ basemake check \"UPDATE topics SET title = 'hacked'\" --dry-run"
sleep 0.3
basemake check "UPDATE topics SET title = 'hacked'" --dry-run 2>&1
echo ""
sleep 0.5

echo "✅ basemake check — catch it in CI, not in production."
