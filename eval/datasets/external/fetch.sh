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
  local out="$DATA_DIR/longmemeval_s.json"
  if [[ -s "$out" ]]; then
    echo "longmemeval: $out already exists, skipping"
    return
  fi
  # LongMemEval data is distributed via Hugging Face. longmemeval_s (~40 MB)
  # is the "small" haystack variant: ~115k tokens of history per instance.
  echo "longmemeval: downloading longmemeval_s.json from Hugging Face (~40 MB)..."
  if ! curl -fL --retry 3 -o "$out" \
    "https://huggingface.co/datasets/xiaowu0162/longmemeval/resolve/main/longmemeval_s.json"; then
    cat >&2 <<'EOF'
longmemeval: automatic download failed.

The dataset may require accepting terms on Hugging Face. Fetch it manually:
  1. Visit https://huggingface.co/datasets/xiaowu0162/longmemeval
     (or the Google Drive link in https://github.com/xiaowu0162/LongMemEval)
  2. Download longmemeval_s.json (or longmemeval_oracle.json for a small,
     evidence-only variant that is much cheaper to run)
  3. Place it at eval/datasets/external/data/longmemeval_s.json
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
