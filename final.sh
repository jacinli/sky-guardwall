#!/bin/bash
export PATH=$PATH:/opt/homebrew/bin:/usr/bin
cd /Users/jacinlee/selfwork/job/self-proj/sky-guardwall
git add -A
git status --short
git diff --cached --stat
# only commit if there's something staged
if ! git diff --cached --quiet; then
  git commit -m "chore: remove temp scripts"
  git push origin main
fi
echo "FINAL_OK"
