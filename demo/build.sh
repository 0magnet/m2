#!/bin/bash
# Build a static demo of the m2 store into ./public
#
# Assembles a staging directory (symlinked source + demo config, example
# products, and placeholder images), runs the server there, mirrors it
# with wget, then rewrites the absolute wasm fetch() paths so the result
# works from any static host path (e.g. GitHub Pages project sites).
# The real checkout requires a server, so it is non-functional here.
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT=$PWD
PORT=9899
STAGE=$(mktemp -d)
SRV=""
# kill by port: killing the `go run` subshell leaves its compiled child alive
trap '[ -n "$SRV" ] && kill "$SRV" 2>/dev/null; fuser -sk "$PORT/tcp" 2>/dev/null; rm -rf "$STAGE"' EXIT

# --- staging dir: the live checkout is never touched -------------------
# real copies, not symlinks: go:embed (source.go) rejects irregular files
for g in "$ROOT"/*.go; do
	case "$(basename "$g")" in other.go) continue ;; esac # site drop-in stays out of the demo
	cp "$g" "$STAGE/"
done
cp "$ROOT/go.mod" "$ROOT/go.sum" "$STAGE/"
cp -r "$ROOT/vendor" "$ROOT/pkg" "$ROOT/htmpl" "$ROOT/wasm" "$STAGE/"
mkdir "$STAGE/content"
cp "$ROOT"/content/style.css "$ROOT"/content/checkout.css \
	"$ROOT"/content/checkout.html "$ROOT"/content/*.example "$STAGE/content/"
cp -r "$ROOT/demo/img" "$STAGE/img"
cp "$ROOT/demo/logo.png" "$ROOT/demo/logo.jpg" "$STAGE/"
cp "$ROOT/products.example.csv" "$STAGE/products.csv"
cp "$ROOT/demo/demo.conf" "$STAGE/demo.conf"

# --- run the server against the demo data ------------------------------
if fuser -s "$PORT/tcp" 2>/dev/null; then
	echo "port $PORT is already in use (stale demo server?)"
	exit 1
fi
(cd "$STAGE" && MENV=demo.conf go run . run) &
SRV=$!
up=""
for _ in $(seq 1 120); do
	if curl -sf "http://127.0.0.1:$PORT/" >/dev/null 2>&1 &&
		[ -f "$STAGE/cart.wasm" ]; then
		up=1
		break
	fi
	sleep 1
done
[ -n "$up" ] || { echo "server did not come up"; exit 1; }
sleep 2 # let the wasm compile log settle

# --- mirror -------------------------------------------------------------
rm -rf "$ROOT/public"
wget --mirror --page-requisites --convert-links --adjust-extension \
	--no-host-directories --directory-prefix="$ROOT/public" \
	"http://127.0.0.1:$PORT/" || true
# the wasm binaries are fetched by JS, wget cannot discover them
wget -q -O "$ROOT/public/cart.wasm" "http://127.0.0.1:$PORT/cart.wasm"

# --- make absolute fetch()/src paths relative for subpath hosting -------
find "$ROOT/public" -name '*.html' | while read -r fp; do
	rel=${fp#"$ROOT/public/"}
	depth=$(awk -F/ '{print NF-1}' <<<"$rel")
	pre="./"
	[ "$depth" -gt 0 ] && pre=$(printf '../%.0s' $(seq 1 "$depth"))
	sed -i "s|fetch(\"/|fetch(\"$pre|g" "$fp"
	# canonical/og meta still carry the local origin; strip it
	sed -i "s|https\?://127.0.0.1:$PORT||g" "$fp"
done

echo "static demo written to ./public"
