#!/bin/bash
# Registers this container as a self-hosted GitHub Actions runner, then
# starts it. Required env vars:
#   REPO_URL       e.g. https://github.com/<owner>/<repo>
#   RUNNER_TOKEN   a short-lived registration token from
#                  Settings > Actions > Runners > New self-hosted runner
set -euo pipefail

if [ -z "${REPO_URL:-}" ] || [ -z "${RUNNER_TOKEN:-}" ]; then
  echo "REPO_URL and RUNNER_TOKEN must be set." >&2
  exit 1
fi

./config.sh --unattended \
  --url "${REPO_URL}" \
  --token "${RUNNER_TOKEN}" \
  --name "${RUNNER_NAME:-simulated-vm}" \
  --labels "self-hosted,simulated-vm" \
  --replace

cleanup() {
  ./config.sh remove --unattended --token "${RUNNER_TOKEN}" || true
}
trap cleanup EXIT

./run.sh
