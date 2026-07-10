<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# Document 模型去重与重构方案（2026-07-10）

## 1. 目标与结论

本方案参考 [base 模型重构方案](../base/base_model_refactor_plan20260708.md) 和 [meta -> core 抽取方案](../meta/meta_core_extraction_plan20260707.md)，聚焦 `modules/document/service/models` 的可执行整理路径，目标是：

1. 优先消除 document 模型间的横向重复（`backendEnv`、`normalizeOptionalText`、`requireUserId` 等）。
2. 充分复用 `modules/core/service/utils` 和 `modules/core/service/runtime` 已有工具。
3. 将上传流水线的纯函数集中，为后续超大文件拆分做准备。
4. 拆分 `attachment_binding.ts` 和 `attachment_object.ts` 两个超大文件，分离业务逻辑与模型壳。
5. 保持领域边界清晰，避免把 domain-specific 语义错误上提到 core。

结论：

1. 应先执行域内去重 + Core 复用（收益最高、风险最低），充分利用已有 core 工具并补齐缺口。
2. env 读取统一策略为 `__choysumBackendEnv` 优先，再回退到 `import.meta.env`。
3. 本轮在 core 侧补齐多 key 正整数 env 读取能力，替代 document 侧重复的 `resolvePositiveIntEnv`。
4. `attachment_object.ts`（1,269 行）和 `attachment_binding.ts`（1,131 行）需要按职责拆分。
5. `contracts.ts` 与 `_owner_authorization.ts` 本轮不纳入重构范围；`error.ts` 纳入改造，新增 `throwDocumentError`（或 `createDocumentError + throwDocumentError`）统一模型层错误构造。
6. 测试文件已迁移到 `service/tests/`，本轮不再调整。

## 2. 纠偏与边界

### 2.1 保持不变原则

1. 对外行为不变：错误文本、错误码、返回结构不变。
2. 模型间 `import` 引用关系不变（避免循环依赖）。
3. 先做"无语义变化重构"，不引入新抽象层。

### 2.2 不建议做的事情

1. 不建议把 `requireUserId` / `requireCompanyId` 上提到 `BaseModel`——它们抛出 document 域错误码，属于域约定。
2. 不建议本轮动 `_owner_authorization.ts`（401 行）——它自身有内部重复，但单独处理更安全。
3. 不建议本轮拆分 `contracts.ts`（399 行）——纯类型定义与常量，拆分收益不大且容易引入循环依赖。
4. 不建议改动 `hook/post_init.ts`——它属于服务生命周期层，不是模型层。

### 2.3 本轮接口与提交约束（新增）

1. `getBackendEnvPositiveInt` 在 A1 后采用兼容签名：
   - `getBackendEnvPositiveInt(key: string, defaultValue: number): number`
   - `getBackendEnvPositiveInt(keys: readonly string[], defaultValue: number): number`
2. `throwDocumentError` 语义固定为 `never`（函数内部直接抛出），模型层不再直接链式调用 `newDocumentError(...).withGrpcCode(...).withMetadata(...)`。
3. A2 强制拆分为 A2a/A2b 两个 PR：
   - A2a：仅做 env/date/normalization/共享 helper 替换。
   - A2b：仅做 `error.ts` 落地与模型层错误链收敛。

## 3. 现状证据与重复热点

### 3.1 文件规模分布

| 文件 | 行数 | 分类 |
|------|------|------|
| `stored_content.ts` | 78 | 轻量 |
| `error.ts` | 80 | 轻量 |
| `hook/post_init.ts` | 81 | 轻量 |
| `attachment_mutation_ledger.ts` | 130 | 中等 |
| `upload_session.ts` | 249 | 中等 |
| `contracts.ts` | 399 | 偏大（本轮不动） |
| `_owner_authorization.ts` | 401 | 偏大（本轮不动） |
| `attachment_binding.ts` | **1,131** | 🔴 超大 |
| `attachment_object.ts` | **1,269** | 🔴 超大 |
| **合计** | **~3,818** | |

