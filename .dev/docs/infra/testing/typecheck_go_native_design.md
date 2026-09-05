<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# Go-native Typecheck 落地设计

> 状态：**一次性交付规格**（可实施）  
> 日期：2026-09-05（修订：硬切无 flag；§6 按 **5 个 PR** 交付）  
> 目标：用一次合并替换 `npx vue-tsc`；`choysum test typecheck` 与 `--with-typecheck` **只**走 Go-native 路径，**零 Node**。  
> 相关：
>
> - 背景评估：[`typecheck_typescript_go_assessment.md`](./typecheck_typescript_go_assessment.md)
> - 旧契约（将被本文覆盖实现段）：[`ts_test_framework_design.md`](./ts_test_framework_design.md) §6.2.1
> - 现实现（删除目标）：`internal/testing/typecheck` → `exec npx vue-tsc`
> - Program 原型：`internal/parser/backendtsparser/semantic_type_resolver.go`（`compiler.NewProgram` + `wrapvfs` + `bundled.WrapFS`）

**实施以本文为准。** 不做 `CHOYSUM_TYPECHECK_ENGINE`、不做 `--engine`、不做默认 `vue-tsc` + 旁路 Go 的长期并存。开发期可用分支 / PR 内对比脚本，**合并到主线即硬切**。

---

## 0. 冻结决策

| 项 | 决策 |
| --- | --- |
| Checker | 仅 `github.com/buke/typescript-go-internal/v7`（当前 Choysum：`v7.0.2`；升级跟 TS 7.x tag） |
| Vue codegen | `@vue/language-core@3.3.7`（与现 `vue-tsc@3.3.7` 同线）经 QuickJS facade |
| 上游 Vue 入口 | `createVueLanguagePlugin` + `plugin.createVirtualCode` + `forEachEmbeddedCode`；取 `script_(js\|jsx\|ts\|tsx)` |
| Host | `compiler.NewCompilerHost` + `wrapvfs` overlay + `bundled.WrapFS`；`.vue` 读到的是 service script 文本 |
| 诊断 | `compiler.GetDiagnosticsOfAnyProgram`（或等价串联 `GetSyntacticDiagnostics` / `GetSemanticDiagnostics` 等） |
| 兼容策略 | **不保留** Node typecheck 回滚开关；合并前用 fixture / 对比脚本验收，合并后删除 `vue-tsc` 调用与 npm 预检 |
| content-mapper | 不阻塞开工；Host 边界可日后适配，codegen 不依赖它 |

---

## 1. 目标与非目标

### 1.1 目标

1. `choysum test typecheck <app|--all>` 与 `test … --with-typecheck` **不**依赖 `node` / `npx` / `vue-tsc` / 为 typecheck 准备的 root `node_modules`。
2. 检查根文件与今日一致：
   - `modules/<app>/*.ts`
   - `modules/<app>/service/**/*.ts`
   - `modules/<app>/web/**/*.{ts,tsx,vue}`
   - 共享 `.d.ts`、type-fetch 产物、`@/*`、ambient stubs、`vite/client`（改为 embed/vendoring）
3. stderr 诊断；失败非 0；`--fail-fast` 行为不变。
4. `.vue` 含 **模板类型检查**（language-core 生成的 `__VLS_*` service script），不是 `@vue/compiler-sfc` 运行时编译产物。

### 1.2 非目标

1. Vitest / Playwright / nyc 去 Node。  
2. golar / Node-NAPI。  
3. QuickJS 内跑完整 checker（`@volar/typescript` / `vue-tsc`）。  
4. `vuesfc` 与 typecheck codegen 共用产物。  
5. 双引擎 feature flag、应急 `vue-tsc` 环境变量。

### 1.3 合并门禁（一次切齐的验收）

1. 无 Node PATH 可跑 `choysum test typecheck --all`。  
2. `internal/testing/typecheck` 无 `exec.Command…("npx"…)` / `vue-tsc`。  
3. Fixture + 选定 app（至少 `auth`、`web`）诊断集合相对旧 `vue-tsc` 基线：**错误码 + 文件**差异已文档化且无静默漏检。  
4. `.vue` 故意模板错误 remap 回源文件行号（列偏差允许有文档化阈值）。

---

## 2. 架构

```text
internal/testing/typecheck.Run / TypecheckApp
        │  仅编排：app 发现、tmp、stderr、exit
        ▼
internal/typecheck.Check(ctx, Options) → Result
        │
        ├─ collectRootFiles + buildCompilerOptions
        ├─ vue.Coder.CreateServiceScript(.vue)     ← QuickJS + @vue/language-core
        ├─ overlayFS(.vue → service script text)
        ├─ compiler.NewCompilerHost + bundled.WrapFS
        ├─ compiler.NewProgram(...)
        ├─ compiler.GetDiagnosticsOfAnyProgram(...)
        └─ remap diagnostics via CodeMapping → stderr
```

