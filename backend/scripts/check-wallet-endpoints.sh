#!/usr/bin/env bash
#
# Cek seluruh endpoint halaman Wallet pada server yang benar-benar berjalan.
#
# Skrip ini:
#   1. build ./cmd/api,
#   2. menyalin data/hotwallet.db ke direktori sementara (DB asli tidak disentuh),
#   3. menjalankan server di port kosong,
#   4. memanggil endpoint halaman Wallet lalu mencetak PASS/FAIL,
#   5. mematikan server dan menyisakan log + DB salinan untuk diperiksa.
#
# Pemakaian:
#   scripts/check-wallet-endpoints.sh                 # hanya baca + estimasi + refresh
#   scripts/check-wallet-endpoints.sh --mutate        # + rotate (hanya di DB salinan)
#   CONFIRM_SEND=yes scripts/check-wallet-endpoints.sh --with-send
#                                                     # + broadcast nyata (biaya gas!)
#
# --with-send mengirim nominal kecil dari wallet pertama di BNB Chain ke alamat
# burn. Butuh CONFIRM_SEND=yes dan hanya dijalankan bila Anda benar-benar mau
# membelanjakan saldo. Secara default tidak ada transaksi yang disiarkan.
#
# Endpoint default yang sengaja tidak diuji: /settings/rpc (masih 501),
# PUT /wallets/{id}/address dan DELETE /wallets/{id} (butuh private key /
# menghapus wallet; tidak dipakai halaman Wallet).
#
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT" || exit 1

MUTATE=0
WITH_SEND=0
KEEP=0
for arg in "$@"; do
  case "$arg" in
    --mutate)    MUTATE=1 ;;
    --with-send) WITH_SEND=1 ;;
    --keep)      KEEP=1 ;;
    -h|--help)   sed -n '3,30p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "argumen tidak dikenal: $arg (lihat --help)" >&2; exit 2 ;;
  esac
done

for bin in curl python3 go; do
  command -v "$bin" >/dev/null 2>&1 || { echo "butuh '$bin' di PATH" >&2; exit 2; }
done
[ -f data/hotwallet.db ] || { echo "data/hotwallet.db tidak ditemukan" >&2; exit 2; }

WORK="${WORK:-$(mktemp -d "${TMPDIR:-/tmp}/hotwallet-check.XXXXXX")}"
mkdir -p "$WORK/data"
cp data/hotwallet.db "$WORK/hotwallet.db"
rm -f "$WORK/hotwallet.db-shm" "$WORK/hotwallet.db-wal"

pick_port() {
  python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
}
PORT="${PORT:-$(pick_port)}"
BASE="http://127.0.0.1:$PORT/api/v1"

echo "== build =="
if ! go build -o "$WORK/hotwallet-api" ./cmd/api; then
  echo "build gagal" >&2
  exit 1
fi

echo "== start server (port $PORT, db salinan) =="
PORT="$PORT" DB_PATH="$WORK/hotwallet.db" DATA_DIR="$WORK/data" \
  "$WORK/hotwallet-api" >"$WORK/server.log" 2>&1 &
SERVER_PID=$!

cleanup() {
  kill "$SERVER_PID" 2>/dev/null
  wait "$SERVER_PID" 2>/dev/null
}
trap cleanup EXIT

READY=0
for _ in $(seq 1 60); do
  if curl -s --max-time 1 "$BASE/health" >/dev/null 2>&1; then READY=1; break; fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then break; fi
  sleep 0.25
done
if [ "$READY" -ne 1 ]; then
  echo "server tidak siap; log terakhir:" >&2
  tail -30 "$WORK/server.log" >&2
  exit 1
fi

PASS=0
FAIL=0
FAILED=()
TIMEOUT="${TIMEOUT:-60}"

# request METHOD PATH [BODY] -> cetak HTTP code, simpan body di $WORK/last.json
request() {
  local method="$1" path="$2" body="${3:-}"
  if [ -n "$body" ]; then
    curl -sS --max-time "$TIMEOUT" -o "$WORK/last.json" -w '%{http_code}' \
      -X "$method" -H 'Content-Type: application/json' --data "$body" "$BASE$path" 2>"$WORK/curl.err" ||
      echo "000"
  else
    curl -sS --max-time "$TIMEOUT" -o "$WORK/last.json" -w '%{http_code}' \
      -X "$method" "$BASE$path" 2>"$WORK/curl.err" || echo "000"
  fi
}

