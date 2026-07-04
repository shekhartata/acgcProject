#!/usr/bin/env bash
# Fetch external benchmark data files into eval/datasets/external/data/.
# The files are not vendored in the repo (size + licensing) — run this once.
#
# Usage:
#   ./eval/datasets/external/fetch.sh            # fetch everything
#   ./eval/datasets/external/fetch.sh locomo     # fetch only LoCoMo
#   ./eval/datasets/external/fetch.sh longmemeval
set -euo pipefail

DATA_DIR="$(cd "$(dirname "$0")" && pwd)/data"
mkdir -p "$DATA_DIR"

WHAT="${1:-all}"

fetch_locomo() {
  local out="$DATA_DIR/locomo10.json"
  if [[ -s "$out" ]]; then
    echo "locomo: $out already exists, skipping"
    return
  fi
  echo "locomo: downloading locomo10.json (~10 conversations, ~2k QA pairs)..."
  curl -fL --retry 3 -o "$out" \
    "https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json"
  echo "locomo: wrote $out"
}

fetch_longmemeval() {
  local out="$DATA_DIR/longmemeval_s_cleaned.json"
  if [[ -s "$out" ]]; then
    echo "longmemeval: $out already exists, skipping"
    return
  fi
  # The author deprecated the original longmemeval dataset in favor of
  # longmemeval-cleaned (removes noisy history sessions that interfere with
  # answer correctness). longmemeval_s is the "small" haystack variant:
  # ~50 sessions / ~115k history tokens per instance.
  echo "longmemeval: downloading longmemeval_s_cleaned.json from Hugging Face (~40 MB)..."
  if ! curl -fL --retry 3 -o "$out" \
    "https://huggingface.co/datasets/xiaowu0162/longmemeval-cleaned/resolve/main/longmemeval_s_cleaned.json"; then
    cat >&2 <<'EOF'
longmemeval: automatic download failed.

Fetch it manually:
  1. Visit https://huggingface.co/datasets/xiaowu0162/longmemeval-cleaned
     (or the links in https://github.com/xiaowu0162/LongMemEval)
  2. Download longmemeval_s_cleaned.json (or longmemeval_oracle.json for a
     small, evidence-only variant that is much cheaper to run)
  3. Place it at eval/datasets/external/data/longmemeval_s_cleaned.json
EOF
    rm -f "$out"
    exit 1
  fi
  echo "longmemeval: wrote $out"
}

case "$WHAT" in
  all)         fetch_locomo; fetch_longmemeval ;;
  locomo)      fetch_locomo ;;
  longmemeval) fetch_longmemeval ;;
  *) echo "usage: $0 [all|locomo|longmemeval]" >&2; exit 2 ;;
esac