| 层 | 包 | 职责 |
| --- | --- | --- |
| CLI 编排 | `internal/testing/typecheck` | 发现 app、临时根、`--keep`、调用引擎、写 stderr |
| 引擎 | `internal/typecheck` | options、file list、Host、Program、诊断 |
| Vue codegen | `internal/typecheck/vue` + `pkg/jsengine/scripts/vuevirtual` | 只做虚拟码；不 check |
| Checker | `typescript-go-internal/v7/pkg/{compiler,core,tsoptions,vfs,bundled,ast}` | 唯一语义引擎 |

边界：

| 现有 | 关系 |
| --- | --- |
| `pkg/jsengine/scripts/vuesfc` | 运行时 SFC；**禁止**作 typecheck 输入 |
| `internal/parser/vueparser` | 结构解析；不替代 language-core |
| `semantic_type_resolver` | 单文件 `NewProgram` 参考；引擎扩展为多文件 project |
| `choysum type-fetch` | `.d.ts` 仍由它产出；Host 必须可读 |

---

## 3. 仓库布局（合并后形态）

```text
internal/typecheck/
  doc.go
  check.go                 // Check(ctx, Options) (Result, error)
  options.go
  config.go                // CompilerOptions + root file list（迁自 testing/typecheck）
  host.go                  // overlay VFS + NewCompilerHost
  program.go               // NewProgram + GetDiagnosticsOfAnyProgram
  report.go                // format + structured Diagnostic
  ambient_vite.go          // go:embed vite/client 最小 d.ts
  ambient_subpath.go       // 原 writeSubpathStubs
  vue/
    coder.go               // Coder 接口 + CreateServiceScript 结果类型
    quickjs.go             // 调 vuevirtual embed
    mapping.go             // CodeMapping → 源 .vue offset
    helpers_embed.go       // template-helpers.d.ts / props-fallback.d.ts
  testdata/                // .ts / .vue fixtures + golden
internal/testing/typecheck/
  typecheck.go             // 只调 internal/typecheck；删除 Node 预检与 npx
pkg/jsengine/scripts/vuevirtual/
  gen.go / package.json / src/index.ts / dist/index.js
  // go:generate 打包；与 vuesfc 并列，禁止互相 import 产物
```

---

## 4. Go 引擎：真实 typescript-go-internal API

依赖（与仓库一致）：

```go
import (
    "github.com/buke/typescript-go-internal/v7/pkg/ast"
    "github.com/buke/typescript-go-internal/v7/pkg/bundled"
    "github.com/buke/typescript-go-internal/v7/pkg/compiler"
    "github.com/buke/typescript-go-internal/v7/pkg/core"
    "github.com/buke/typescript-go-internal/v7/pkg/tsoptions"
    "github.com/buke/typescript-go-internal/v7/pkg/vfs"
    "github.com/buke/typescript-go-internal/v7/pkg/vfs/osvfs"
    "github.com/buke/typescript-go-internal/v7/pkg/vfs/wrapvfs"
)
```

### 4.1 `Options` / `Check`

```go
package typecheck

type Options struct {
    ModulesPath string // 绝对或相对仓库的 modules 根
    RepoRoot    string
    App         string // 单 app；由 testing 层循环 --all
    KeepDir     string // 非空则落盘虚拟 .vue service script 等
}

type Diagnostic struct {
    File     string // remap 后路径（.vue 或 .ts）
    Start    int    // 源文件 UTF-16/UTF-8 约定在 mapping.go 冻结；先按 TS 常用 UTF-16 码元对齐 volar
    Length   int
    Code     int32  // TS####
    Category string // "error" | "warning" | ...
    Message  string
}

type Result struct {
    Diagnostics []Diagnostic
}

func Check(ctx context.Context, opts Options) (Result, error)
```

`internal/testing/typecheck.TypecheckApp` 变为：解析路径 → `typecheck.Check` → 格式化 `Result` 到 stderr → `len(errors)>0` 则返回 error。

### 4.2 CompilerOptions（对齐现临时 tsconfig）

在 `config.go` 构造（字段名以 `core.CompilerOptions` 为准，与现 JSON 语义对齐）：

| 现 tsconfig | Go |
| --- | --- |
| `target: ES2020` | `Target: core.ScriptTargetES2020` |
| `module: ESNext` | `Module: core.ModuleKindESNext` |
| `moduleResolution: bundler` | `ModuleResolution: core.ModuleResolutionKindBundler` |
| `strict: true` | 打开对应 Strict* 标志（与现 vue-tsc 临时配置一致） |
| `experimentalDecorators` | `ExperimentalDecorators: core.TSTrue` |
| `allowJs` | `AllowJs: core.TSTrue` |
| `allowArbitraryExtensions` | `AllowArbitraryExtensions: core.TSTrue` |
| `skipLibCheck` | `SkipLibCheck: core.TSTrue` |
| `noEmit` | `NoEmit: core.TSTrue` |
| `paths` / `@/*` | 写入 `Paths`（从 `modules/tsconfig.json` 读，同现 `resolveModulePaths`） |
| `typeRoots` / `types` | 同现 `resolveTypeRoots` / `resolveCompilerTypes`；路径变为 Host 可见 |

可选：写临时 `tsconfig.json` 后调用：

```go
parsed, errs := tsoptions.GetParsedCommandLineOfConfigFile(
    configPath, /*options*/ nil, /*optionsRaw*/ nil, host, extendedConfigCache,
)
```

