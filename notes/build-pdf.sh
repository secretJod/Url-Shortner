#!/usr/bin/env bash
# Rebuilds url-shortener-field-notebook.pdf from the 5 source markdown files
# in this folder. Requires: pandoc, weasyprint (`brew install pandoc`,
# `pip3 install weasyprint`, `brew install pango` — pango is weasyprint's
# native dependency; without it weasyprint fails with a libgobject error).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

cat 00-title.md \
    part1-setup-architecture.md \
    part2-backend-apis.md \
    part3-frontend-user-journey.md \
    part4-monitoring-deployment.md > /tmp/combined.md

# DYLD_FALLBACK_LIBRARY_PATH works around weasyprint/cffi not finding
# Homebrew's libgobject on macOS by default.
DYLD_FALLBACK_LIBRARY_PATH=/opt/homebrew/lib pandoc /tmp/combined.md \
  --from markdown --to html5 --css style.css --standalone \
  --metadata title="URL Shortener — field notebook" \
  --toc --toc-depth=2 \
  -o /tmp/combined.html

DYLD_FALLBACK_LIBRARY_PATH=/opt/homebrew/lib weasyprint \
  /tmp/combined.html url-shortener-field-notebook.pdf

rm -f /tmp/combined.md /tmp/combined.html
echo "Rebuilt url-shortener-field-notebook.pdf"