envelope_ok() {
  python3 - "$WORK/last.json" <<'PY'
import json, sys
try:
    with open(sys.argv[1]) as fh:
        doc = json.load(fh)
except Exception:
    sys.exit(1)
sys.exit(0 if doc.get("success") is True else 1)
PY
}

show_body() {
  python3 -m json.tool "$WORK/last.json" 2>/dev/null | head -40 || head -c 800 "$WORK/last.json"
  if [ -s "$WORK/curl.err" ]; then
    sed 's/^/    curl: /' "$WORK/curl.err" | head -5
  fi
}

check() {
  local method="$1" path="$2" want="$3" body="${4:-}"
  local code extra=""
  code="$(request "$method" "$path" "$body")"
  [ "$code" = "000" ] && [ -s "$WORK/curl.err" ] && extra=" $(head -1 "$WORK/curl.err")"

  local ok=1
  [ "$code" = "$want" ] || ok=0
  case "$want" in
    # /health tidak memakai envelope {success,...}.
    2*) if [ "$path" != "/health" ] && ! envelope_ok; then ok=0; fi ;;
  esac

  if [ "$ok" -eq 1 ]; then
    PASS=$((PASS + 1))
    printf 'PASS  %s  %-4s %s\n' "$code" "$method" "$path"
  else
    FAIL=$((FAIL + 1))
    FAILED+=("$method $path (HTTP $code, diharapkan $want)$extra")
    printf 'FAIL  %s  %-4s %s  (diharapkan %s%s)\n' "$code" "$method" "$path" "$want" "$extra"
    if [ "$code" != "000" ]; then
      show_body | sed 's/^/    /'
    fi
  fi
}

# jget <file> <dotted.path> -> nilai skalar, kosong bila tidak ada
jget() {
  python3 - "$1" "$2" <<'PY'
import json, sys
try:
    with open(sys.argv[1]) as fh:
        cur = json.load(fh)
except Exception:
    print(""); raise SystemExit
for part in sys.argv[2].split("."):
    if isinstance(cur, list):
        if not part.isdigit() or int(part) >= len(cur):
            print(""); raise SystemExit
        cur = cur[int(part)]
    elif isinstance(cur, dict):
        cur = cur.get(part)
    else:
        print(""); raise SystemExit
    if cur is None:
        print(""); raise SystemExit
print(cur if not isinstance(cur, (dict, list)) else json.dumps(cur))
PY
}

wallet_id_for() { # network_key
  python3 - "$WORK/wallets.json" "$1" <<'PY'
import json, sys
try:
    doc = json.load(open(sys.argv[1]))
except Exception:
    raise SystemExit
for wallet in doc.get("data") or []:
    if wallet.get("network_key") == sys.argv[2]:
        print(wallet.get("id", ""))
        break
PY
}

wallet_ids() {
  python3 - "$WORK/wallets.json" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
for wallet in doc.get("data") or []:
    print(wallet.get("id", ""))
PY
}

wallet_address() { # wallet_id
  python3 - "$WORK/wallets.json" "$1" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
for wallet in doc.get("data") or []:
    if wallet.get("id") == sys.argv[2]:
        print(wallet.get("address", ""))
        break
PY
}

echo
echo "== endpoint umum =="
check GET /health 200
check GET /wallets 200
cp "$WORK/last.json" "$WORK/wallets.json"
check GET /wallets/overview 200
check GET /wallets/addresses 200
check GET /wallets/networks 200
check GET /contacts 200
check GET /settings/security 200

echo
echo "== diagnostics =="
check GET /wallets/diagnostics 200
python3 - "$WORK/last.json" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
data = doc.get("data") or {}
print(f"    ready {data.get('ready')}/{data.get('total')}")
for net in data.get("networks") or []:
    state = "ok" if net.get("reachable") and net.get("chain_id_ok") else "GAGAL"
    print(f"    {net.get('network_key'):<10} {net.get('transport'):<16} {state:<6}"
          f" saldo={net.get('native_balance')} {net.get('native_symbol')}"
          f" gas_ready={net.get('gas_ready')}"
          + (f" error={net.get('error')}" if net.get("error") else ""))
PY

echo
echo "== wallet pertama: route statis vs parameter =="
check GET /wallets/tidak-ada 404

