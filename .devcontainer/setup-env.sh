#!/usr/bin/env bash
# Generates .env / backend/.env for a GitHub Codespaces demo run.
#
# This is NOT the production path. It targets the plain docker-compose.yml
# (MailHog visible, no secret hardening) so a reviewer can see the whole app
# working end to end with zero manual configuration. For an always-on public
# deployment with real secrets and the hardening overlay, see DEPLOY.md.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

[ -f .env ] || cp .env.example .env
[ -f backend/.env ] || cp backend/.env.example backend/.env

# Codespaces forwards port 80 at:
#   https://<codespace-name>-80.<port-forwarding-domain>
# Neither half is known until the codespace actually exists, so this can't be
# baked into .env.example — it's computed here instead. Both variables are
# set automatically by Codespaces; app.github.dev is the fallback for an
# older environment that doesn't export the domain variable.
domain="${GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN:-app.github.dev}"
codespace_url="https://${CODESPACE_NAME}-80.${domain}"

echo "Codespace public URL: ${codespace_url}"

set_var() {
  local file="$1" key="$2" value="$3"
  if grep -q "^${key}=" "$file"; then
    sed -i "s|^${key}=.*|${key}=\"${value}\"|" "$file"
  else
    echo "${key}=\"${value}\"" >> "$file"
  fi
}

set_var .env BASE_URL "$codespace_url"
set_var .env CORS_ALLOWED_ORIGINS "$codespace_url"

echo "Wrote BASE_URL and CORS_ALLOWED_ORIGINS into .env."
echo "Remember: in the Ports tab, port 80 must be set to Public before the"
echo "link above is reachable by anyone other than you."