优先保证与现 include/exclude 字节级可对比；稳定后可改为纯 Go 构造 `tsoptions.ParsedCommandLine`，去掉临时文件（`--keep` 仍可 dump）。

### 4.3 Host + VFS（含 `.vue` overlay）

参考 `buildSemanticProgram` / `newSemanticOverlayFS`，扩展为 **多文件 map**：

```go
// host.go
func newTypecheckFS(overlays map[string]string /* slash path → content */) vfs.FS {
    base := osvfs.FS()
    caseSensitive := base.UseCaseSensitiveFileNames()
    overlay := wrapvfs.Wrap(base, wrapvfs.Replacements{
        FileExists: func(p string) bool {
            if _, ok := lookupOverlay(overlays, p, caseSensitive); ok {
                return true
            }
            return base.FileExists(p)
        },
        ReadFile: func(p string) (string, bool) {
            if content, ok := lookupOverlay(overlays, p, caseSensitive); ok {
                return content, true
            }
            return base.ReadFile(p)
        },
        // 若 .vue 需作为“存在的源文件”参与枚举，按 spike 补 DirectoryExists / GetAccessibleEntries
    })
    return bundled.WrapFS(overlay)
}

host := compiler.NewCompilerHost(
    currentDir, // modules 或 repo 根，与 path 解析一致
    fs,
    bundled.LibPath(),
    nil, // extendedConfigCache：可按需缓存
    nil, // trace
)
```

**`.vue` 约定（冻结）：**

- Program root / import 路径保持 **`Foo.vue`**（与今日 `vue-tsc` / `allowArbitraryExtensions` 一致）。  
- `ReadFile("…/Foo.vue")` 返回 **service script 文本**（`script_ts` / `script_tsx` 内容），不是 SFC 原文。  
- **ScriptKind 策略（PR-3 冻结为 B）：** typescript-go-internal 当前对 `.vue` 报 TS6054，因此 Program root 使用虚拟旁路 **`Foo.vue.ts`**（内容同 service script）；同时 overlay `Foo.vue` 以便 `import … from './Foo.vue'` 解析。诊断 `FileName` remap 回 `.vue`，offset 经 `SpanMapping`。策略 A（根路径字面量 `.vue`）待上游扩展名支持后再评估。  

额外 overlay：

- embed 的 `vite/client` ambient  
- subpath stubs  
- `@vue/language-core/types/template-helpers.d.ts`、`props-fallback.d.ts`（改写 service script 内 `/// <reference types="…" />` 为稳定虚拟路径）

### 4.4 Program + 诊断

```go
program := compiler.NewProgram(compiler.ProgramOptions{
    Host: host,
    Config: &tsoptions.ParsedCommandLine{
        ParsedConfig: &core.ParsedOptions{
            FileNames:       rootFiles, // 含 .ts/.tsx/.vue（及必要 .d.ts）
            CompilerOptions: opts,
        },
    },
    SingleThreaded: core.TSTrue, // 先单线程；稳定后可放开
})

diags := compiler.GetDiagnosticsOfAnyProgram(
    ctx,
    program,
    nil,  // 全 program；单文件调试时传 *ast.SourceFile
    true, // skipNoEmitCheckForDtsDiagnostics：与 noEmit 门禁对齐
    program.GetBindDiagnostics,
    program.GetSemanticDiagnostics,
)
```

将 `*ast.Diagnostic` 转为 `typecheck.Diagnostic`：文件名、start/length、code、message。若 `FileName` 落在 overlay 的 `.vue` service script 上，用该文件的 `CodeMapping` 映回 SFC 原文 offset，再格式化为行列。

### 4.5 报告

stderr 一行一例：

```text
modules/web/web/pages/HomeView.vue:42:5 - error TS2322: Type 'string' is not assignable to type 'number'.
```

测试用稳定键：`sort(File, Code, Message)`，不绑死列号。

---

## 5. Vue codegen：真实 `@vue/language-core` API

钉死版本：**`@vue/language-core@3.3.7`** + 配套 `@volar/language-core@2.4.28`（随 language-core 依赖）。  
**必须注入完整 JS `typescript` 模块**（language-core 未将其列为 runtime dependency；`createVueLanguagePlugin` 第一参）。

### 5.1 调用链（facade 内必须照此实现）