> 对比：base 模型总 1,950 行、最大文件 392 行。document 两个超大文件分别为 base 最大文件的 **2.9x** 和 **3.2x**。

### 3.2 P0（高优先，逐字重复）

#### 3.2.1 `backendEnv` / `resolvePositiveIntEnv` / `parseNowInput` —— 3 文件逐字重复

以下三个函数在 `attachment_object.ts`、`attachment_mutation_ledger.ts`、`upload_session.ts` 中各拷一份，完全一致：

| 函数 | 行数 | Core 等价物 | 可直接替换？ |
|------|------|------------|------------|
| `backendEnv()` | ~6 | `getBackendEnv()` from `@/core/service/runtime/env/backend_env` | ✅ A1 统一优先级后可直接替换 |
| `resolvePositiveIntEnv(keys, fallback)` | ~8 | `getBackendEnvPositiveInt(keys, fallback)` from 同上 | ✅ A1 支持多 key 后可直接替换 |
| `parseNowInput(nowISO?)` | ~5 | `parseISODate()` | ✅ A1 补齐后可直接替换 |

#### 3.2.2 `backendEnv` fallback 顺序统一决议（A1 落地）

| 实现 | 优先顺序 |
|------|---------|
| **Core** `getBackendEnv()` | `__choysumBackendEnv` → `import.meta.env` → `{}` |
| **Document** `backendEnv()` | `__choysumBackendEnv` → `import.meta.env` → `{}` |

统一策略：`__choysumBackendEnv` 优先，`import.meta.env` 次之。该策略会在 A1 阶段落地到 core env 工具，并通过冲突场景测试（两个来源同时存在且值不同）锁定行为。

#### 3.2.3 `normalizeOptionalText` —— 3 处重复

| 文件 | 作用域 |
|------|--------|
| `_owner_authorization.ts` | standalone `function` |
| `attachment_binding.ts` | `private static` |
| `attachment_object.ts` | `private static` |

Core 已有 `normalizeOptionalString`，语义完全一致：`String(value \|\| '').trim()` → empty→undefined。**可直接替换。**

#### 3.2.4 `requireText` —— 2 处完全相同

| 文件 | 作用域 |
|------|--------|
| `attachment_binding.ts` | `private static` |
| `attachment_object.ts` | `private static` |

`_owner_authorization.ts` 另有变体（多 `stage` 参数，抛 `PERMISSION_DENIED` 而非 `INVALID_ARGUMENT`）。**本轮只合并 binding + object 两个同版本到共享 helper。**

#### 3.2.5 `requireUserId` —— 2 处完全相同

| 文件 | 作用域 |
|------|--------|
| `attachment_binding.ts` | `private static` |
| `attachment_object.ts` | `private static` |

#### 3.2.6 `requireCompanyId` —— 2 处仅 stage 联合类型不同

| 文件 | stage 类型 |
|------|-----------|
| `attachment_binding.ts` | `'bind' \| 'unbind' \| 'descriptor'` |
| `attachment_object.ts` | `'prepare' \| 'finalize'` |

#### 3.2.7 更多 2 处重复

| 函数 | 文件 |
|------|------|
| `toDate(value)` | `attachment_binding.ts` + `attachment_object.ts` |
| `normalizeOptionalNonNegativeInt(value)` | `attachment_binding.ts` + `attachment_object.ts` |
| `normalizePrincipal(raw)` | `attachment_binding.ts` + `attachment_object.ts` |
| `asRecord(input)`（null vs undefined 差异） | `_owner_authorization.ts` + `attachment_object.ts` |
| `DEFAULT_GC_BATCH_SIZE = 200` | `attachment_object.ts` + `attachment_mutation_ledger.ts` + `upload_session.ts` |

### 3.3 P1（中优先，结构性重复）

#### 3.3.1 GC 分页循环 —— 5+ 处重复

