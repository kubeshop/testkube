#!/usr/bin/env bash
# Report which emSign / eMudhra root CAs a container image still trusts.
#
# Works on images without a shell: the image is instantiated with `docker create`
# (never started) and files are read with `docker cp`. For each image it prints
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
# Exit code: 1 when any image fails, 0 otherwise.
set -euo pipefail

DOCKER=${DOCKER:-docker}
ALLOW=${ALLOW:-'emSign Root CA - G1'}
PLATFORM_ARGS=()
[ -n "${PLATFORM:-}" ] && PLATFORM_ARGS=(--platform "$PLATFORM")

if [ $# -lt 1 ]; then
  sed -n '2,24p' "$0"
  exit 2
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
summary=()
failed=0

# cp_file <cid> <path>: print the file content, non-zero when absent.
cp_file() {
  "$DOCKER" cp -L "$1:$2" - 2>/dev/null | tar -xO 2>/dev/null
}

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

check_image() {
  local img=$1 cid status=PASS flavor=debian strict=${STRICT:-0} count=0
  echo "=== $img"
  if [ -z "${NO_PULL:-}" ]; then
    if ! "$DOCKER" pull -q ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} "$img" >/dev/null 2>&1; then
      echo "  ERROR: pull failed"; summary+=("$img|ERROR|pull failed"); failed=1; return
    fi
  fi
  if ! cid=$("$DOCKER" create ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} "$img" /nonexistent 2>/dev/null); then
    echo "  ERROR: docker create failed"; summary+=("$img|ERROR|create failed"); failed=1; return
  fi

  # How the service starts.
  echo "  entrypoint: $("$DOCKER" inspect --format '{{json .Config.Entrypoint}}' "$cid")  cmd: $("$DOCKER" inspect --format '{{json .Config.Cmd}}' "$cid")"
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
  local file found="" p hits=""
  file=$(printf '%s\n' "$env" | sed -n 's/^SSL_CERT_FILE=//p')
  for p in ${file:+"$file"} /etc/ssl/certs/ca-certificates.crt /etc/pki/tls/certs/ca-bundle.crt /etc/ssl/ca-bundle.pem /etc/pki/tls/cacert.pem /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/cert.pem; do
    if cp_file "$cid" "$p" >"$tmp/bundle.pem" && [ -s "$tmp/bundle.pem" ]; then found=$p; break; fi
  done
  if [ -n "$found" ]; then
    count=$(grep -c 'BEGIN CERTIFICATE' "$tmp/bundle.pem" || true)
    echo "  bundle: $found ($count certs)"
    hits=$(openssl crl2pkcs7 -nocrl -certfile "$tmp/bundle.pem" 2>/dev/null \
      | openssl pkcs7 -print_certs -noout 2>/dev/null \
      | sed -n 's/^subject=//p' | grep -iE 'emSign|eMudhra' | sort -u || true)
  else
    echo "  bundle: none (no CA bundle in this image)"
  fi
  if [ -n "$hits" ]; then
    while IFS= read -r s; do judge "$s" "$strict" || status=FAIL; done <<<"$hits"
  elif [ -n "$found" ]; then
    echo "  ok    no emSign/eMudhra subjects in the bundle"
  fi

  # Per-certificate entries in /etc/ssl/certs (Go loads every file there too).
  local dirhits
  dirhits=$("$DOCKER" cp "$cid:/etc/ssl/certs" - 2>/dev/null | tar -t 2>/dev/null | grep -iE 'emsign|emudhra' | sed 's#^certs/##' | sort -u || true)
  if [ -n "$dirhits" ]; then
    while IFS= read -r f; do
      judge "$(printf '%s' "$f" | sed 's/\.pem$//; s/\.crt$//; s/_/ /g') (/etc/ssl/certs/$f)" "$strict" || status=FAIL
    done <<<"$dirhits"
  fi

  # Node: roots compiled into the binary (PEM text never contains "emSign", parse the subject).
  if cp_file "$cid" /usr/local/bin/node >/dev/null 2>&1; then
    local js out nodehits
    js='const {X509Certificate}=require("crypto");const tls=require("tls");
const subj=c=>{try{return new X509Certificate(c).subject.replace(/\n/g,", ")}catch(e){return "?"}};
const hit=l=>l.map(subj).filter(s=>/emSign|eMudhra/i.test(s));
console.log("node "+process.version+", bundled roots: "+tls.rootCertificates.length);
for(const s of hit(tls.rootCertificates))console.log("NODEHIT "+s);
if(tls.getCACertificates){const d=tls.getCACertificates("default");console.log("tls.getCACertificates(default): "+d.length);for(const s of hit(d))console.log("NODEHIT "+s)}'
    out=$("$DOCKER" run --rm ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"} --entrypoint node "$img" -e "$js" 2>&1 || true)
    printf '%s\n' "$out" | grep -v '^NODEHIT ' | sed 's/^/  /'
    nodehits=$(printf '%s\n' "$out" | sed -n 's/^NODEHIT //p' | sort -u)
    if [ -n "$nodehits" ]; then
      while IFS= read -r s; do judge "$s (node built-in store)" 1 || status=FAIL; done <<<"$nodehits"
    else
      echo "  ok    no emSign/eMudhra roots in the Node store"
    fi
  fi

  "$DOCKER" rm -v "$cid" >/dev/null
  echo "  result: $status"
  [ "$status" = FAIL ] && failed=1
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