```js
import * as ts from "typescript";
import {
  createVueLanguagePlugin,
  createParsedCommandLineByJson,
  forEachEmbeddedCode,
} from "@vue/language-core";

const SERVICE_SCRIPT_RE = /^script_(js|jsx|ts|tsx)$/;

/**
 * Choysum facade（自有导出名，不是上游 API）。
 * 等价于 vue-tsc 经 @volar/typescript 挂上的 language plugin 的 codegen 段。
 */
export function createServiceScript(fileName, source, options = {}) {
  const snapshot = {
    getText: (s, e) => source.slice(s, e),
    getLength: () => source.length,
    getChangeRange: () => undefined,
  };

  const vueOptions =
    options.vueCompilerOptions ??
    createParsedCommandLineByJson(ts, /*sys*/ minimalSys, options.currentDirectory ?? "/", {}).vueOptions;

  const compilerOptions = options.compilerOptions ?? {
    strict: true,
    target: ts.ScriptTarget.ES2020,
    module: ts.ModuleKind.ESNext,
    moduleResolution: ts.ModuleResolutionKind.Bundler,
  };

  const plugin = createVueLanguagePlugin(
    ts,
    compilerOptions,
    vueOptions,
    (scriptId) => scriptId,
  );

  const root = plugin.createVirtualCode(fileName, "vue", snapshot, {
    getAssociatedScript: () => undefined,
  });
  if (!root) {
    throw new Error(`createVirtualCode returned undefined for ${fileName}`);
  }

  let service = null;
  for (const code of forEachEmbeddedCode(root)) {
    if (SERVICE_SCRIPT_RE.test(code.id)) {
      service = code;
      break;
    }
  }
  if (!service) {
    throw new Error(`no service script embedded for ${fileName}`);
  }

  const content = service.snapshot.getText(0, service.snapshot.getLength());
  const lang = service.id.slice("script_".length); // js|jsx|ts|tsx

  return {
    embeddedId: service.id,
    scriptKind: lang,
    content: rewriteVueHelperReferences(content), // 见 §5.3
    mappings: flattenCodeMappings(fileName, service.mappings ?? []),
  };
}

function flattenCodeMappings(sourceFile, mappings) {
  // CodeMapping: sourceOffsets[], generatedOffsets[], lengths[], data (CodeInformation)
  const out = [];
  for (const m of mappings) {
    const n = Math.min(
      m.sourceOffsets?.length ?? 0,
      m.generatedOffsets?.length ?? 0,
      m.lengths?.length ?? 0,
    );
    for (let i = 0; i < n; i++) {
      const data = m.data ?? {};
      const verification = data.verification;
      if (verification === false) continue;
      out.push({
        sourceFile,
        sourceStart: m.sourceOffsets[i],
        sourceEnd: m.sourceOffsets[i] + m.lengths[i],
        generatedStart: m.generatedOffsets[i],
        generatedEnd: m.generatedOffsets[i] + m.lengths[i],
        verification,
      });
    }
  }
  return out;
}
```

`createVueLanguagePlugin` 内 `typescript.getServiceScript` 使用同一 `script_(js|jsx|ts|tsx)` 规则；facade **必须**与之保持一致，不要自创 id。

实测 embedded 树（勿全部喂给 Program）：

```text
main
└─ root_tags
   ├─ script_ts          ← 主检查（含模板表达式 / __VLS_*）
   ├─ scriptsetup_raw    ← format；默认不进 Program
   └─ template
      └─ template_inline_ts_*  ← 已并入 script_ts 语义；默认不单独进 Program
```

### 5.2 QuickJS 打包（`pkg/jsengine/scripts/vuevirtual`）

对标 `vuesfc`：

1. `package.json` 钉 `@vue/language-core@3.3.7`、`typescript`（与 language-core 兼容的 5.x/6.x；**bundle 进 IIFE**，运行时不读 `node_modules`）。  
2. `go:generate` → `dist/index.js` → `//go:embed`。  
3. 导出 `createServiceScript`（或挂到既有 jsengine 全局约定）。  
4. `minimalSys`：`getCurrentDirectory` / 空 `readDirectory` 等；codegen 路径避免依赖真实 Node `fs`（language-core 已偏 browser-friendly：`path-browserify` 等）。

Go：

```go
// internal/typecheck/vue/coder.go
type ServiceScript struct {
    EmbeddedID string
    ScriptKind string // ts|tsx|js|jsx
    Content    string
    Mappings   []SpanMapping
}

type Coder interface {
    CreateServiceScript(path, source string, opts CodegenOptions) (ServiceScript, error)
}
```

`quickjs.go` 加载 embed 脚本，调用 `createServiceScript`，反序列化结果。每个 `.vue` 可缓存 `(path, contentHash) → ServiceScript`。

### 5.3 helper `.d.ts` 与 reference 改写

生成码常见：

```ts
/// <reference types="./node_modules/@vue/language-core/types/template-helpers.d.ts" />
/// <reference types="./node_modules/@vue/language-core/types/props-fallback.d.ts" />
```

合并要求：

1. 从 npm 包 `types/` **vendoring / go:embed** 进 `internal/typecheck/vue`。  
2. facade 把 reference 改成稳定虚拟路径（例如 `/choysum-vue-virtual/types/template-helpers.d.ts`）。  
3. overlay 提供这些路径。

### 5.4 明确不做

- 上游不存在的 `generateVueVirtual`。  
- `@vue/compiler-sfc` `compileSFC` / render 输出当检查输入。  
- QuickJS 内 `vue-tsc` / `@volar/typescript` `runTsc`。  
- 把 `scriptsetup_raw` / 仅 format 用 embedded 当 Program root（除非 spike 证明必须）。

---

## 6. 交付 PR（5 个；文件 / 函数 / 测试粒度）

**冻结拆分：** 共 **5 个 PR**，顺序合入；**PR-5（原 T4）合并主线即硬切**。内部仍可用 T0/T1/… 作清单小节标签，**不要**写进生产源码注释。

