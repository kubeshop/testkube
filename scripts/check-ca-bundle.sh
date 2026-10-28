#!/usr/bin/env bash
# Report which emSign / eMudhra root CAs a container image still trusts.
#
# Works on images without a shell: the image is instantiated with `docker create`
# (not started) and files are read with `docker cp`. Only Node images are run once,
# with `--network none`, to read the roots compiled into the binary. For each image it prints
#   - entrypoint, cmd and TLS-related env (NODE_OPTIONS, NODE_EXTRA_CA_CERTS, SSL_CERT_*),
#   - OS, ca-certificates package version (apk or dpkg) and Go version of the entrypoint,
#   - the CA bundle file Go would load (SSL_CERT_FILE, else the default bundle paths),
#     every certificate in it whose subject matches emSign or eMudhra,
#   - emSign entries in /etc/ssl/certs (Go also loads that directory),
#   - for Node images, the emSign roots compiled into the Node binary.
#
# Usage: scripts/check-ca-bundle.sh <image> [<image>...]
#
# Env:
#   ALLOW='emSign Root CA - G1'  regex of emSign/eMudhra subjects still tolerated (Mozilla keeps G1)
#   STRICT=1                     fail on every non-allowed emSign root, also in Debian-derived bundles
#                                (default: Debian bundles only fail on "emSign ECC Root CA - G3",
#                                because Debian keeps C1 and ECC C3; Alpine and Node follow NSS 3.131.1
#                                and fail on G3, C1 and C3)
#   PLATFORM=linux/amd64         platform for docker create/run (default: host platform)
#   NO_PULL=1                    use the local image, do not pull
#   DOCKER=docker                docker binary
#
# Exit code: 1 when any image fails or could not be inspected (ERROR), 0 otherwise.
set -euo pipefail

DOCKER=${DOCKER:-docker}
ALLOW=${ALLOW:-'emSign Root CA - G1'}
PLATFORM_ARGS=()
[ -n "${PLATFORM:-}" ] && PLATFORM_ARGS=(--platform "$PLATFORM")