`attachment_object.ts`（2 处）、`attachment_mutation_ledger.ts`（1 处）、`upload_session.ts`（1 处）都使用相同的 `for(;;)` + `Search` + `break` 批处理分页模式。Core 已有 `normalizeOffset`、`normalizeLimit`、`paginateAndWrap`，可配合构造通用分页迭代器。

#### 3.3.2 `mustLoad*` 模式 —— 5+ 处

`mustLoadBinding`、`mustLoadActiveBinding`、`mustLoadUploadSession` 等方法结构一致：

```ts
const rows = await this.Search([...conditions], { limit: 1 });
if (!rows[0]) throw newDocumentError({ code: NOT_FOUND, ... });
return rows[0];
```

#### 3.3.3 错误抛出链 —— 52 处

`newDocumentError({...}).withGrpcCode(...).withMetadata(...)` 模式在 4 个文件中出现 52 次。可提供 `throwDocumentError(code, message, grpcCode, meta?)` 收敛。

#### 3.3.4 `@Field` CompanyId —— 5 个模型重复

所有 5 个模型文件中 CompanyId 字段定义完全一致：

```ts
@Field({ type: 'ManyToOneRef', targetModel: 'base.Company', column: { size: 20, notNull: true, index: true } })
CompanyId: string;
```

这是框架约定，不宜抽取（会破坏装饰器语义）。仅作记录。

### 3.4 超大文件拆分需求

#### 3.4.1 `attachment_object.ts`（1,269 行）

职责混合：

- 模型字段定义（~50 行）
- 环境工具（`backendEnv`、`resolvePositiveIntEnv`、`parseNowInput`、`asRecord`，~35 行）→ 抽到 core / `_helpers.ts`
- 参数归一化（`normalize*` 系列 + 类型定义 Normalized*Req，~250 行）
- PrepareUpload 业务（~25 行委托 + 归一化）
- FinalizeUpload 业务（~30 行委托 + 归一化 + 权限）
- AuthorizeUploadPut 业务（~60 行）
- CommitUploadPut 业务（~85 行）
- GC 业务（`RunGarbageCollection` + `garbageCollectUnboundObjects`，~280 行）
- Cleanup state 管理（`readCleanupState`、`writeCleanupState`、`computeRetryBackoffSeconds`，~70 行）
- 各种 assert/mustLoad/require 辅助（~200 行）

建议拆分（Phase C）：

- `attachment_object.ts`：保留模型定义 + 薄委托（~120 行）
- `_attachment_upload_workflow.ts`：Prepare / Finalize / Authorize / Commit 上传流水线
- `_attachment_gc.ts`：GC 逻辑 + cleanup state 管理
- 归一化工具合并到共享 helper（Phase A/B）

#### 3.4.2 `attachment_binding.ts`（1,131 行）

职责混合：

- 模型字段定义（~40 行）
- 参数归一化（`normalize*` 系列 + Normalized*Req 类型，~200 行）
- Bind 业务（~80 行）
- Unbind 业务（~60 行）
- BatchDescribe 业务（~80 行）
- ResolveDownloadContent 业务（~100 行）
- 各种 mustLoad/assert/find 辅助（~400 行）
- require/throw 工具（~50 行）

建议拆分（Phase C）：

- `attachment_binding.ts`：保留模型定义 + 薄委托（~120 行）
- `_attachment_binding_ops.ts`：Bind / Unbind / BatchDescribe / ResolveDownloadContent 业务逻辑
- 归一化 + 辅助工具合并到共享 helper（Phase A）

### 3.5 设计要点（参考 base 经验）

提取的业务函数接收接口类型而非 `typeof AttachmentBinding` / `typeof AttachmentContent`，避免循环依赖：

```ts
type ModelOps = {
  Search: (condition: unknown, options?: unknown) => Promise<unknown[]>;
  Browse: (condition: unknown, options?: unknown) => Promise<unknown[]>;
  Update: (id: string, values: unknown) => Promise<unknown>;
};
```

模型侧仅保留薄委托 + 类型导出 + 字段定义。

## 4. 抽取边界（Document → Core）

### 4.1 可上提到 Core 的纯函数