| PR | 合并内容 | 合入后 CLI |
| --- | --- | --- |
| **PR-1** | T0 + T1：骨架 + service 纯 TS Program | 仍 `vue-tsc` |
| **PR-2** | T2：web TS/TSX + vite/subpath ambient | 仍 `vue-tsc` |
| **PR-3** | T3a + T3b：golden codegen + `.vue` overlay/remap（GoldenCoder） | 仍 `vue-tsc` |
| **PR-4** | T3c：QuickJS `vuevirtual` embed | 仍 `vue-tsc` |
| **PR-5** | T4：接线 + 拆除 Node typecheck | **硬切 Go-native** |

依赖顺序：

```text
PR-1 → PR-2 → PR-3 → PR-4 → PR-5
         ╲
          可与 PR-3 内「Node golden 脚本」并行起草，但 PR-3 合入须在 PR-2 之后
```

对比旧 `vue-tsc`：仅允许出现在 **PR-1～PR-4** 的 `scripts/` / 一次性工具；**PR-5 后**产品路径为零。

粗估合计约 **8–12 人周**（PR-3 / PR-4 最大）。

---

### 6.1 PR-1（T0 + T1）— 骨架 + service 纯 TS

**建议 PR 标题：** `feat(typecheck): internal/typecheck service Program diagnostics`

**目标：** 一次交付包骨架与可跑的 service/app 根 `.ts` 检查；`testing/typecheck` **仍走 vue-tsc**。

#### 改造清单

| # | 文件 | 动作 |
| --- | --- | --- |
| 1 | `internal/typecheck/doc.go` | 新建：包意图 |
| 2 | `internal/typecheck/options.go` | 新建：`Options`、`CodegenOptions` |
| 3 | `internal/typecheck/result.go` | 新建：`Diagnostic`、`Result`；`Err()` |
| 4 | `internal/typecheck/errors.go` | 新建：Options 校验哨兵错误 |
| 5 | `internal/typecheck/check.go` | `Check`：完整 service 路径（勿留永久 `notImplemented` stub） |
| 6 | `internal/typecheck/config.go` | 从现 `TypecheckApp` 迁出 CompilerOptions 语义 |
| 7 | `internal/typecheck/collect.go` | `CollectRootFiles`：仅 service + app 根 `.ts` + 共享 `.d.ts`（不含 `web/`、`.vue`） |
| 8 | `internal/typecheck/host.go` | `newTypecheckFS`、`lookupOverlay`、`NewCompilerHost` |
| 9 | `internal/typecheck/program.go` | `buildProgram`、`collectDiagnostics` |
| 10 | `internal/typecheck/report.go` | `mapASTDiagnostic`、`FormatStderr` |
| 11 | `internal/typecheck/paths.go` | 迁 paths helpers（**复制**；PR-5 再删 testing 旧份） |
| 12 | `internal/typecheck/typeroots.go` | 迁 typeRoots/types（可仍读磁盘 `@types`） |
| 13 | `scripts/…` 或 `internal/typecheck/cmd/compare/` | **可选**：vue-tsc vs `Check` diff 工具 |

从 `internal/testing/typecheck/typecheck.go` **迁入语义、暂不删除原函数**：

- `resolveModulePaths`、`readModuleTSConfigPaths`、`matchTSConfigPathPattern`、`applyPathPatternReplacements`
- `resolveTypeRoots`、`resolveCompilerTypes`、`globalNpmRoot`
- include/exclude 对齐现临时 tsconfig 的 **service 子集**

#### 函数级

```text
check.go
  Check → validate Options → CollectRootFiles(ScopeService)
       → BuildCompilerOptions → newHost → buildProgram
       → collectDiagnostics → toResult

collect.go
  CollectRootFiles(modulesPath, app, ScopeService) ([]string, error)

config.go
  BuildCompilerOptions(modulesPath, repoRoot) (*core.CompilerOptions, error)
  // 或 BuildParsedCommandLine → *tsoptions.ParsedCommandLine

host.go
  newTypecheckFS(overlays map[string]string) vfs.FS
  newHost(currentDir string, fs vfs.FS) compiler.CompilerHost
  // osvfs → wrapvfs.Wrap → bundled.WrapFS
  // compiler.NewCompilerHost(dir, fs, bundled.LibPath(), nil, nil)

program.go
  buildProgram(...) *compiler.Program
  collectDiagnostics(ctx, program) []*ast.Diagnostic
  // compiler.GetDiagnosticsOfAnyProgram(ctx, program, nil, true,
  //   program.GetBindDiagnostics, program.GetSemanticDiagnostics)

report.go
  mapASTDiagnostic(*ast.Diagnostic) Diagnostic
  FormatStderr(io.Writer, []Diagnostic)
```

参考：`semantic_type_resolver.go` 的 `buildSemanticProgram` / `newSemanticOverlayFS`（多 root）。

**本 PR 不改** `TypecheckApp`。

#### 测试清单

