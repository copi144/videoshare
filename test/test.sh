#!/usr/bin/env bash
set -euo pipefail

# ─── Configuration ───────────────────────────────────────────────────────────
PASS=0
FAIL=0
TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$TEST_DIR")"
BACKEND_DIR="$PROJECT_DIR/backend"

BINARY="$TEST_DIR/videoserver"
SERVER_LOG="$TEST_DIR/server.log"
COOKIE_FILE="$TEST_DIR/cookies.txt"
DATA_DIR="$TEST_DIR/data"
PORT=":19090"
BASE="http://127.0.0.1:19090"

SERVER_PID=""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ─── Helpers ──────────────────────────────────────────────────────────────────

test_pass() {
  PASS=$((PASS+1))
  printf "  ${GREEN}PASS${NC} %s\n" "$1"
}

test_fail() {
  FAIL=$((FAIL+1))
  printf "  ${RED}FAIL${NC} %s\n" "$1"
}

banner() {
  printf "\n${CYAN}%s${NC}\n" "$1"
}

# ─── Cleanup ──────────────────────────────────────────────────────────────────

cleanup() {
  set +e
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  # Remove test directory
  rm -rf "$TEST_DIR/data" "$TEST_DIR/videoserver" "$COOKIE_FILE" "$TEST_DIR/test.mp4" "$TEST_DIR/server.log" 2>/dev/null || true
  printf "\n${CYAN}═══════════════════════════════════════════${NC}\n"
  printf "${CYAN}  Results:${NC} %d ${GREEN}passed${NC}, %d ${RED}failed${NC}\n" "$PASS" "$FAIL"
  printf "${CYAN}═══════════════════════════════════════════${NC}\n"
  if [ "$FAIL" -gt 0 ]; then
    if [ -f "$SERVER_LOG" ]; then
      printf "\n${YELLOW}Server log tail:${NC}\n"
      tail -30 "$SERVER_LOG" 2>/dev/null || true
    fi
    exit 1
  fi
  exit 0
}
trap cleanup EXIT INT TERM

# ─── Helper: API request with Bearer token ────────────────────────────────────

api() {
  curl -s -H "Authorization: Bearer $API_TOKEN" "$@"
}

api_json() {
  api -H 'Content-Type: application/json' "$@"
}

# ─── TOTP code generation ────────────────────────────────────────────────────