以下 4 个函数无领域依赖、纯计算，适合上提：

| 函数 | 目标文件 | 说明 |
|------|---------|------|
| `asRecord(input)` | `core/utils/normalization.ts` | 纯类型守卫，统一返回 `null` |
| `parseNowInput(nowISO?)` → `parseISODate` | `core/utils/` (新文件或并入 normalization) | 纯日期解析 |
| `toDate(value)` | `core/utils/` 同上 | 纯日期转换 |
| `normalizeOptionalNonNegativeInt(value)` | `core/utils/normalization.ts` | 与已有 `normalizeOffset` 语义相近 |

### 4.2 可直接复用 Core 已有工具

| Document 当前 | 替换为 Core | 注意事项 |
|-------------|------------|---------|
| `normalizeOptionalText(v)` | `normalizeOptionalString(v)` from `@/core/service/utils/normalization` | 语义完全一致 |
| `backendEnv()` | `getBackendEnv()` from `@/core/service/runtime/env/backend_env` | A1 后优先级一致，可直接替换 |
| `resolvePositiveIntEnv(keys, fallback)` | `getBackendEnvPositiveInt(keys, fallback)` from `@/core/service/runtime/env/backend_env` | A1 本轮补齐多 key 能力 |

### 4.3 留在 Document 域的

| 函数 | 原因 |
|------|------|
| `requireText(value, fieldName)` | 抛出 `newDocumentError(INVALID_ARGUMENT)` |
| `requireUserId()` | 依赖 `BaseModel.ensureUserId()` + document 错误 |
| `requireCompanyId(stage)` | 依赖 `BaseModel` + document 错误 |
| `normalizePrincipal()` | 依赖 `PrincipalContext` 类型 |
| `normalizeChecksum()` | document 领域校验语义 |
| 各 `normalize*Req()` | 依赖 document contracts 类型 |
| `throwDocumentError()` | document 错误域 |
| GC 分页循环 | 依赖 `this.Search()`（BaseModel） |
| `mustLoad*` | 依赖 `this.Search()` + document 错误 |

## 5. 分阶段实施

### 5.1 阶段总览

```
Phase A (域内去重 + Core 复用)     ← 当前优先
├── A1: 上提纯函数到 core
└── A2a/A2b: document 侧替换 + error 层收敛

Phase B (上传流水线集中)
├── B1: 将 attachment_object.ts 的上传流水线纯函数集中到 _upload_helpers.ts

Phase C (超大文件拆分)
├── C1: attachment_object.ts → 模型壳 + _attachment_upload_workflow.ts + _attachment_gc.ts
└── C2: attachment_binding.ts → 模型壳 + _attachment_binding_ops.ts

Phase D (样板缩减)
├── D1: GC 分页循环通用化
├── D2: mustLoad* 模式统一
└── D3: 错误构造链收敛
```

### 5.2 Phase A1：上提纯函数到 Core

**目标**：先在 core 落地纯函数与 env 工具增强，不改 document 调用点。

**变更文件**：

1. 修改 `modules/core/service/utils/normalization.ts`
   - 新增 `asRecord(input: unknown): Record<string, unknown> | null`
   - 新增 `normalizeOptionalNonNegativeInt(value: unknown): number | undefined`
2. 新增 `modules/core/service/utils/date.ts`
   - 新增 `parseISODate(iso?: string): Date`
   - 新增 `toDate(value: unknown): Date | undefined`
3. 修改 `modules/core/service/runtime/env/backend_env.ts`
   - `getBackendEnv()` 统一为 `__choysumBackendEnv` 优先
   - `getBackendEnvPositiveInt()` 支持 `string | string[]`（兼容现有单 key 调用）
4. 新增 `modules/core/service/utils/date.test.ts`
5. 修改 `modules/core/service/utils/normalization.test.ts`（补充新函数测试）
6. 修改 `modules/core/service/runtime/env/backend_env.test.ts`（新增优先级冲突与多 key 测试）

**测试项**：