| 文件 | 用例 |
| --- | --- |
| `check_test.go` | `TestCheck_RequiresOptions` |
| 同上 | `TestCheck_ServiceFixture` 端到端 |
| `collect_test.go` | `TestCollectRootFiles_Service`：排除 `*.test.ts`、`web/` |
| `config_test.go` | `TestBuildCompilerOptions_PathsAlias` |
| `host_test.go` | `TestOverlayReadFile` |
| `program_test.go` | `TestProgram_ServiceOK` / `TestProgram_ServiceTypeError` |
| `report_test.go` | `TestFormatStderr_StableShape` |
| `testdata/service_ok/`、`service_err/` | fixtures |

#### DoD

1. `go test ./internal/typecheck/...` 绿；**无 node/npx**。  
2. `service_err` 稳定报类型错误。  
3. 默认 `choysum test typecheck` 行为不变。  
4. （建议）`auth`/`core` service 子集 vs vue-tsc diff 附 PR 描述。

---

### 6.2 PR-2（T2）— Web 纯 TS + ambient

**建议 PR 标题：** `feat(typecheck): include web TS/TSX and embed vite/client ambient`

**目标：** 纳入 `web/**/*.{ts,tsx}`；embed vite/client 与 subpath stubs；仍 **跳过 `.vue`**；CLI 仍 vue-tsc。

#### 改造清单

| # | 文件 | 动作 |
| --- | --- | --- |
| 1 | `internal/typecheck/collect.go` | `ScopeNoVue`：service + web ts/tsx + d.ts |
| 2 | `internal/typecheck/ambient_vite.go` | `//go:embed` + `ViteClientOverlay()` |
| 3 | `internal/typecheck/ambient/vite_client.d.ts` | 最小 ambient |
| 4 | `internal/typecheck/ambient_subpath.go` | 迁 `writeSubpathStubs` → overlay |
| 5 | `internal/typecheck/host.go` / `check.go` | merge vite + subpath overlays |
| 6 | `internal/typecheck/config.go` | include 对齐现 TypecheckApp（除 `.vue`） |

#### 函数级

```text
CollectRootFiles(..., ScopeNoVue)
ViteClientOverlay() (path, content string)
SubpathStubOverlay() (path, content string)
mergeOverlays(...)
```

#### 测试清单

| 文件 | 用例 |
| --- | --- |
| `collect_test.go` | `TestCollectRootFiles_NoVue` |
| `ambient_vite_test.go` | 无磁盘 `node_modules/vite` 时 fixture 不因缺 client 挂 |
| `testdata/web_ts_ok/`、`web_ts_err/` | |
| `check_test.go` | `TestCheck_WebTSFixture` |

#### DoD

1. 无 `.vue`（或过滤后）可 `Check`。  
2. 不依赖 `node_modules/vite/client.d.ts`。  
3. CLI 仍 vue-tsc。

---

### 6.3 PR-3（T3a + T3b）— Vue golden + overlay / remap

**建议 PR 标题：** `feat(typecheck): Vue SFC typecheck via language-core service-script overlay`

**目标：** 同一 PR 内完成 (a) Node 开发脚本 + golden，(b) Host 挂 `.vue` service script + remap（默认 **GoldenCoder**）。产品二进制仍 **不含** QuickJS language-core bundle；CLI 仍 vue-tsc。

#### 改造清单 — 部分 A（codegen golden）

| # | 文件 | 动作 |
| --- | --- | --- |
| 1 | `scripts/typecheck_vue_codegen/`（或 `vue/devtools/`） | Node 脚本；钉/复用 `@vue/language-core@3.3.7` |
| 2 | 同上 `create_service_script.mjs` | 实现 §5.1 `createServiceScript` |
| 3 | `internal/typecheck/testdata/vue/` | 5–10 个代表性 `.vue` |
| 4 | `…/vue/golden/*.service.ts`、`*.mappings.json` | 提交 golden |
| 5 | 脚本旁 README | 刷新 golden 步骤 |

**样例 `.vue` 必须覆盖：** script setup + template；至少一类宏；template `unknownVar`；`Parent`/`Child.vue` import；非 setup `<script lang="ts">`。

#### 改造清单 — 部分 B（Host / remap）

| # | 文件 | 动作 |
| --- | --- | --- |
| 6 | `internal/typecheck/vue/coder.go` | `Coder`、`ServiceScript`、`SpanMapping` |
| 7 | `internal/typecheck/vue/mapping.go` | `RemapOffset`；过滤 `verification === false` |
| 8 | `internal/typecheck/vue/golden_coder.go` | 读 golden；本 PR 默认 Coder |
| 9 | `internal/typecheck/vue/helpers_embed.go` + `helpers/*.d.ts` | vendoring language-core types |
| 10 | `internal/typecheck/collect.go` | `ScopeAll`：**含 `*.vue`** |
| 11 | `internal/typecheck/host.go` | `ReadFile(.vue)` → service script；helper 虚拟路径 |
| 12 | `internal/typecheck/check.go` | `prepareVueOverlays` + 诊断 remap |
| 13 | `internal/typecheck/vue_path.go` | **冻结** ScriptKind 策略 A（`.vue` 即 TS 文本）或 B（`.vue.ts` 旁路） |

#### 函数级

