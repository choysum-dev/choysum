#!/usr/bin/env bash
# Typecheck modules unit + e2e sources (excluded from `choysum test typecheck`).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if [[ ! -x ./node_modules/typescript/bin/tsc ]]; then
  echo "error: missing ./node_modules/typescript/bin/tsc (install root node_modules first)" >&2
  exit 1
fi
if [[ ! -f modules/tsconfig.json ]]; then
  echo "error: modules/tsconfig.json missing (run type-fetch / install to generate paths)" >&2
  exit 1
fi
# Refresh @playwright/test path mapping from the live modules tsconfig.
python3 - <<'PY'
import json
from pathlib import Path
base = json.loads(Path("modules/tsconfig.json").read_text())
co = dict(base.get("compilerOptions") or {})
paths = dict(co.get("paths") or {})
paths["@playwright/test"] = ["../node_modules/@playwright/test/index.d.ts"]
cfg = {
    "$schema": "https://json.schemastore.org/tsconfig",
    "display": "Modules unit + e2e tests",
    "compilerOptions": {**co, "noEmit": True, "skipLibCheck": True, "paths": paths},
    "include": [
        "**/*.test.ts",
        "**/*.test.tsx",
        "**/*.spec.ts",
        "**/*.spec.tsx",
        "**/e2e/**/*.ts",
        "**/*.d.ts",
    ],
    "exclude": ["**/node_modules/**"],
}
Path("modules/tsconfig.tests.json").write_text(json.dumps(cfg, indent=2) + "\n")
PY
exec ./node_modules/typescript/bin/tsc -p modules/tsconfig.tests.json --pretty false "$@"
