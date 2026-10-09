#!/usr/bin/env bash
# Local version of the PR security scan that runs in Testkube on every PR to main
# (Semgrep, Gitleaks, Trivy config), using the same images, versions and rules.
#
# Scans what your branch changes vs the base branch, including uncommitted and
# untracked files (a snapshot commit is created without touching your index or
# working tree). Only needs git and Docker.
#
# Env:
#   BASE_REF=origin/main      branch to compare against
#   ENFORCE=1                 exit non-zero when there are findings
#   SEMGREP_RULES=...         Semgrep rulesets (space separated)
#   GITLEAKS_IGNORE_PATHS=... path regexes Gitleaks never reports (space separated)
#   OUT=.security-scan        where JSON reports are written
set -euo pipefail

SEMGREP_IMAGE=semgrep/semgrep:1.101.0
GITLEAKS_IMAGE=zricethezav/gitleaks:v8.21.2
TRIVY_IMAGE=aquasec/trivy:0.58.1
PYTHON_IMAGE=python:3.12-alpine

SEMGREP_RULES="${SEMGREP_RULES:-p/golang}"
GITLEAKS_IGNORE_PATHS="${GITLEAKS_IGNORE_PATHS:-}"
BASE_REF="${BASE_REF:-origin/main}"
ENFORCE="${ENFORCE:-}"

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
OUT_REL="${OUT:-.security-scan}"
OUT=$(mkdir -p "$OUT_REL" && cd "$OUT_REL" && pwd -P)
rm -f "$OUT"/*.json "$OUT"/*.status "$OUT"/*.tsv "$OUT"/gitleaks.toml

command -v docker >/dev/null || { echo "security-scan: Docker is required" >&2; exit 2; }
if [ "$(git config --get remote.origin.promisor)" = "true" ]; then
  echo "WARN: partial clone detected: Semgrep cannot download missing blobs from inside Docker." >&2
  echo "      If Semgrep reports 'error', run 'git fetch --refetch origin' once or use a full clone." >&2
fi
git fetch --quiet origin "${BASE_REF#origin/}" 2>/dev/null || echo "WARN: could not fetch $BASE_REF, using the local copy"
BASE=$(git merge-base HEAD "$BASE_REF")

# Snapshot of the working tree (tracked changes + untracked, .gitignore respected) as a dangling commit.
TMP=$(mktemp -d "${TMPDIR:-/tmp}/security-scan.XXXXXX")
TMP=$(cd "$TMP" && pwd -P)
WT="$TMP/tree"
cleanup() {
  git worktree remove --force "$WT" >/dev/null 2>&1 || true
  rm -rf "$TMP"
}
trap cleanup EXIT
cp "$(git rev-parse --git-path index)" "$TMP/index"
GIT_INDEX_FILE="$TMP/index" git add -A
GIT_INDEX_FILE="$TMP/index" git rm -r -q --cached --ignore-unmatch -- "$OUT_REL"
SNAP=$(git commit-tree "$(GIT_INDEX_FILE="$TMP/index" git write-tree)" -p HEAD -m "security-scan snapshot")
git worktree add --quiet --detach "$WT" "$SNAP"
git diff --name-only --diff-filter=ACMR "$BASE" "$SNAP" > "$OUT/changed-files.txt"

GIT_COMMON=$(cd "$(git rev-parse --git-common-dir)" && pwd -P)
run() {
  local image=$1
  shift
  docker run --rm -i --user "$(id -u):$(id -g)" -e HOME=/tmp \
    -e GIT_CONFIG_COUNT=1 -e GIT_CONFIG_KEY_0=safe.directory -e GIT_CONFIG_VALUE_0='*' \
    -v "$GIT_COMMON:$GIT_COMMON" -v "$TMP:$TMP" -v "$OUT:$OUT" -w "$WT" \
    --entrypoint "" "$image" "$@"
}

echo "security-scan: $(wc -l < "$OUT/changed-files.txt" | tr -d ' ') changed file(s) vs $BASE_REF ($(git rev-parse --short "$BASE"))"
echo

echo "=== Code — Semgrep ($SEMGREP_RULES)"
RULES=()
for r in $SEMGREP_RULES; do RULES+=(--config "$r"); done
status=0
run "$SEMGREP_IMAGE" semgrep scan --metrics=off --quiet "${RULES[@]}" --error \
  --baseline-commit "$BASE" --json -o "$OUT/semgrep.json" . >/dev/null || status=$?
echo "$status" > "$OUT/semgrep.status"

echo "=== Secrets — Gitleaks"
CONFIG=()
if [ -n "$GITLEAKS_IGNORE_PATHS" ]; then
  {
    printf '[extend]\nuseDefault = true\n\n[allowlist]\ndescription = "security-scan GITLEAKS_IGNORE_PATHS"\npaths = [\n'
    set -f
    for p in $GITLEAKS_IGNORE_PATHS; do printf "  '''%s''',\n" "$p"; done
    set +f
    printf ']\n'
  } > "$OUT/gitleaks.toml"
  CONFIG=(--config "$OUT/gitleaks.toml")
fi
status=0
run "$GITLEAKS_IMAGE" gitleaks detect --source . --redact --no-banner --log-level error ${CONFIG[@]+"${CONFIG[@]}"} \
  --log-opts="$BASE..$SNAP" --report-path "$OUT/gitleaks.json" --report-format json --exit-code 1 || status=$?
echo "$status" > "$OUT/gitleaks.status"

echo "=== Configuration — Trivy config"
# Helm templates only render as a whole chart: map files to their chart, vendor local file://
# dependencies and drop the ones that cannot be resolved offline (remote / OCI repositories).
run "$PYTHON_IMAGE" python3 - "$WT" "$TMP/helm" "$OUT" <<'EOF'
import re, shutil, sys
from pathlib import Path

R, H, O = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
cfg = [f for f in (O / "changed-files.txt").read_text().split()
       if re.search(r"\.(tf|ya?ml|tpl)$|(^|/)Dockerfile", f) and (R / f).exists()]

def deps(chart_yaml):
    lines = chart_yaml.read_text().splitlines(keepends=True)
    start = next((i for i, l in enumerate(lines) if re.match(r"dependencies:\s*$", l)), None)
    if start is None:
        return lines, None, None, []
    end = start + 1
    while end < len(lines) and (not lines[end].strip() or lines[end][0] in " -#"):
        end += 1
    items = []
    for l in lines[start + 1:end]:
        if re.match(r"\s*- ", l):
            items.append([l])
        elif items:
            items[-1].append(l)
    return lines, start, end, items

def field(item, key):
    m = re.search(rf"^\s*-?\s*{key}:\s*[\"']?([^\"'\s]+)", "".join(item), re.M)
    return m.group(1) if m else ""

def chart_name(d):
    return field([(d / "Chart.yaml").read_text()], "name") if (d / "Chart.yaml").exists() else ""

def vendor(src, dst, depth=0):
    for item in deps(dst / "Chart.yaml")[3]:
        repo = field(item, "repository")
        sub = (src / repo[len("file://"):]).resolve() if repo.startswith("file://") else None
        if depth > 4 or not sub or not (sub / "Chart.yaml").exists():
            continue
        if chart_name(sub) not in {chart_name(c) for c in (dst / "charts").glob("*") if c.is_dir()}:
            tgt = dst / "charts" / sub.name
            shutil.copytree(sub, tgt, dirs_exist_ok=True)
            vendor(sub, tgt, depth + 1)

def prune(root):
    for cy in root.rglob("Chart.yaml"):
        d = cy.parent
        (d / "Chart.lock").unlink(missing_ok=True)
        lines, start, end, items = deps(cy)
        if start is None:
            continue
        present = {chart_name(c) for c in (d / "charts").glob("*") if c.is_dir()}
        present |= {t.name.rsplit("-", 1)[0] for t in (d / "charts").glob("*.tgz")}
        keep = [i for i in items if field(i, "name") in present]
        head = "dependencies:\n" if keep else "dependencies: []\n"
        cy.write_text("".join(lines[:start]) + head + "".join("".join(i) for i in keep) + "".join(lines[end:]))

def chart_root(f):
    d, found = (R / f).parent, False
    while d != R:
        if (d / "Chart.yaml").exists():
            found = True
            if field([(d / "Chart.yaml").read_text()], "type") != "library":
                return d
        d = d.parent
    return "library" if found else None

targets = {}
for f in cfg:
    root = chart_root(f)
    if isinstance(root, Path):
        targets.setdefault(str(root.relative_to(R)), root)
    elif root is None and not f.endswith(".tpl"):
        targets.setdefault(str(Path(f).parent), R / Path(f).parent)
rows = []
for i, (display, src) in enumerate(sorted(targets.items()), 1):
    if (src / "Chart.yaml").exists():
        dst = H / str(i)
        shutil.copytree(src, dst)
        vendor(src, dst)
        prune(dst)
        src = dst
    rows.append(f"{display}\t{src}")
(O / "trivy-targets.tsv").write_text("".join(r + "\n" for r in rows))
EOF
status=0
n=0
while IFS=$'\t' read -r display path; do
  n=$((n + 1))
  echo "$display" > "$OUT/trivy-config-$n.dir"
  run "$TRIVY_IMAGE" trivy config --quiet --skip-check-update --severity HIGH,CRITICAL \
    --format json --output "$OUT/trivy-config-$n.json" "$path" || status=$?
done < "$OUT/trivy-targets.tsv"
echo "$status" > "$OUT/trivy-config.status"
[ "$n" -eq 0 ] && echo "no Dockerfile / YAML / Helm / Terraform changes"

echo
run "$PYTHON_IMAGE" python3 - "$OUT" "$ENFORCE" <<'EOF'
import json, sys
from pathlib import Path

O, enforce = Path(sys.argv[1]), sys.argv[2] not in ("", "0", "false")

def load(name):
    try:
        return json.loads((O / name).read_text())
    except (OSError, ValueError):
        return None

semgrep = [(r["extra"].get("severity", "?"), f"{r['path']}:{r['start']['line']}", r["check_id"].split(".")[-1],
            r["extra"].get("message", "")) for r in (load("semgrep.json") or {}).get("results") or []]
gitleaks = [("SECRET", f"{f['File']}:{f['StartLine']}", f["RuleID"], f"commit {f['Commit'][:7]}")
            for f in load("gitleaks.json") or []]
trivy = []
for d in sorted(O.glob("trivy-config-*.dir")):
    folder = d.read_text().strip()
    for r in (load(d.with_suffix(".json").name) or {}).get("Results") or []:
        trivy += [(m["Severity"], f"{folder}/{r['Target']}", m["ID"], m["Title"]) for m in r.get("Misconfigurations") or []]

def result(key, rows):
    if rows:
        return "findings"
    return "clean" if (O / f"{key}.status").read_text().strip() == "0" else "error"

scanners = [("Code — Semgrep", "semgrep", semgrep), ("Secrets — Gitleaks", "gitleaks", gitleaks),
            ("Configuration — Trivy config", "trivy-config", trivy)]
print(f"{'Scanner':<32}{'Result':<10}Findings")
for label, key, rows in scanners:
    print(f"{label:<32}{result(key, rows):<10}{len(rows)}")
for label, _, rows in scanners:
    if rows:
        print(f"\n{label}:")
        for sev, loc, rule, msg in rows:
            print(f"  {sev:<9}{loc}  [{rule}]  {msg[:110]}")
if trivy:
    print("\nNote: unlike the PR check, the local Trivy scan also lists misconfigurations already on the base branch.")
print(f"\nJSON reports: {O}")
total = sum(len(rows) for _, _, rows in scanners)
if total and enforce:
    sys.exit(1)
if total:
    print("Report-only: set ENFORCE=1 to fail on findings.")
EOF