```text
// 脚本
createServiceScript / flattenCodeMappings / rewriteVueHelperReferences

// Go
Coder.CreateServiceScript(...)
prepareVueOverlays(coder, vuePaths)
remapDiagnostic(d, script) Diagnostic
GoldenCoder 读 testdata/vue/golden
```

#### 测试清单

| 文件 | 用例 |
| --- | --- |
| `vue/mapping_test.go` | `TestRemapOffset_SimpleSegment` |
| `vue/path_test.go` | 策略 A/B 约定 |
| `check_vue_test.go` | `TestCheck_VueTemplateErrorRemaps` |
| 同上 | `TestCheck_VueImportChild` / `TestCheck_VueScriptSetupOk` |
| `vue/helpers_test.go` | helper reference 可读 |
| golden 文本 assert | CI 用已提交 golden（刷新脚本无 Node 时可 skip） |

#### DoD

1. golden 已提交；刷新步骤文档化。  
2. `GoldenCoder` 下 `.vue` fixture 无 Node、无 QuickJS bundle 可跑通。  
3. ScriptKind / 模块身份策略写进 PR 并回写 §4.3。  
4. 跨 `.vue` import：至少 1 绿 + 1 故意错误。  
5. CLI 仍 vue-tsc；产品不 embed language-core。

---

### 6.4 PR-4（T3c）— QuickJS `vuevirtual` embed

**建议 PR 标题：** `feat(typecheck): embed language-core createServiceScript in QuickJS`

**目标：** `QuickJSCoder` 替换默认 `GoldenCoder`；运行时无 Node；CLI 仍 vue-tsc。

#### 改造清单

| # | 文件 | 动作 |
| --- | --- | --- |
| 1 | `pkg/jsengine/scripts/vuevirtual/package.json` | `@vue/language-core@3.3.7` + `typescript`（bundle） |
| 2 | `vuevirtual/src/index.ts` | `createServiceScript`（§5.1） |
| 3 | `vuevirtual/src/minimal_sys.ts`、`rewrite_refs.ts` | |
| 4 | `vuevirtual/gen.go`、`script.go` | 对标 `vuesfc`：`go:generate` + `//go:embed` |
| 5 | `internal/typecheck/vue/quickjs.go` | `NewQuickJSCoder` / `CreateServiceScript` |
| 6 | `internal/typecheck/vue/cache.go` | contentHash 缓存 |
| 7 | `internal/typecheck/check.go` | 默认 `Coder = NewQuickJSCoder()` |
| 8 | `vue/golden_coder.go` | 仅测试 / build tag |

**禁止**复用 `vuesfc` 的 `compileSFC` 产物。

#### 函数级

```text
vuevirtual: export createServiceScript
QuickJSCoder.CreateServiceScript → eval embed → 解码 ServiceScript
cachedCoder 包装
```

#### 测试清单

| 文件 | 用例 |
| --- | --- |
| `vue/quickjs_test.go` | `TestQuickJSCoder_MatchesGolden` |
| 同上 | `TestQuickJSCoder_NoNode`（PATH 无 node） |
| `vue/cache_test.go` | 同内容二次不重复 codegen |
| PR 描述 | 记录 `dist/index.js` 大小；超阈值开 Go codegen issue（**不**恢复 vue-tsc） |

#### DoD

1. `go test ./internal/typecheck/...` 默认 QuickJS、无 Node。  
2. 与 golden 对齐。  
3. `go generate ./pkg/jsengine/scripts/vuevirtual/...` 说明写入包 README / AGENTS。  
4. CLI 仍 vue-tsc。

---

### 6.5 PR-5（T4）— 硬切 CLI

**建议 PR 标题：** `feat(typecheck): replace vue-tsc with internal/typecheck (hard cut)`

**目标：** `choysum test typecheck` / `--with-typecheck` 只调 Go 引擎；删除 npm/vue-tsc 预检与相关测试。

#### 改造清单

| # | 文件 | 动作 |
| --- | --- | --- |
| 1 | `internal/testing/typecheck/typecheck.go` | `TypecheckApp` → `typecheck.Check` + `FormatStderr`；删 npx/vue-tsc/临时 tsconfig（若引擎自管） |
| 2 | 同上 | **删除：** `typecheckRequiredModules`、`typecheckInstallCommand`、`validateTypecheckToolchainVersions`、`nodePackageVersion`、`resolveNpxPath`、`resolveViteClientDTS`、npm 安装指引、`typecheckVueTSCVersion` / `typecheckTypeScriptVersion` |
| 3 | 同上 | **保留：** `Run`、`ResolveApps`、`HasTargets`、`hasTypecheckInputs`、`sanitizeAppToken`、import boundary、`warnMissingTypeAssetsPrecheck`（文案去 vue-tsc） |
| 4 | `typecheck_test.go` | 删/改 npx、版本钉死、vite client 相关测 |
| 5 | `resolve_test.go` | 删 `TestResolveNpxPath`；清理已迁 helpers |
| 6 | `run_test.go` | 去 fake vue-tsc；改 fixture / 真实引擎 |
| 7 | `typecheck_app_test.go` | 断言诊断；保留 tmp/paths/boundary |
| 8 | `noderuntime.go` / `runner/defaults.go` | 收窄 typecheck 预检；确认 `--with-typecheck` |
| 9 | `cmd/cmd_test_cmd_unit.go` | 去掉 `requires npm install` 文案 |
| 10 | `ts_test_framework_design.md` §6.2.1、`AGENTS.md`、assessment、CI deps / workflows | 见 §10 |

