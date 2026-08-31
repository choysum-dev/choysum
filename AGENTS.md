# AGENTS.md

## Agent skills

On-demand procedures and repository policies live under [`.agents/skills/`](.agents/skills/)
([Agent Skills](https://agentskills.io/specification) open format). Notable skills:

| Skill | Use when |
| --- | --- |
| [`git-commit`](.agents/skills/git-commit/SKILL.md) | Creating commits with conventional message format |
| [`code-comment`](.agents/skills/code-comment/SKILL.md) | Reviewing or cleaning up source comments |
| [`module-initdata`](.agents/skills/module-initdata/SKILL.md) | Adding or moving module bootstrap/demo init data |

## Cloud and local agent setup

Choysum is a single product: a Go binary (`choysum`) that embeds a QuickJS
TypeScript runtime and serves the ERP platform (gRPC + gRPC-Web + Vue web UI)
from one process. TypeScript/Vue modules under `modules/` are compiled and run
inside that Go process; Node.js is only used for the dev/test toolchain, not at
runtime. Tooling versions (Go 1.26.x, Node 22, Python 3) are already installed.

### Build the CLI (required before install/run; artifacts are git-ignored)

The embedded assets (`internal/bootstrap/web/dist`,
`pkg/jsengine/scripts/vuesfc/dist/index.js`) and the `choysum` binary are all
git-ignored, so they must exist before you can install modules or run the app.
Regenerate + build after a fresh checkout or after changing embedded/web code:

```bash
go generate ./pkg/jsengine/scripts/vuesfc/...   # needs network (esm.sh)
go generate ./internal/bootstrap/web/...         # needs network (esm.sh); 404 type-fetch warnings are harmless
go build -o choysum .
```

The `go generate` steps fetch npm packages from `https://esm.sh`; the first run
populates a cache under `.choysum/pkg/esm` and later runs can work offline.

### Install modules, then run the server

Modules live in the local `./modules` dir (auto-detected because cwd contains
`modules/`). Install them before running; install is idempotent:

```bash
./choysum install core base task meta auth document web partner
./choysum run --config config.yaml   # serves http://localhost:9527 ( / redirects to /web/ )
```

Gotchas:
- `server.environment` in config is a **scope factory name**; only `default` is
  registered. Using `development`/`production` fails with
  `scope factory not registered`. The built-in default is already `default`, so
  running with no config works too; `config.yaml` here just enables `hotReload`.
- Bootstrap seeds (roles + `admin` + `base.company_main`) apply without
  `--with-demo`. Avoid `--with-demo` for `base`: `base.company_demo` currently
  omits required `CurrencyId` and aborts the install.
- Default DB is embedded SQLite at `.choysum/choysum.sqlite`; no external DB
  needed. Postgres/MySQL and S3 document storage are optional.

### Auth / first login

The `auth` module seeds a `admin` user and a `base.company_main` company at
install time (see `modules/auth/data/bootstrap.json`). New users can self-register
at `/web/register`, which auto-logs in — the simplest way to exercise the stack.

### Lint / test / build

| Scope | Command |
| --- | --- |
| Go format (lint) | `go fmt ./...` |
| Go build | `go build ./...` |
| Go tests | `go test ./... -count=1` |
| Module typecheck | `./choysum test typecheck <module>` or `--all` |
| Module unit (BE+FE) | `./choysum test unit <module>` (`--be` / `--fe` to scope) |
| Module E2E | `./choysum test e2e <module>` (auth/base/meta/task; needs Playwright browsers) |

Module `test typecheck`/`test unit`/`test e2e` need the root `node_modules` on
PATH. Populate it (matches CI) and prepend its bin dir:

```bash
mkdir -p .choysum/tmp
python3 scripts/ci/compute_root_node_modules_deps.py --modules-path modules \
  --target-modules-json '[]' --output .choysum/tmp/root-node-modules-deps.txt
python3 scripts/ci/install_root_node_modules_deps.py \
  --deps-file .choysum/tmp/root-node-modules-deps.txt \
  --workspace "$PWD"
export PATH="$PWD/node_modules/.bin:$PATH"
```

For E2E also run `npx playwright install --with-deps chromium` first.

### Publishing npm modules (`@choysum-dev/*`)

CI workflow: `.github/workflows/modules-publish.yml` (OIDC + Environment
`npm-publish`). `push` to `main` only validates; real publish is
`workflow_dispatch` (requires environment approval).

New module first-time registry + Trusted Publishing binding (local, idempotent):

```bash
python3 scripts/ci/modules_npm_trust.py --module <name> --apply
# preview: omit --apply
```

`--apply` opens a browser OTP for `npm trust` (classic bypass-2FA tokens cannot
bind Trusted Publishing). Prerelease `0.0.0-*` packages publish with `--tag latest`.

Design notes: `.dev/docs/infra/ci/modules_publish_oidc_plan.md`.