echo
echo "== per wallet =="
IDS="$(wallet_ids)"
if [ -z "$IDS" ]; then
  echo "FAIL  tidak ada wallet di database; isi private key di .env lalu jalankan ulang"
  FAIL=$((FAIL + 1))
  FAILED+=("tidak ada wallet")
fi

for id in $IDS; do
  [ -n "$id" ] || continue
  echo "  - $id"
  check GET "/wallets/$id" 200
  check GET "/wallets/$id/balance" 200
  check GET "/wallets/$id/receive" 200

  dest="$(wallet_address "$id")"
  if [ -n "$dest" ]; then
    # Estimasi native ke alamat sendiri: tidak butuh saldo, tidak menyiarkan.
    check POST "/wallets/$id/send/estimate" 200 \
      "{\"to\":\"$dest\",\"amount\":\"0.001\"}"
  fi
done

echo
echo "== refresh =="
for id in $IDS; do
  [ -n "$id" ] || continue
  check POST "/wallets/$id/refresh" 200
done
check POST /wallets/refresh 200

if [ "$MUTATE" -eq 1 ]; then
  echo
  echo "== mutate (hanya DB salinan) =="
  rotate_id="$(wallet_id_for ethereum)"
  [ -n "$rotate_id" ] || rotate_id="$(wallet_id_for solana)"
  if [ -z "$rotate_id" ]; then
    echo "FAIL  tidak ada wallet EVM/Solana untuk rotate"
    FAIL=$((FAIL + 1))
    FAILED+=("rotate: tidak ada wallet")
  else
    before="$(wallet_address "$rotate_id")"
    check POST "/wallets/$rotate_id/rotate" 200
    after="$(jget "$WORK/last.json" data.new_address)"
    if [ -n "$after" ] && [ "$after" != "$before" ]; then
      PASS=$((PASS + 1))
      printf 'PASS  %-4s  %s\n' "-" "alamat berubah: $before -> $after (index $(jget "$WORK/last.json" data.derivation_index))"
    else
      FAIL=$((FAIL + 1))
      FAILED+=("rotate: alamat tidak berubah")
      printf 'FAIL  %-4s  %s\n' "-" "alamat tidak berubah ($before)"
    fi
  fi
fi

if [ "$WITH_SEND" -eq 1 ]; then
  echo
  echo "== broadcast nyata (BNB Chain) =="
  if [ "${CONFIRM_SEND:-}" != "yes" ]; then
    echo "SKIP  butuh CONFIRM_SEND=yes untuk mengirim transaksi sungguhan"
  else
    send_id="$(wallet_id_for bsc)"
    if [ -z "$send_id" ]; then
      echo "FAIL  wallet BNB Chain tidak ditemukan"
      FAIL=$((FAIL + 1))
      FAILED+=("send: wallet bsc tidak ada")
    else
      code="$(request POST "/wallets/$send_id/send" \
        '{"to":"0x000000000000000000000000000000000000dEaD","amount":"0.0001"}')"
      msg="$(jget "$WORK/last.json" error)"
      if [ "$code" = "200" ]; then
        PASS=$((PASS + 1))
        printf 'PASS  %s  POST /wallets/%s/send  tx=%s\n' "$code" "$send_id" "$(jget "$WORK/last.json" data.tx_hash)"
      elif [ "$code" = "400" ] && printf '%s' "$msg" | grep -qi "insufficient funds"; then
        # Node menolak karena saldo 0: jalur build -> sign -> broadcast terbukti.
        PASS=$((PASS + 1))
        printf 'PASS  %s  POST /wallets/%s/send  (ditolak node: %s)\n' "$code" "$send_id" "$msg"
      else
        FAIL=$((FAIL + 1))
        FAILED+=("send: HTTP $code $msg")
        printf 'FAIL  %s  POST /wallets/%s/send\n' "$code" "$send_id"
        show_body | sed 's/^/    /'
      fi
    fi
  fi
fi

echo
echo "------------------------------------------------------------------------"
printf 'Ringkasan: %d lulus, %d gagal\n' "$PASS" "$FAIL"
if [ "$FAIL" -ne 0 ]; then
  printf 'Gagal:\n'
  for item in "${FAILED[@]}"; do printf '  - %s\n' "$item"; done
fi
echo "Log server : $WORK/server.log"
echo "DB salinan : $WORK/hotwallet.db"
if [ "$KEEP" -eq 0 ]; then
  echo "(direktori $WORK disimpan untuk debugging; hapus manual bila tidak perlu)"
fi

[ "$FAIL" -eq 0 ]