1. `asRecord`：null、undefined、数组、普通对象、函数、基本类型。
2. `normalizeOptionalNonNegativeInt`：undefined、null、正数、零、负数、NaN、Infinity、字符串数字。
3. `parseISODate`：undefined、空串、合法 ISO 字符串、非法字符串。
4. `toDate`：Date 实例（合法/NaN）、字符串、undefined、null。
5. `getBackendEnv`：global/meta 同时存在时 global 优先。
6. `getBackendEnvPositiveInt`：多 key 顺序命中（数组签名）、单 key 兼容（字符串签名）、无效值 fallback。
7. 运行 `go run . test typecheck --all`。
8. 运行 `go run . test unit --all`。

**验收标准**：

1. 4 个函数行为与当前 document helper 完全一致。
2. env 读取优先级在 core 内部完全一致。
3. `getBackendEnvPositiveInt` 多 key 能力与单 key 兼容性同时成立。
4. 不引入任何 domain 依赖。
5. 新增测试覆盖所有边界分支。

### 5.3 Phase A2：Document 侧替换与去重（拆分为 A2a / A2b）

**目标**：消除 document 模型间的横向重复，接入 core 工具。

执行策略：A2 强制拆分为两个连续 PR。

1. A2a（替换与去重）：不改 `error.ts`，先完成 env/date/normalization/helper 收敛。
2. A2b（错误层收敛）：只改 `error.ts` 与模型层错误调用点，把链式错误构造统一到 `throwDocumentError`。

**A2a 变更文件**：

1. 新增 `modules/document/service/models/_helpers.ts`
   - `requireText(value, fieldName): string`（从 binding + object 收敛）
   - `requireUserId(): string`（从 binding + object 收敛）
   - `requireCompanyId(stage: string): string`（从 binding + object 收敛，统一 stage 为 `string`）
   - `resolveGcBatchSize(): number`（统一 3 处 `DEFAULT_GC_BATCH_SIZE`）
2. 新增 `modules/document/service/tests/_helpers.test.ts`
3. 修改 `modules/document/service/models/attachment_binding.ts`
   - 替换 `backendEnv()` → `getBackendEnv()` from core
   - 替换 `normalizeOptionalText()` → `normalizeOptionalString()` from core
   - 替换 `asRecord()` → `asRecord()` from core（如有使用）
   - 替换 `toDate()` → `toDate()` from core
   - 替换 `normalizeOptionalNonNegativeInt()` → from core
   - 替换 `requireText` / `requireUserId` / `requireCompanyId` → from `_helpers.ts`
4. 修改 `modules/document/service/models/attachment_object.ts`
   - 同上所有替换 + `parseNowInput()` → `parseISODate()` from core
5. 修改 `modules/document/service/models/attachment_mutation_ledger.ts`
   - 替换 `backendEnv()` / `resolvePositiveIntEnv()` / `parseNowInput()` → core 对应函数
   - 替换 `DEFAULT_GC_BATCH_SIZE` → `resolveGcBatchSize()`
6. 修改 `modules/document/service/models/upload_session.ts`
   - 同上替换（含 `resolvePositiveIntEnv` → core 多 key 函数）

**A2b 变更文件**：

1. 修改 `modules/document/service/error.ts`
   - 新增 `throwDocumentError`（或 `createDocumentError + throwDocumentError`）
   - `throwDocumentError` 返回类型为 `never`，并统一封装 `newDocumentError(...).withGrpcCode(...).withMetadata(...)`
2. 修改 `modules/document/service/models/attachment_binding.ts`
   - 替换内联错误构造链 → `throwDocumentError()` from `../error`
3. 修改 `modules/document/service/models/attachment_object.ts`
   - 同上替换
4. 修改 `modules/document/service/models/attachment_mutation_ledger.ts`
   - 同上替换
5. 修改 `modules/document/service/models/stored_content.ts`（如涉及）
   - 同上替换

**预计消除成果**：