#### 函数级（切换后）

```text
TypecheckApp:
  hasTypecheckInputs → (optional prechecks)
  → typecheck.Check → FormatStderr
  → errors ⇒ non-zero
```

#### 测试清单

| 文件 | 用例 |
| --- | --- |
| `run_test.go` | 无 node PATH 下 Run |
| `typecheck_app_test.go` | 故意错误 → stderr `TS####`；`--keep` dump（若有） |
| `internal/typecheck` | 全绿 |
| 手工/CI | `PATH=/usr/bin:/bin go run . test typecheck <app>` |

#### DoD

1. typecheck **产品路径**无 `vue-tsc` / npx。  
2. 无 Node 可跑 typecheck。  
3. §10 勾完；**无** engine flag。

---

### 6.6 PR ↔ 包结构对照（全部合入后）

```text
internal/typecheck/
  doc.go options.go result.go errors.go check.go     ← PR-1
  collect.go config.go paths.go typeroots.go         ← PR-1，PR-2 扩 collect
  host.go program.go report.go                       ← PR-1
  ambient_vite.go ambient_subpath.go ambient/*.d.ts  ← PR-2
  vue/coder.go vue/mapping.go vue/helpers*           ← PR-3
  vue/golden_coder.go                                ← PR-3（测）
  vue/quickjs.go vue/cache.go                        ← PR-4
  vue_path.go                                        ← PR-3
  testdata/service_*                                 ← PR-1
  testdata/web_ts_*                                  ← PR-2
  testdata/vue/                                      ← PR-3
pkg/jsengine/scripts/vuevirtual/                     ← PR-4
internal/testing/typecheck/typecheck.go              ← PR-5 瘦身接线
```

---

## 7. content-mapper（非阻塞）

| | |
| --- | --- |
| 现在 | overlay + `ReadFile(.vue)=service script` 即可 |
| 以后 | typescript-go / TS 7.1 content-mapper 成熟且 `typescript-go-internal` 发对应 **tag** 后，可将 codegen 挂到 mapper 协议；Choysum 仍自管 codegen（去 Node） |
| Choysum `go.mod` | 钉 **semver tag**，不钉 git `main` |

---

## 8. 测试与 CI（总则）

细节按 PR 见 **§6.1–§6.5 测试清单**。总原则：

| 层 | 要求 |
| --- | --- |
| `internal/typecheck` | 默认 `go test` **无 Node**；fixture + golden |
| PR-3 刷新脚本 | 仅开发者本机 / 可选 CI；不进产品 |
| PR-5 后 CI | typecheck job 去掉仅为 `vue-tsc` 的 npm install；Vitest/E2E Node 保留 |
| 合并前对比 | PR-1～PR-4 可附 go vs 旧 vue-tsc 的 `(File,Code,Message)` diff |

---

## 9. 风险

| 风险 | 缓解 |
| --- | --- |
| embed `typescript`+language-core 体积/内存 | PR-4 前测包体；超阈值开 Go codegen 专项，不回退 Node typecheck |
| `.vue` ScriptKind / 模块解析与 vue-tsc 不一致 | PR-3 强制覆盖 `import X from './Y.vue'`；必要时改 `.vue.ts` 旁路并统一 remap |
| helper reference 路径 | §5.3 强制 embed + 改写 |
| `GetDiagnosticsOfAnyProgram` 行为与 tsc CLI 细微差别 | fixture 对齐错误码；文档化已知差异 |
| 硬切后无法一键回 vue-tsc | 用 git revert 整 PR；**产品内不留开关** |

---

## 10. 文档与代码同步清单（归入 PR-5）

见 **§6.5 改造清单 #10**。合并前核对：

1. `ts_test_framework_design.md` §6.2.1 已改写。  
2. `AGENTS.md` typecheck 不再要求 root `node_modules`。  
3. assessment §10 与硬切一致。  
4. `typecheckVueTSCVersion` / `typecheckRequiredModules` / `resolveNpxPath` 等已删。

---

## 11. 决策摘要

1. **一次交付硬切**：PR-5 合并后只有 Go-native typecheck；无 `CHOYSUM_TYPECHECK_ENGINE`。  
2. **Checker API**：`compiler.NewCompilerHost`、`compiler.NewProgram`、`compiler.GetDiagnosticsOfAnyProgram`、`bundled.WrapFS`、`wrapvfs`。  
3. **Vue API**：`createVueLanguagePlugin` → `createVirtualCode` → `forEachEmbeddedCode` → `script_*`；Choysum facade 名 `createServiceScript`。  
4. **不等** content-mapper；**不**用 `compiler-sfc` 冒充类型检查。  
5. **交付形态：5 个 PR**（PR-1=T0+T1 … PR-5=T4）；评估文档留作背景，**施工与验收以本文为准**。