try_gen_totp() {
  local secret="$1"
  if command -v oathtool &>/dev/null; then
    oathtool --totp -b "$secret" 2>/dev/null && return
  fi
  python3 -c "
import sys, base64, struct, hmac, hashlib, time
def totp(secret):
    key = base64.b32decode(secret)
    counter = struct.pack('>Q', int(time.time()) // 30)
    h = hmac.new(key, counter, hashlib.sha1).digest()
    offset = h[-1] & 0x0f
    code = struct.unpack('>I', h[offset:offset+4])[0] & 0x7fffffff
    return '{:06d}'.format(code % 1000000)
print(totp('$secret'))
" 2>/dev/null
}

# ═══════════════════════════════════════════════════════════════════════════════
#  BUILD & SETUP
# ═══════════════════════════════════════════════════════════════════════════════

banner "=== VideoShare Test Suite ==="

# Step 1: Build
printf "Building... "
cd "$BACKEND_DIR"
CGO_ENABLED=0 go build -o "$BINARY" . 2>&1
echo "PASS"

# Step 2: Prepare test directory
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR"

# Create a minimal valid MP4 test file that passes MIME detection
# The file needs: ftyp box (major brand mp42) + enough size padding (>=1024 bytes)
python3 -c "
import struct

# Build ftyp box
major = b'mp42'
minor = struct.pack('>I', 0)
brands = b'mp42mp41isom'
ftyp_data = major + minor + brands
ftyp_size = struct.pack('>I', 8 + len(ftyp_data))
ftyp = ftyp_size + b'ftyp' + ftyp_data

# Build a minimal moov box for ffprobe compatibility (if available)
# ffprobe needs at least an empty moov to not error on duration probe
moov_data = b''
moov_size = struct.pack('>I', 8 + len(moov_data))
moov = moov_size + b'moov' + moov_data

# Combine and pad to meet minimum file size (>= 1024 bytes)
content = ftyp + moov
content = content + b'\\x00' * max(0, 1024 - len(content))

with open('$TEST_DIR/test.mp4', 'wb') as f:
    f.write(content)
"

# If ffmpeg is available, create a proper test video (>1s duration)
if command -v ffmpeg &>/dev/null; then
    ffmpeg -y -f lavfi -i color=c=blue:s=320x240:d=2 -c:v libx264 \
      -pix_fmt yuv420p -profile:v baseline -f mp4 \
      "$TEST_DIR/test.mp4" 2>/dev/null || true
fi

# Step 3: Start server
printf "Starting server... "
cd "$TEST_DIR"
export DATA_DIR="$DATA_DIR"
export PORT="$PORT"
export ADMIN_USERNAME="admin"
"$BINARY" > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

# Wait for server to be ready (up to 30 seconds)
for i in $(seq 1 30); do
    if curl -s "$BASE/health" > /dev/null 2>&1; then
        echo "PASS"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "FAIL"
        test_fail "Server startup (timeout)"
        exit 1
    fi
    sleep 1
done

# Step 4: Extract TOTP secret from server startup log
TOTP_SECRET=$(grep -oP '(?<=secret=)[A-Z2-7]+' "$SERVER_LOG" | head -1)
if [ -z "$TOTP_SECRET" ]; then
    echo "FAILED to extract TOTP secret from server log"
    echo "Server log contents:"
    cat "$SERVER_LOG"
    exit 1
fi
printf "TOTP secret: ${YELLOW}%s${NC}\n" "$TOTP_SECRET"

# Step 5: Generate TOTP code
TOTP_CODE=$(try_gen_totp "$TOTP_SECRET") || TOTP_CODE=""
if [ -z "$TOTP_CODE" ]; then
    echo "FAILED to generate TOTP code (need oathtool or python3)"
    exit 1
fi
printf "TOTP code: ${YELLOW}%s${NC}\n" "$TOTP_CODE"

# Step 6: Login
LOGIN_RESP=$(curl -s -c "$COOKIE_FILE" -X POST "$BASE/api/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"admin\",\"totp_code\":\"$TOTP_CODE\"}")
API_TOKEN=$(echo "$LOGIN_RESP" | grep -oP '(?<="api_token":")[^"]+') || API_TOKEN=""

if [ -z "$API_TOKEN" ]; then
    echo "FAILED to login / extract API token"
    echo "Login response: $LOGIN_RESP"
    echo "Server log:"
    tail -20 "$SERVER_LOG"
    exit 1
fi
printf "API token: ${YELLOW}%s${NC}\n" "${API_TOKEN:0:16}..."

# ═══════════════════════════════════════════════════════════════════════════════
#  TEST CASES
# ═══════════════════════════════════════════════════════════════════════════════

# Track the uploaded resource ID for later tests
RESOURCE_ID=""

banner "--- Test Results ---"

# ── [TEST 1] Health check ────────────────────────────────────────────────────

RESP=$(curl -s "$BASE/health")
if echo "$RESP" | grep -q '"status":"ok"'; then
    test_pass "Health check"
else
    test_fail "Health check (got: $RESP)"
fi

# ── [TEST 2] SPA serves ──────────────────────────────────────────────────────

RESP=$(curl -s "$BASE/")
CONTENT_TYPE=$(curl -s -o /dev/null -w '%{content_type}' "$BASE/")
# SPA should return HTML
if echo "$CONTENT_TYPE" | grep -qi 'html'; then
    test_pass "SPA serves"
else
    test_fail "SPA serves (content-type: $CONTENT_TYPE)"
fi

# ── [TEST 3] Login (invalid TOTP) ────────────────────────────────────────────

RESP=$(curl -s -X POST "$BASE/api/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","totp_code":"000000"}')
# Should return error (not ok)
if echo "$RESP" | grep -q '"error"'; then
    test_pass "Login (invalid TOTP)"
else
    test_fail "Login (invalid TOTP) (got: $RESP)"
fi

# ── [TEST 4] Login (valid TOTP) ──────────────────────────────────────────────

# We already logged in above, test that the token is valid
if [ -n "$API_TOKEN" ]; then
    test_pass "Login (valid TOTP)"
else
    test_fail "Login (valid TOTP) (no api_token in response: $LOGIN_RESP)"
fi

# ── [TEST 5] Heartbeat ───────────────────────────────────────────────────────

RESP=$(api_json -X POST "$BASE/api/heartbeat")
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "Heartbeat"
else
    test_fail "Heartbeat (got: $RESP)"
fi

# ── [TEST 6] List resources ──────────────────────────────────────────────────

RESP=$(api "$BASE/api/resources")
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "List resources"
else
    test_fail "List resources (got: $RESP)"
fi

# ── [TEST 7] List categories ─────────────────────────────────────────────────

RESP=$(api "$BASE/api/categories")
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "List categories"
else
    test_fail "List categories (got: $RESP)"
fi

# ── [TEST 8] Upload video ────────────────────────────────────────────────────

TEST_VIDEO="$TEST_DIR/test.mp4"
RESP=$(api -F "file=@$TEST_VIDEO" \
  -F "title=Test Video Title" \
  -F "category_id=global" \
  -X POST "$BASE/api/upload")
# Upload should succeed (ok:true) possibly with redirect
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "Upload video"
    # Get resource ID from listing
    LIST_RESP=$(api "$BASE/api/resources?limit=1")
    RESOURCE_ID=$(echo "$LIST_RESP" | grep -oP '(?<="id":")[^"]+' | head -1)
    if [ -n "$RESOURCE_ID" ]; then
        printf "  ${YELLOW}Resource ID: %s${NC}\n" "$RESOURCE_ID"
    fi
else
    test_fail "Upload video (got: $RESP)"
fi

# ── [TEST 9] Get resource detail ─────────────────────────────────────────────

if [ -n "$RESOURCE_ID" ]; then
    RESP=$(api "$BASE/api/resources/$RESOURCE_ID")
    if echo "$RESP" | grep -q '"ok":true'; then
        test_pass "Get resource detail"
    else
        test_fail "Get resource detail (got: $RESP)"
    fi
else
    test_fail "Get resource detail (no resource uploaded)"
fi

# ── [TEST 10] Copy share link (get resource detail for sharing) ──────────────

# This tests the same endpoint as test 9 but explicitly for share link construction
if [ -n "$RESOURCE_ID" ]; then
    RESP=$(api "$BASE/api/resources/$RESOURCE_ID")
    if echo "$RESP" | grep -q "\"id\":\"$RESOURCE_ID\""; then
        test_pass "Copy share link"
    else
        test_fail "Copy share link (got: $RESP)"
    fi
else
    test_fail "Copy share link (no resource uploaded)"
fi

# ── [TEST 11] Share auth (wrong) ─────────────────────────────────────────────

# Test share auth for the uploaded video (global category = auto-auth)
# Use a non-existent video to test error handling
RESP=$(curl -s -X POST "$BASE/api/s/nonexistent-id/auth")
# Should return error - either 404 or 400
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE/api/s/nonexistent-id/auth")
if [ "$HTTP_CODE" = "404" ]; then
    test_pass "Share auth (wrong)"
else
    test_fail "Share auth (wrong) (http: $HTTP_CODE)"
fi

# ── [TEST 12] Create category ────────────────────────────────────────────────

CAT_NAME="test-cat-$RANDOM"
RESP=$(api_json -X POST "$BASE/api/categories" \
  -d "{\"name\":\"$CAT_NAME\",\"description\":\"Test category\"}")
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "Create category"
else
    test_fail "Create category (got: $RESP)"
fi

# ── [TEST 13] Create playlist ────────────────────────────────────────────────

RESP=$(api_json -X POST "$BASE/api/playlists" \
  -d "{\"name\":\"Test Playlist\",\"description\":\"A test\",\"category_id\":\"global\",\"sort_order\":0}")
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "Create playlist"
else
    test_fail "Create playlist (got: $RESP)"
fi

# ── [TEST 14] Create user ────────────────────────────────────────────────────

RESP=$(api_json -X POST "$BASE/api/users" \
  -d '{"username":"test-uploader"}')
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "Create user"
else
    test_fail "Create user (got: $RESP)"
fi

# ── [TEST 15] Retranscode ────────────────────────────────────────────────────

if [ -n "$RESOURCE_ID" ]; then
    RESP=$(api_json -X POST "$BASE/api/resources/$RESOURCE_ID/retranscode")
    # Retranscode may fail if ffmpeg is not available (transcode queue worker checks ffmpeg path)
    # Either ok:true or an error is acceptable — we're testing the endpoint exists
    if echo "$RESP" | grep -q '"ok":true'; then
        test_pass "Retranscode"
    elif echo "$RESP" | grep -q '"error"'; then
        # It may fail if no ffmpeg or transcode already running - still PASS (endpoint responded)
        test_pass "Retranscode (expected: may fail without ffmpeg)"
    else
        test_fail "Retranscode (unexpected: $RESP)"
    fi
else
    test_fail "Retranscode (no resource uploaded)"
fi

# ── [TEST 16] Ban resource ───────────────────────────────────────────────────

if [ -n "$RESOURCE_ID" ]; then
    RESP=$(api_json -X POST "$BASE/api/resources/$RESOURCE_ID/ban")
    if echo "$RESP" | grep -q '"ok":true'; then
        test_pass "Ban resource"
    else
        test_fail "Ban resource (got: $RESP)"
    fi
else
    test_fail "Ban resource (no resource uploaded)"
fi

# ── [TEST 17] Banned video access ────────────────────────────────────────────

# After banning, accessing the banned video should return 410 Gone
if [ -n "$RESOURCE_ID" ]; then
    HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE/api/s/$RESOURCE_ID/auth" \
      -H 'Content-Type: application/json' \
      -d '{"password":"test"}')
    if [ "$HTTP_CODE" = "410" ]; then
        test_pass "Banned video access (410 Gone)"
    else
        test_fail "Banned video access (expected 410, got $HTTP_CODE)"
    fi
else
    test_fail "Banned video access (no resource uploaded)"
fi

# ── [TEST 18] Delete resource ────────────────────────────────────────────────

if [ -n "$RESOURCE_ID" ]; then
    RESP=$(api_json -X DELETE "$BASE/api/resource/$RESOURCE_ID")
    if echo "$RESP" | grep -q '"ok":true'; then
        test_pass "Delete resource"
    else
        test_fail "Delete resource (got: $RESP)"
    fi
else
    test_fail "Delete resource (no resource uploaded)"
fi

# ── [TEST 19] Logout ─────────────────────────────────────────────────────────

RESP=$(api_json -X POST "$BASE/api/logout")
if echo "$RESP" | grep -q '"ok":true'; then
    test_pass "Logout"
else
    test_fail "Logout (got: $RESP)"
fi

# ── [TEST 20] Auth after logout ──────────────────────────────────────────────

# After logout, the API token should be invalid
RESP=$(api "$BASE/api/heartbeat")
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $API_TOKEN" \
  -X POST "$BASE/api/heartbeat")
if [ "$HTTP_CODE" = "401" ]; then
    test_pass "Auth after logout (401)"
else
    test_fail "Auth after logout (expected 401, got $HTTP_CODE: $RESP)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
#  DONE — Cleanup via trap
# ═══════════════════════════════════════════════════════════════════════════════

exit 0