| 重复项 | 消除前 | 消除后 |
|--------|--------|--------|
| `backendEnv()` | 3 处 | 0 处（→ core `getBackendEnv`） |
| `resolvePositiveIntEnv()` | 3 处 | 0 处（→ core `getBackendEnvPositiveInt(keys, fallback)`） |
| `parseNowInput()` | 3 处 | 0 处（→ core `parseISODate`） |
| `normalizeOptionalText()` | 3 处 (含 authz) | 1 处（authz 保留，其余 → core） |
| `requireText()` | 2 处 | 1 处（`_helpers.ts`） |
| `requireUserId()` | 2 处 | 1 处（`_helpers.ts`） |
| `requireCompanyId()` | 2 处 | 1 处（`_helpers.ts`） |
| `toDate()` | 2 处 | 0 处（→ core） |
| `normalizeOptionalNonNegativeInt()` | 2 处 | 0 处（→ core） |
| `asRecord()` | 3 处 | 1 处（authz 保留，其余 → core） |
| `DEFAULT_GC_BATCH_SIZE` | 3 处 | 1 处（`_helpers.ts`） |
| 错误构造链 3 行模式 | 52 处 | 52→`error.ts` 的 `throwDocumentError` 单调用 |

**预计代码量变化**：

- `attachment_object.ts`：~1,269 → ~1,120（净减少 ~149 行）
- `attachment_binding.ts`：~1,131 → ~980（净减少 ~151 行）
- `attachment_mutation_ledger.ts`：~130 → ~95（净减少 ~35 行）
- `upload_session.ts`：~249 → ~215（净减少 ~34 行）
- `_helpers.ts`：0 → ~90（净增加 ~90 行）
- `error.ts`：~80 → ~115（净增加 ~35 行，A2b）
- 四个主模型文件小计：~2,779 → ~2,410（净减少 ~369 行）
- A2 合计（含 `error.ts` 与 `_helpers.ts`）：~2,859 → ~2,615（净减少 ~244 行）

**测试项**：

1. A2a：运行 `go run . test unit document --be --fe=false`（全部 49 测试通过）。
2. A2a：运行 `go run . test typecheck --all`。
3. A2a：重点回归上传流水线、GC、Bind/Unbind 的边界场景。
4. A2a：新增 `_helpers.test.ts` 覆盖所有共享 helper。
5. A2b：新增 `error.ts` 对 `throwDocumentError`（或组合 API）的单测。
6. A2b：新增规则校验（如 grep/lint）确认模型层不再新增链式 `newDocumentError(...).withGrpcCode(...).withMetadata(...)`。

**验收标准**：

1. 所有 49 个现有测试不变（无新增、无删除、无改名）。
2. 新增 helper 与 error 便利函数测试覆盖全分支。
3. 模型层错误抛出统一经 `throwDocumentError`，不再引入新的链式错误构造样板。
4. 三处 GC 入口的 batch size 行为不变。

### 5.4 Phase B1：上传流水线纯函数集中

**目标**：将 `attachment_object.ts` 中 Prepare/Finalize/Authorize/Commit 的纯规范化函数集中到一个文件，减少文件内混杂度。

**变更文件**：

1. 新增 `modules/document/service/models/_upload_helpers.ts`
   - NormalizedPrepareUploadReq 类型
   - NormalizedAuthorizeUploadPutReq 类型
   - NormalizedCommitUploadPutReq 类型
   - `normalizePrepareUploadReq(req)`
   - `normalizeAuthorizeUploadPutReq(req)`
   - `normalizeCommitUploadPutReq(req)`
   - `assertUploadSessionPrincipal(session, principal, stage)`
   - `assertFinalizeIdentity(session)`
   - `assertPrepareReplayConsistency(session, normalized)`
   - 各 upload session TTL / max size 常量
2. 修改 `modules/document/service/models/attachment_object.ts`
   - 移除上述函数/类型/常量定义
   - 改为从 `_upload_helpers.ts` import
3. 新增 `modules/document/service/tests/_upload_helpers.test.ts`（如规范化逻辑有可独立测试的纯函数）