if [ $# -lt 1 ]; then
  sed -n '2,26p' "$0"
  exit 2
fi

tmp=$(mktemp -d)
cid=""
# shellcheck disable=SC2329  # invoked by the EXIT trap
cleanup() {
  if [ -n "$cid" ]; then "$DOCKER" rm -v "$cid" >/dev/null 2>&1 || true; fi
  rm -rf "$tmp"
}
trap cleanup EXIT
summary=()
failed=0

# cp_file <cid> <path>: print the file content; returns 1 when absent, 2 when the read failed.
cp_file() {
  local rc=0
  "$DOCKER" cp -L "$1:$2" - 2>"$tmp/cp.err" | tar -xO 2>/dev/null || rc=$?
  [ "$rc" = 0 ] && return 0
  cp_missing && return 1
  return 2
}

# cp_missing: true when the last docker cp failed only because the path does not exist.
cp_missing() { grep -q 'Could not find the file' "$tmp/cp.err"; }

# judge <subject> <strict 0|1>: prints ok/warn/BAD and returns 1 for BAD.
judge() {
  local s=$1 strict=$2
  if printf '%s' "$s" | grep -Eq "$ALLOW"; then
    echo "  ok    $s"
  elif [ "$strict" = 0 ] && ! printf '%s' "$s" | grep -q 'ECC Root CA - G3'; then
    echo "  warn  $s (kept by Debian upstream; set STRICT=1 to fail)"
  else
    echo "  BAD   $s"
    return 1
  fi
}

# mark_fail: a BAD root sets FAIL unless the image already hit an ERROR (status is check_image's).
mark_fail() { [ "$status" = ERROR ] || status=FAIL; }

check_image() {
  local img=$1 status=PASS flavor=debian strict=${STRICT:-0} count=0
  echo "=== $img"
  if [ -z "${NO_PULL:-}" ]; then
    if ! "$DOCKER" pull -q ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} "$img" >/dev/null 2>&1; then
      echo "  ERROR: pull failed"; summary+=("$img|ERROR|pull failed"); failed=1; return
    fi
  fi
  if ! cid=$("$DOCKER" create ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} "$img" /nonexistent 2>/dev/null); then
    echo "  ERROR: docker create failed"; summary+=("$img|ERROR|create failed"); failed=1; return
  fi

  # How the service starts, read from the image for the requested platform (the container's
  # cmd is the /nonexistent placeholder; `image inspect --platform` needs Docker 28.1+).
  echo "  entrypoint: $("$DOCKER" image inspect ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} --format '{{json .Config.Entrypoint}}' "$img")  cmd: $("$DOCKER" image inspect ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} --format '{{json .Config.Cmd}}' "$img")"
  local env
  env=$("$DOCKER" inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$cid" \
    | grep -E '^(NODE_OPTIONS|NODE_EXTRA_CA_CERTS|NODE_USE_SYSTEM_CA|SSL_CERT_FILE|SSL_CERT_DIR|NODE_VERSION)=|use-(openssl|system)-ca' || true)
  [ -n "$env" ] && printf '%s\n' "$env" | sed 's/^/  env: /'

  # OS and package versions.
  local os apk deb
  os=$(cp_file "$cid" /etc/os-release | sed -n 's/^PRETTY_NAME=//p' | tr -d '"' || true)
  [ -n "$os" ] && echo "  os: $os"
  apk=$(cp_file "$cid" /lib/apk/db/installed | awk '/^P:ca-certificates(-bundle)?$/{p=substr($0,3)} /^V:/{if(p){printf "%s %s; ", p, substr($0,3); p=""}}' || true)
  deb=$(cp_file "$cid" /var/lib/dpkg/status | awk '/^Package: ca-certificates$/{f=1} f&&/^Version:/{print $2; exit}' || true)
  [ -z "$deb" ] && deb=$(cp_file "$cid" /var/lib/dpkg/status.d/ca-certificates | awk '/^Version:/{print $2; exit}' || true)
  if [ -n "$apk" ]; then echo "  apk: $apk"; flavor=alpine; strict=1; fi
  [ -n "$deb" ] && echo "  deb: ca-certificates $deb"

  # Go version of the entrypoint binary (from its build info).
  local ep
  ep=$("$DOCKER" inspect --format '{{if .Config.Entrypoint}}{{index .Config.Entrypoint 0}}{{end}}' "$cid" 2>/dev/null || true)
  if [ -n "$ep" ] && command -v go >/dev/null && cp_file "$cid" "$ep" >"$tmp/bin" && [ -s "$tmp/bin" ]; then
    local gv
    gv=$(go version "$tmp/bin" 2>/dev/null | awk '{print $2}' || true)
    [ -n "$gv" ] && echo "  go: $gv ($ep)"
  fi

  # Bundle file, the way Go picks it: SSL_CERT_FILE first, then the default list.
  local file found="" p hits="" rc unread=""
  file=$(printf '%s\n' "$env" | sed -n 's/^SSL_CERT_FILE=//p')
  for p in ${file:+"$file"} /etc/ssl/certs/ca-certificates.crt /etc/pki/tls/certs/ca-bundle.crt /etc/ssl/ca-bundle.pem /etc/pki/tls/cacert.pem /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/cert.pem; do
    rc=0; cp_file "$cid" "$p" >"$tmp/bundle.pem" || rc=$?
    if [ "$rc" = 0 ] && [ -s "$tmp/bundle.pem" ]; then found=$p; break; fi
    [ "$rc" = 2 ] && unread+=" $p"
  done
  # A path that exists but could not be copied must not read as "no bundle".
  if [ -n "$unread" ]; then
    echo "  ERROR: could not read$unread: $(tr '\n' ' ' <"$tmp/cp.err")"
    status=ERROR
  fi
  if [ -n "$found" ]; then
    count=$(grep -c 'BEGIN CERTIFICATE' "$tmp/bundle.pem" || true)
    echo "  bundle: $found ($count certs)"
    # A parse failure must not look like a clean bundle: report ERROR instead.
    local subjects parsed=1
    subjects=$(openssl crl2pkcs7 -nocrl -certfile "$tmp/bundle.pem" 2>"$tmp/openssl.err" \
      | openssl pkcs7 -print_certs -noout 2>>"$tmp/openssl.err" \
      | sed -n 's/^subject=//p') || parsed=0
    if [ "$parsed" = 0 ] || { [ "$count" -gt 0 ] && [ -z "$subjects" ]; }; then
      echo "  ERROR: openssl could not parse the bundle: $(tr '\n' ' ' <"$tmp/openssl.err")"
      status=ERROR
    else
      hits=$(printf '%s\n' "$subjects" | grep -iE 'emSign|eMudhra' | sort -u || true)
    fi
  elif [ -z "$unread" ]; then
    echo "  bundle: none (no CA bundle in this image)"
  fi
  if [ -n "$hits" ]; then
    while IFS= read -r s; do judge "$s" "$strict" || mark_fail; done <<<"$hits"
  elif [ -n "$found" ] && [ "$status" != ERROR ]; then
    echo "  ok    no emSign/eMudhra subjects in the bundle"
  fi

  # Per-certificate entries in /etc/ssl/certs (Go loads every file there too).
  local dirlist dirhits="" dirrc=0
  dirlist=$("$DOCKER" cp "$cid:/etc/ssl/certs" - 2>"$tmp/cp.err" | tar -t 2>/dev/null) || dirrc=$?
  if [ "$dirrc" != 0 ] && ! cp_missing; then
    echo "  ERROR: could not read /etc/ssl/certs: $(tr '\n' ' ' <"$tmp/cp.err")"
    status=ERROR
  elif [ "$dirrc" = 0 ]; then
    dirhits=$(printf '%s\n' "$dirlist" | grep -iE 'emsign|emudhra' | sed 's#^certs/##' | sort -u || true)
  fi
  if [ -n "$dirhits" ]; then
    while IFS= read -r f; do
      judge "$(printf '%s' "$f" | sed 's/\.pem$//; s/\.crt$//; s/_/ /g') (/etc/ssl/certs/$f)" "$strict" || mark_fail
    done <<<"$dirhits"
  fi

  # Node: roots compiled into the binary (PEM text never contains "emSign", parse the subject).
  rc=0; cp_file "$cid" /usr/local/bin/node >/dev/null 2>&1 || rc=$?
  if [ "$rc" = 2 ]; then
    echo "  ERROR: could not read /usr/local/bin/node"
    status=ERROR
  elif [ "$rc" = 0 ]; then
    local js out nodehits
    js='const {X509Certificate}=require("crypto");const tls=require("tls");
const subj=c=>{try{return new X509Certificate(c).subject.replace(/\n/g,", ")}catch(e){return "?"}};
const hit=l=>l.map(subj).filter(s=>/emSign|eMudhra/i.test(s));
console.log("node "+process.version+", bundled roots: "+tls.rootCertificates.length);
for(const s of hit(tls.rootCertificates))console.log("NODEHIT "+s);
if(tls.getCACertificates){const d=tls.getCACertificates("default");console.log("tls.getCACertificates(default): "+d.length);for(const s of hit(d))console.log("NODEHIT "+s)}'
    local noderc=0
    out=$("$DOCKER" run --rm --network none ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} --entrypoint node "$img" -e "$js" 2>&1) || noderc=$?
    # A probe that did not run (wrong PLATFORM, broken image) is an ERROR, not a clean store.
    if [ "$noderc" != 0 ] || ! printf '%s\n' "$out" | grep -q 'bundled roots: '; then
      echo "  ERROR: node probe failed (rc=$noderc): $(printf '%s' "$out" | head -3 | tr '\n' ' ')"
      status=ERROR
    else
      printf '%s\n' "$out" | grep -v '^NODEHIT ' | sed 's/^/  /'
      nodehits=$(printf '%s\n' "$out" | sed -n 's/^NODEHIT //p' | sort -u)
      if [ -n "$nodehits" ]; then
        while IFS= read -r s; do judge "$s (node built-in store)" 1 || mark_fail; done <<<"$nodehits"
      else
        echo "  ok    no emSign/eMudhra roots in the Node store"
      fi
    fi
  fi

  "$DOCKER" rm -v "$cid" >/dev/null 2>&1 || echo "  warn  could not remove container $cid (the EXIT trap retries)"
  cid=""
  echo "  result: $status"
  [ "$status" != PASS ] && failed=1
  local pkg=${apk%; }
  [ -z "$pkg" ] && pkg=${deb:+ca-certificates $deb}
  [ -z "$pkg" ] && pkg="no ca-certificates package"
  summary+=("$img|$status|$flavor, $pkg, $count certs")
}

for img in "$@"; do check_image "$img"; done

echo
echo "image|result|details"
printf '%s\n' "${summary[@]}"
exit "$failed"