**预计代码量变化**：

- `attachment_object.ts`：~1,120 → ~890（净减少 ~230 行）
- `_upload_helpers.ts`：新增 ~260 行

**测试项**：

1. 运行 `go run . test unit document --be --fe=false`。
2. 运行 `go run . test typecheck --all`。

**验收标准**：

1. 上传流水线 4 个公开方法行为不变。
2. `attachment_object.ts` 降至 ~900 行以内。

### 5.5 Phase C1：拆分 attachment_object.ts

**目标**：将 GC 和上传流水线业务逻辑从模型文件中分离，模型侧仅保留字段定义 + 薄委托。

**变更文件**：

1. 新增 `modules/document/service/models/_attachment_upload_workflow.ts`
   - `prepareUpload(modelOps, params)` — PrepareUpload 业务逻辑
   - `finalizeUpload(modelOps, params)` — FinalizeUpload 业务逻辑
   - `authorizeUploadPut(modelOps, params)` — AuthorizeUploadPut 业务逻辑
   - `commitUploadPut(modelOps, params)` — CommitUploadPut 业务逻辑
2. 新增 `modules/document/service/models/_attachment_gc.ts`
   - `runGarbageCollection(modelOps, nowISO?)` — GC 业务逻辑
   - `garbageCollectUnboundObjects(modelOps, ...)` — 子流程
   - Cleanup state 管理函数
3. 修改 `modules/document/service/models/attachment_object.ts`
   - 4 个 `public static async` 方法改为薄委托
   - 移除内联业务逻辑

**设计要点**：

- 提取函数接收 `ModelOps` 接口（`{ Search, Browse, Update, ... }`），避免 `import` 循环。
- 模型侧保持：字段定义 + `@Constraint`（如有）+ 薄委托 + 类型导出。

**预计代码量变化**：

| 文件 | 变更 |
|------|------|
| `attachment_object.ts` | ~890 → **~120** |
| `_attachment_upload_workflow.ts` | 新增 ~350 |
| `_attachment_gc.ts` | 新增 ~320 |
| `_upload_helpers.ts` | 保持 ~260 |

**测试项**：

1. 运行 `go run . test unit document --be --fe=false`。
2. 运行 `go run . test typecheck --all`。

**验收标准**：

1. PrepareUpload / FinalizeUpload / AuthorizeUploadPut / CommitUploadPut / RunGarbageCollection 签名和行为不变。
2. 模型文件降至 ~120 行（达到 base `currency.ts` 90 行的量级）。

### 5.6 Phase C2：拆分 attachment_binding.ts

**目标**：将 Bind/Unbind/BatchDescribe/ResolveDownloadContent 业务逻辑分离。

**变更文件**：

1. 新增 `modules/document/service/models/_attachment_binding_ops.ts`
   - `bindAttachment(modelOps, params)` — Bind 业务逻辑
   - `unbindAttachment(modelOps, params)` — Unbind 业务逻辑
   - `batchDescribeAttachments(modelOps, params)` — BatchDescribe 业务逻辑
   - `resolveDownloadContent(modelOps, params)` — ResolveDownloadContent 业务逻辑
   - `mustLoadActiveAttachmentContent` / `mustLoadActiveStoredContentById` 等辅助
2. 修改 `modules/document/service/models/attachment_binding.ts`
   - 4 个 `public static async` 方法改为薄委托

**预计代码量变化**：

| 文件 | 变更 |
|------|------|
| `attachment_binding.ts` | ~980 → **~120** |
| `_attachment_binding_ops.ts` | 新增 ~860 |

**测试项**：

1. 运行 `go run . test unit document --be --fe=false`。
2. 运行 `go run . test typecheck --all`。

**验收标准**：

1. Bind / Unbind / BatchDescribe / ResolveDownloadContent 签名和行为不变。
2. 模型文件降至 ~120 行。

### 5.7 Phase D：样板缩减

**Phase D1**：GC 分页循环通用化

- 提供 `paginateBatch(modelOps, condition, processor, options)` 工具，消除 5+ 处 `for(;;)` + `Search` + `break` 重复。

**Phase D2**：`mustLoad*` 模式统一

- 提供 `mustLoadOne(modelOps, condition, notFoundMessage)` 通用加载器。

**Phase D3**：错误构造链收敛

- 已在 Phase A2b 通过 `throwDocumentError()` 完成大部分收敛。

## 6. 执行顺序与 PR 拆分

建议按以下顺序提交，每个 PR 独立可回滚：

Phase A 强制拆分为两个独立 PR（不可合并提交）：

1. A1 仅允许修改 core 相关文件。
2. A2a 仅允许修改 document 模型与 helper（不改 `error.ts`）。
3. A2b 仅允许修改 `error.ts` 与模型错误调用点收敛。
4. A2a/A2b 必须基于 A1 合并后的主干分支 rebase。

| 顺序 | PR | 内容 | 风险 |
|------|-----|------|------|
| 1 | **A1** | 上提 4 个纯函数并增强 core env 工具（新增，不改 document） | 极低 |
| 2 | **A2a** | document 侧去重 + 接入 core 工具（不改 `error.ts`） | 低 |
| 3 | **A2b** | `error.ts` 落地 + 模型错误链收敛 | 低 |
| 4 | **B1** | 上传流水线纯函数集中到 `_upload_helpers.ts` | 低 |
| 5 | **C1** | 拆分 `attachment_object.ts` | 中 |
| 6 | **C2** | 拆分 `attachment_binding.ts` | 中 |
| 7 | **D1-D3** | GC 分页/mustLoad/错误链样板缩减 | 低 |

## 7. 风险、回滚与闸门

### 7.1 主要风险

1. A1 会将 core env 读取优先级统一为 `__choysumBackendEnv` 优先，需对 base/meta/document 做回归，确认无旧顺序依赖。
2. `normalizeOptionalText` → `normalizeOptionalString` 切换时，`String(value || '')` vs `String(value ?? '')` 对 `0` / `false` 的处理差异（实际 document 中不会传 number/boolean 给这些函数）。
3. `attachment_object.ts` / `attachment_binding.ts` 拆分后 model 间动态 `import()` 路径变化可能导致循环依赖。

### 7.2 回滚策略

1. 每个 Phase 独立 PR，按 Phase 回滚。
2. 出现回归直接回退当期 PR，不保留兼容壳。
3. 回退后补充失败样例再重提。

### 7.3 强制测试门禁

每个 Phase 合并前，必须在仓库根目录执行并通过：

1. `go run . test typecheck --all`
2. `go run . test unit document --be --fe=false`
3. `go run . test unit --all`（A1 阶段）
4. `go run . test e2e --all`（C1/C2 合并候选阶段）

## 8. 执行状态

| Phase | 状态 | 说明 |
|-------|------|------|
| 测试文件迁移 | ✅ 已完成 | 5 个 test 文件移至 `service/tests/` |
| **Phase A1** | ✅ 已完成 | core: +asRecord/normalizeOptionalNonNegativeInt/parseISODate/toDate; env 统一优先级 + 多 key |
| **Phase A2a** | ✅ 已完成 | document 去重: _helpers.ts + 替换 env/date/normalization 调用 |
| **Phase A2b** | ✅ 已完成 | error.ts: +throwDocumentError; 模型层错误链收敛 |
| **Phase B1** | ✅ 已完成 | 上传流水线纯函数集中到 _upload_helpers.ts (889+240) |
| **Phase C1** | ✅ 已完成 | GC 逻辑提取到 _attachment_gc.ts + GcModelOps |
| **Phase C2** | ⬜ 待开始 | 拆分 attachment_binding.ts |
| **Phase D** | ⬜ 待开始 | 样板缩减 |

## 9. 交付物

1. 本文档：document 模型重构方案。
2. 实施配套：
   - 重复消减清单（before/after）
   - 回归用例清单
   - 门禁命令执行记录
   - 回滚说明
