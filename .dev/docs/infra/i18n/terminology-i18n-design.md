<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# Choysum 术语 i18n 设计方案

更新时间：2026-07-20（D1–D21；**`_t` / `_lt`** 双 helper，硬切删除 `output`；冻结 FE 术语引用显示边界；**D19** `defineModelActions`/`entityTitle`；**D20 否决**种子/demo JSON `{ "_t" }`；**D21** L1–L9）

> **改造中（2026-08）：** 与 **injectappmodel / `TranslationTerm` MetaModel**、删除 Go `{app}.I18n` 业务 RPC、删除 `/web/i18n/terms`、Editor 路径 α、PO 按 module、共享 packaged 写 helper 等相关的条款，**冲突时以** [terminology-i18n-injectappmodel-design.md](./terminology-i18n-injectappmodel-design.md) **为准**（一次性改造；冻结结论见该文 §2；落地任务见该文 §12）。主文档 §1 / §3 / §6 / §8 / §13 / §14 的完整回写在改造 **P6**；下文对 D5·D14·D17 等已加 **「将被废止/改写」** 批注。

**怎么读：**

| 读者 | 路径 |
|---|---|
| 拍板 / 评审 | §0 → **§1** → **§3** → §13；**改造冲突 → [injectappmodel 设计](./terminology-i18n-injectappmodel-design.md)** |
| 实现 BE / Gateway | §3 → §5–§6 → §8 → §9 → **§13** → **§14**；改造实现跟 injectappmodel 设计 §12 |
| 实现 FE | §5.1（`_t`/`_lt`）→ §7 → §8.2 → §14 PR-S4*；改译 UI 跟 injectappmodel §6.2 / PR-P4 |
| 实现 CLI / extract | §5.1 → §5.4 → §9 → §13 D16·D18·D21 → §14 PR-S1* |

分层口诀：**Go（DDL + TermStore/cache + `{app}.I18n` + PO import + `$choysum.i18n`）→ TS（`_t` / `_lt` / scope / lang 门面）→ 宿主 Gateway → Terminology Editor / FE**。

> 改造后目标口诀见 [injectappmodel 设计 §0](./terminology-i18n-injectappmodel-design.md)：表 ← TS `TranslationTerm`（ORM + `GetTranslations`）← Gateway translations；改译 ← 模型客户端（路径 α）；`_t` ← Go 查词缓存；PO packaged ← 共享 helper。

语言口诀：**lang = 术语码（`zh_CN`）；locale = UI 键/格式（`zh-CN`）；`base.Language` = 语言主数据。**

---

## 0. 术语表（Glossary）

后文首次出现可带英文对照，之后用中文简称。

| 中文 | English | 含义 |
|---|---|---|
| **术语 i18n** | terminology i18n | 显式 `_t` 标记的固定文案（≠ 数据 i18n / 业务字段多语言值） |
| **术语词条 / 词条** | term | `TranslationTerm` 一行 |
| **Src** | msgid | 英文源文；表字段 `Src`（UI 勿与字段 `Source` 混淆） |
| **Value** | msgstr | 译文；表字段 `Value` |
| **Scope** | scope | 查词键；PO 中为 `msgctxt` |
| **location** | location | auto Scope 可选后缀（`path[@location]`）；**不是** Scope 本身 |
| **msgctxt** | msgctxt | Scope 在 PO 中的序列化（GNU gettext Contexts） |
| **POT / PO** | pot / po | gettext 模板 / 语言文件 |
| **Module** | module | manifest 模块名；表字段 `Module` |
| **Application** | application | 部署/服务边界；Gateway dial 目标 |
| **lang / locale** | lang / locale | 术语码 vs 格式码（见口诀） |
| **Language** | — | `base.Language` 语言主数据；`IsActive` 控制术语启用 / 切换项 |
| **Source** | source | `packaged`（PO）/ `override`（编辑器） |
| **Kind** | kind | 默认 `literal`；协议、存储、PO 与 bridge 保留显式自定义 kind 的通用兼容能力 |
| **存储层** | store layer | `{app}_translation_term` + Go TermStore/cache（§13 D17） |
| **应用服务** | app service | `{app}.I18n`（Go；仅宿主 dial；§13 D17） |
| **宿主 i18n Gateway** | host i18n Gateway | 浏览器唯一 HTTP 入口（词表 + 术语目录） |
| **运行时词表** | runtime catalog | `GET /web/i18n/translations` |
| **术语目录 API** | term catalog API | `GET` / `PATCH /web/i18n/terms` |
| **术语翻译编辑器** | Terminology Editor | `TerminologyEditor.vue`；菜单名「术语翻译」 |
| **壳层词条** | shell terms | web Application 自有 layout/chrome 文案 |
| **内部持久化表** | internal persisted table | 有表、无公共 CRUD UI；MVP 不注册 MetaModel（§13 D14） |
| **`$choysum.i18n` bridge** | i18n QuickJS bridge | sync `t(module,lang,scope,src[,kind])` |
| **`_t`** | translate (text) | 立即查词 → `string`；允许插值；BE bridge / FE Composer（§5.1·**D18·D21**） |
| **`_lt`** | lazy term reference | 钉可序列化 `TermReference`；**不**查词、**禁止**插值；≠ Odoo LazyGettext 隐式再译（§5.1·**D21**） |
| **`TermReference`** | term reference | 可序列化术语坐标 `{ key, module, scope, src, kind }`；不查词；wire 字段名仍为 `titleText` / `pageTitleText` / 字段标题 `stringText`（§5.5）；selection **禁止** `labelText` |
| **`translateTerm`** | Composer adapter | FE 非 template 的唯一术语引用显示适配器：`translateTerm(composer, reference, fallback)`（§7.2） |
| **统一术语视图** | unified term view | UX 聚合观感；非独立术语库 |
| **词条所属模块** | term owner module | 声明显式 `_t` 的物理 Module |
| **组件提示** | component hint | 由 Scope path 推导的展示分组；非表字段 |
| **语言切换 remount** | locale remount | 切语言后 remount；整页 reload 为兜底（§13 D9） |
| **术语重载** | terminology reload | 改译后本会话重拉词表；不做跨会话推送 |
| **termHash / catalogHash** | — | 单 app / 多 app 按 lang 的内容哈希（§13 D2） |
| **moduleNames** | moduleNames | GetTranslations 过滤（宿主填；浏览器不传） |
| **canonical** | `{ module, scope, src, kind }` | 对内词条坐标；kind 缺省 `literal` |

---

## 1. 范围、冻结结论与反模式

### 1.1 范围

| 类型 | 含义 | 存储 | 本文 |
|---|---|---|---|
| **术语 i18n** | 显式 `_t` 标记的固定文案（含 menu/route/action、校验文案、字段 label 等） | 表 + PO + Gateway | ✅ |
| **数据 i18n** | 业务记录上可译字段值（含模块打包的 seed/demo 字段，如 `Company.Name`） | 模型 JSON/JSONB 列（[data-i18n-design.md](./data-i18n-design.md)） | ❌ |

**边界（冻结）：** `data/*.json` / `demo/*.json` 里的业务字段值属于 **数据 i18n**，**禁止**用 `{ "_t" }` / 术语表 / extract 入 pot。术语 i18n 只管代码与静态元数据里的 `_t`。

### 1.2 冻结结论（索引）

细则见对应章节与 **§13**；冲突时以 §13 为准。

1. **英文 msgid + per-app 表**：`_t('Save')`；权威存储 `{app}_translation_term`；跳过 `core`；无 core `@Model`；MVP 无 MetaModel → §6、**D13·D14**。
2. **Go 权威运行时**：TermStore/cache + `{app}.I18n` + PO import 在 Go；TS 仅门面；`_t` 经 sync `$choysum.i18n` → **D17**、§6.2、§8.3.0。
3. **三层拓扑**：存储层 / `{app}.I18n` / 宿主 Gateway（读 translations + 写 terms）→ **§3**。
4. **查词键** `(Module, Lang, Scope, Src, Kind)`；IMD/patch 词条归声明 Module → §5.3、**§4.1**。
5. **切语言**：UI + `RequestContext.lang` + BE 一致；S4-MVP 默认 reload → **D9**、§7.3。
6. **CLI**：`extract` / `sync` / `status`；无 `import|export|merge|update|missing` → **§9.2**。
7. **在线改译**：仅经 Gateway terms；单 Terminology Editor；默认跨 App 搜；`web.I18n` 仅壳层 → **§8.6**、**D8**。
8. **权限**：translations Public；terms Authenticated+角色；GetTranslations 仅内部；UpdateTerm ACL default-deny → **§8.7**、**D1·D6**。
9. **gettext 契约**：PO 格式跟 GNU；**`_t` + `_lt`**（硬切，无 `output`）；Vue 响应性由标准 computed 建立；gotext 仅读写 PO → **§5.4**、**§5.1**、**§9.3**、**D18·D21**。
10. **落地**：S0–S6（§11）；S0 顺序 **D15**；PR 细单 **§14**。**术语×injectappmodel 改造**落地见 [injectappmodel 设计 §12](./terminology-i18n-injectappmodel-design.md)（P0–P6；与下方部分 §14 任务冲突时以该文为准）。
11. **model action**：`defineModelActions` 的 `entityTitle`/`titles` 用 **`_lt`** + `TitleText` → **§5.5**、**D19**。
12. **种子/demo JSON**：属数据 i18n；**禁止** `{ "_t" }` 入术语管线 → **§3.3**、**§5.6**、**D20（否决）**。

### 1.3 反模式索引（明确不做）

完整清单以此节为准；§14 Reviewer 只抽查本表 + PR 特有项。

> **改造批注（草稿，P6 回写）：** 「通用 Model CRUD 作改译」「MVP MetaModel」等行将被改写——改造后允许 per-app `TranslationTerm` MetaModel + Editor 路径 α（标准模型客户端）；仍禁止跨 app Gateway terms 聚合、禁止 `core.TranslationTerm`。详见 [injectappmodel 设计 §2 / §8](./terminology-i18n-injectappmodel-design.md)。

| 类别 | 不做 |
|---|---|
| **范围** | 数据 i18n；表挂业务 `res_id`；独立 `i18n` App 作唯一存储（含「唯一库 + 副本同步」） |
| **产物** | `dist/web/i18n/**` 快照；GlobalWebBuild 写术语 JSON |
| **运行时** | `_t` 跨服务查词；TS 权威 cache / `app_i18n_service` / TS PO import；gotext/`node-gettext` 作查词权威；async `$choysum.db.query` 热路径 |
| **标记 API** | call/factory `output` 选项；单一 `_t` 靠 output 切换；Odoo 式 LazyGettext（`_lt` 隐式再查词）；text `_t` 写入静态 wire；Selection 显式 `labelText` / FE `translateTerm(selection)`；`label: _t(...)`；`output` 双读/兼容窗 |
| **FE 术语引用显示** | 直接显示 `reference.src`；按 menu/breadcrumb/selection/router 各造翻译 wrapper；template 调 `translateTerm`；TS/h-render/router 直接拼 `$t` 或重复 `te`/`t` |
| **提取** | `xgettext` / `xgotext` / `gettext-extractor` 作权威；第二套 TS/script 解析；`backendtsparser.Parse()` 抽 `_t`；扫描未显式标记的 `label` / `title` / selection / JSON `Name`；MVP plural API |
| **Gateway / 服务** | Gateway 跨库；浏览器传 `moduleNames`；浏览器直连 `{app}.I18n`；宿主聚合塞进业务 ApplicationService；`web.I18n` 跨 App 聚合；entryPolicy 整组 Public `*.I18n/*` |
| **改译 / ACL** | 每 App TranslationTerm 菜单；通用 Model CRUD 作改译；FieldRule/RecordRule 当主写 ACL；改译全会话广播；把 patch 词条并入基座模块表 |
| **模型** | core `@Model('TranslationTerm')` / `core_translation_term`；MVP MetaModel |
| **CLI** | `choysum i18n import|export|merge|update|missing`（能力在 sync/status） |
| **其它** | `_t.distinct` / 行号式 location；Locale.IsActive 驱动术语；全语言进热缓存 |

### 1.4 相关文档

- `.dev/docs/roadmap.md`
- `.dev/docs/infra/registry/manifest-package-json-merger.md`
- `.dev/docs/infra/registry/module-management-bootstrap-refactoring.md`
- `.dev/docs/infra/architecture-enterprise-boundary.md`
- `.dev/docs/infra/cmd/choysum_cli.md`
- `.dev/docs/infra/build/esm-dependency-architecture.md`
- `.dev/docs/document/document_architecture_refactor.md`（Gateway / 分阶段体例）
- `.dev/docs/core/service/constraint_refactor_plan20260711.md`（PR / DoD 体例）
- `.dev/docs/auth/permission_system_design.md`
- `.dev/docs/auth/acl_wildcard_optimization.md`
- `.dev/docs/core/web/field-metadata-fields-get-design.md`（字段标题 `string` + `stringText`；推荐 `_lt`）
- `.dev/docs/base/language-merge-optimization.md`（`base.Language` 单模型；术语码 POSIX）
- [data-i18n-design.md](./data-i18n-design.md)（业务字段值多语言；同行 JSONB；与本文硬隔离）
- [terminology-i18n-injectappmodel-design.md](./terminology-i18n-injectappmodel-design.md)（**进行中**：TranslationTerm inject、删 Go I18n/terms、Editor α；冲突时以该文为准）

---

## 2. 背景

Choysum：**Module ⊂ Application**（后端部署边界）+ 全局 Web SPA（side-import 各模块 `web` entry）。

需求：模块自带 `i18n/*.po`；前后端 msgid 对齐；请求级 `lang`；每 app 本地词条，跨 app 只传 `lang`。

**现状（待迁移）：** `modules/web/web/i18n/source` + 手写 `zh-CN`；已有 vue-i18n / `i18nStore` / `RequestContext.locale`；`User.Language`、`Language.IsActive` 已预留（语言主数据见 `.dev/docs/base/language-merge-optimization.md`）。

不做静态术语快照：正式源是 Gateway→表；在线改译后快照会双源漂移。缺译回退英文 msgid。

---

## 3. 架构

### 3.1 三层与数据流

| 层次 | 载体 | 谁用 | 在线改译 |
|---|---|---|---|
| **存储层** | `{app}_translation_term` + Go TermStore/cache | BE `_t`（bridge）；`{app}.I18n` | 写表 + 刷 cache + bump termHash |
| **应用服务** | `{app}.I18n`（Go） | **仅**宿主 dial | Get / Search / Update |
| **Gateway 词表** | `GET /web/i18n/translations` | FE 唯一读 | FE 重拉（术语重载） |
| **Gateway 目录** | `GET\|PATCH /web/i18n/terms` | Terminology Editor 唯一管/写 | fan-out 写各 app |
| **编辑器** | Terminology Editor | 设置角色 | 只调 terms API |

```text
源码 _t ──extract──► *.pot ──sync──► *.po
                           │ install / Editor（经 Gateway）改译
                           ▼
                  {app}_translation_term（Go ensure）
                           │
              ┌────────────┴────────────────┐
              ▼                             ▼
     Go TermStore/cache              {app}.I18n（Go）
              ▲                      Get / Search / Update
              │ sync $choysum.i18n              │
        TS _t / scope / lang                    ▼
                                    宿主 Gateway（唯一浏览器入口）
                    ┌────────────────┼────────────────┐
                    ▼                ▼                ▼
           GET …/translations   GET …/terms    PATCH …/terms
```

约束（细节见 §1.3 / §8）：无 `dist/web/i18n`；Gateway 不跨库；浏览器不传 `moduleNames`、不 dial `{app}.I18n`；对外 `messages`、对内 `{ module, scope, src }`；表由 Go ensure，**跳过 `core`**（**D13**）；合部署多表用 Editor 虚拟成统一视图。

### 3.2 角色

| 组件 | 职责 |
|---|---|
| **core（TS）** | `_t`/scope/`resolveRequestLang`；调 `$choysum.i18n`；无 `@Model` / 权威 cache |
| **base** | `Language` 语言主数据；`IsActive` 启用语言（非改译表） |
| **meta** | install 编排；触发 Go PO import；已装目录供 Gateway |
| **各 Application** | 本地表 + 注册 Go `{app}.I18n` |
| **auth** | 试点；`User.Language`；`auth.I18n` |
| **Go `internal/i18n`** | DDL、store、service、import、bridge（**D17**） |
| **ApplicationService** | 挂载 Go `{app}.I18n`（非 handler→JS） |
| **宿主 Gateway** | HTTP 读+写聚合、catalogHash（§8） |
| **web** | 壳层 PO + `web.I18n`；Terminology Editor；切语言 UX |
| **CLI** | `extract` · `sync` · `status`（§9.2） |

### 3.3 与数据 i18n 的边界

| 内容 | 归属 |
|---|---|
| 按钮、UserError message | 术语 |
| menu / route / **model action** title（`defineModelActions` + `_lt`） | 术语（**D19**） |
| `data/*.json` / `demo/*.json` 中的业务字段（如 `Name`） | **数据 i18n**（本文不做；见 [data-i18n-design.md](./data-i18n-design.md)）；写入 DB 原文；展示读记录值 |
| 用户录入的 `partner.Name` / 公司名等 | 数据 i18n（见 [data-i18n-design.md](./data-i18n-design.md)） |
| selection `value` / `label` | value 不译；`label: _lt` 入 pot + FieldsGet `_t` 出已译 string；裸 `label` 不译、不入 pot |
| `{ code, message: _t(...) }` | code 不译；message 术语 |

翻译表 **无业务 `res_id`**。业务字段多语言值不得进入 `TranslationTerm`。

### 3.4 否决形态

| 形态 | 结论 |
|---|---|
| 独立 `i18n` App = 唯一存储，其它全靠 gRPC | ❌ `_t` 热路径不跨服务 |
| 独立库 + 各 App 副本同步 | ❌ 过重 |
| per-app 表 + `{app}.I18n` + 单 Editor | ✅ |
| 编排-only App（不持存储） | 不优先 |

---

## 4. 模块约定

```text
modules/<name>/
  i18n/
    <name>.pot
    zh_CN.po          # 文件名 = Language.Code
  service/            # createTranslate('name')
  web/
```

- 无 `i18n/`：跳过导入，不算错。
- `web/i18n` 仅壳层词条；业务译文不堆 web。
- 内部语言码 `zh_CN`；UI/Intl 映射 `zh-CN`。

### 4.1 IMD / 视图扩展与词条归属（冻结）

roadmap：后端 IMD **仅同 Application**；前端允许编译期跨 App 补丁 UI（如 `auth` xpath 扩展 `web` 的 `OHeader`）。

**词条归属 = 声明源码所在 Module（物理目录），不是被扩展基座所属 Module。**

| 规则 | 含义 |
|---|---|
| extract / Module 列 | 以含显式 `_t` 的源文件所属模块为准 |
| `createTranslate('…')` | 必须与该文件所属模块名一致 |
| Scope | 仍按该文件 `path[@location]` |
| 装卸载 | 卸 `auth` → 只删 `Module=auth`；基座保留 |
| **禁止** | patch 文案合并进基座表；按合成组件名重归属 |

#### 4.1.1 典型：`OHeader`

```text
modules/web/.../OHeader.vue     → Module=web → web 表 / web.I18n
modules/auth/.../OHeader.vue    → 「Log In」等在 auth 源码 → auth 表 / auth.I18n
```

SPA 看到合并后的一个 Header；Gateway 合并 messages 后两侧同时生效。  
Editor 若只筛 `application=web`，会找不到 auth 补丁文案——是**筛错所属模块**，不是丢词。

#### 4.1.2 会踩的坑

1. 按屏幕找词漏掉其它 App 补丁。  
2. 同 msgid 多条（不同 Module/Scope）可并存，勿强行合并。  
3. BE IMD：词条归写出显式 `_t` 的模块。  
4. 误把 auth 文案写进 web pot → 卸载漂移。

#### 4.1.3 Terminology Editor 操作优化

| 能力 | Stage |
|---|---|
| 默认跨 App 搜索 msgid/译文 | S5-MVP |
| 显示 Application / Module / Scope 列；按行内 application 写回 | S5-MVP |
| 组件提示（Scope path 启发式） | S5；更丰富提示须另行设计稳定契约（**D11**） |
| 「相关词条」/ 缺译全局视图 | S5+ / S6 |

```text
搜「Log In」→ application=auth → PATCH terms → auth.I18n/UpdateTerm
→ 术语重载 GET translations → Header 补丁文案更新
```

xpath patch 内新增可见字符串必须 `_t`，且 `createTranslate` 用**当前模块**。

---

## 5. 标记 API 与 Scope

### 5.1 `_t` / `_lt` 与工厂

`createTranslate(module, options?)` 在 **BE（core）与 FE（web）** 同形返回 **`{ _t, _lt }`**（**D21**）。工厂选项仅为默认 `scope` / `path` / `location` / `kind`，**不含** `output`。

| Helper | 返回值 | 行为 |
| --- | --- | --- |
| **`_t`** | `string` | 按当前 lang 查词；允许插值 |
| **`_lt`** | `TermReference` | 不查词、不插值；可 JSON round-trip |

**Odoo 对照（动机，非照搬）：** Odoo `_` ≈ Choysum `_t`（立刻译）；Odoo `_lt`（LazyGettext）**≠** Choysum `_lt`。Choysum `_lt` 对齐的是「早定义、可序列化、显示端再译」；返回值**永不**在隐式 coerce / `__str__` 时查词。显示仍走 §7.2（`$t` / `translateTerm`）。

```ts
const { _t, _lt } = createTranslate('auth', {
  scope: 'service/models/session', // 仅影响未显式传 scope 的调用
});

// 报错 / 日志 / 运行时文案
_t('Save')
_t('User %s not found', login)
_t('Save', { scope: 'game.rescue' })

// 静态元数据（menu / route / model action / @Field.string）
_lt('Access Control')
_lt('Left to right', { scope: 'base.Language.Direction.ltr' })

// 同文件：字段标题用 _lt，报错用 _t
const { _lt: fieldT } = createTranslate('auth', {
  scope: 'auth.model.Session.fields',
});
@Field({ type: 'varchar', string: fieldT('Access Token ID') })
AccessTokenId: string;

// Vue 响应式值由调用方显式 computed；进程内延迟用显式闭包
const saveLabel = computed(() => _t('Save'))
const resolveAccessError = () => _t('Access Error')
```

可选模块辅助（非必须）：`fieldTerms(model)` → 预置 `scope: '<module>.model.<Model>.fields'` 的 `_lt`。

**类型（示意）：**

```ts
export type CreateTranslateResult = {
  _t: (src: string, opts?: TermOptions, ...args: unknown[]) => string;
  _lt: (src: string, opts?: TermOptions) => TermReference;
};
```

- `_lt` **无**插值 rest overload；运行时若收到 primitive/array 第二参或多余参数则抛错。
- `TermOptions` / factory options **禁止** `output` 回潮。
- 旧名 `_td` / `_tr` / `LazyTranslate` / `TextDescriptor` / `TranslateOutput` **已废**；extractor / parser **不再**识别 `output: 'reference'`。

**运行时：** `_t` 后端经 sync bridge 查 Go cache，前端经 vue-i18n Composer + catalog revision；miss 回退 `src`。`_lt` 只规范化 canonical identity 并生成稳定 `key`，不读语言。

**extract / parser：** 识别字面量 `_t('…')`（text）与 `_lt('…')`（及解构别名指向 `_lt` 的 callee）；理解 factory 默认 scope；动态 src / 缺失 scope 的 `_lt` 告警并跳过。

**禁止：** `_t('X', { output: 'reference' })`；`createTranslate(..., { output })`；`_lt('X', {}, arg)` 插值；`@Field({ string: _t('X') })` 用 text `_t` 当静态标题；Selection 显式 `labelText` / `label: _t(...)`（见 §5.5 / D5；**允许** `label: _lt(...)`）。

### 5.2 RequestContext

| 键 | 用途 |
|---|---|
| `lang` | `_t` / 错误 message |
| `locale` | 日期数字格式（≠ lang） |

解析：`baggage lang` → `User.Language` → Company 默认 → `en_US`（**D12d**：勿把 `locale` 当 `lang`）。  
FE：`setLocale` 同步 RequestContext。

### 5.3 Scope

查词键：`(Module, Lang, Scope, Src, Kind)`。PO：`msgctxt` = Scope。

```text
Scope = <path>[@<location>]   # auto
      | <manual-scope>
```

**location 优先级：** ① 手动 `withI18nScope` / `{ scope }` → ② 最近命名符号 → ③ Vue 具名标识 → ④ 仅 path。

Vue `<template>`（S1-MVP，**D12e**）：regex 抽字面量；默认 Scope=`path@template`；S1+ 换 template AST。

禁止：行号、出现序号、`_t.distinct`。硬拆：改英文或手动 scope。

| 场景 | 行为 |
|---|---|
| 不同模块 / 不同 path 同 Src | 并存 |
| 同 location / 仅 path 回退 | 一条 |
| 跨文件同译法 | `_shared` 或公共 helper |

FE：`messages[module][scope][src]`；禁止全局扁平 merge。extract 与 runtime 同算法。

### 5.4 GNU gettext 字段对照

真理源：[PO Files](https://www.gnu.org/software/gettext/manual/html_node/PO-Files.html)、[Contexts](https://www.gnu.org/software/gettext/manual/html_node/Contexts.html)。库选型见 **§9.3** / **D18**。

| gettext | Choysum | 说明 |
|---|---|---|
| `msgid` | `Src` | 英文源文 |
| `msgstr` | `Value` | 译文 |
| `msgctxt` | `Scope` | 必有；无则导入拒绝（**D12c**） |
| `#:` | Comments | 可选；不进查词键 |
| `#. kind:` | Kind | 显式通用兼容；缺省 `literal`；当前自动提取只产生 `literal` |
| plural | — | MVP 不做 |
| pot / po | `modules/<m>/i18n/` | extract / sync / install |

`sync` 语义对齐 `msgmerge`；对外只暴露 `choysum i18n sync`。

### 5.5 静态元数据的显式边界

**menu / route / model action / `@Field.string` 等静态显示元数据**必须使用 **`_lt`**（或等价 reference 路径），与 text **`_t`** 明确分离（**D21 L4**）：

- menu/route 在文件级 factory 上保留默认 scope：`const { _lt } = createTranslate('…', { scope: 'web/menu/menus' })`。
- `_lt` 返回可 JSON round-trip 的 `TermReference`：`{ key, module, scope, src, kind: 'literal' }`。`key` 是 canonical identity 的 vue-i18n-safe 编码（`__terms.<hex>`，hex 段为 UTF-8 长度前缀 identity，不含点）；不读取当前语言、不执行翻译。
- reference 的 `src` 是英文 msgid / 第一 fallback。显示端必须走 §7.2；只有 reference 缺失时才直接显示 legacy plain string。
- extractor 对 `_lt('literal', …)` 与 `_t('literal', …)` 一样入 pot。`_lt` 的动态 src、动态/缺失 scope 告警并跳过；Module 仍由声明文件所属模块决定。
- menu/route 使用英文 `title`/`pageTitle` 加可选 `titleText`/`pageTitleText`。这些 **wire/元数据属性名保持不变**（避免 DB/API 迁移）；其值类型为 `TermReference`。旧纯字符串输入继续兼容。

**`defineModelActions` / `entityTitle`（D19）：**

- 调用侧允许 `entityTitle: _lt('Country')`（及 `titles?: Partial<Record<op, ResourceTitle>>`），与 menu/route 同一 `ResourceTitle = string | TermReference` 契约。
- 构建期 parser 识别 `_lt('literal')`：英文 `src` → `MetaUiResource.Title`；`TermReference` → `MetaUiResource.TitleText`（新列/等价持久化）。
- 仍可由 parser 合成 `Create/Edit/Delete/Copy ` + entity msgid；`titles.*` 覆盖优先。
- **extract** 在识别 `defineModelActions({ entityTitle: _lt('…') })` 时，同步把合成后的 `Create/Edit/Delete/Copy …`（及 `titles.*` 覆盖）写入 pot，scope 与 entity `_lt` 一致（否则 `TitleText.src` 无法命中词表）。
- 运行时 `defineModelActions` 注册 action declaration（含 `title` + `titleText`），对齐 `defineAction`。
- 权限树等消费端走 §7.2：`$t(row.TitleText.key, row.TitleText.src || row.Title)` / `translateTerm`；禁止另造 action 专用 wrapper。
- List/Form 共享按钮文案（如「新建」）属 `modules/web` 组件，**不在**本条范围内。

**`@Field.string` 字段标题（与 menu 同类静态显示；细节见 `.dev/docs/core/web/field-metadata-fields-get-design.md` §5.1）：**

- **推荐：** `string: _lt('…')`；scope 建议 `<module>.model.<Model>.fields`（可用 factory 默认或 `fieldTerms(model)`）。
- 效果：extract 入 pot；持久化 / codegen 为英文 `string` + `stringText`（`TermReference`）；FE 无 FieldsGet 时走 §7.2 `translateTerm` / `$t`。
- **允许但不推荐：裸字符串** `string: 'Code'`。Extract **不**抓取裸字面量 → **不**进 pot、**无** `stringText` → 默认不翻译（仅显示 msgid / prop）。适用于测试夹具或明确不需翻译的占位字段；业务模型应写 `_lt`。
- 裸字符串 **不**抛错（除非日后另开硬错误决议）。不要依赖「另有路径保证入 pot」。
- **禁止：** `string: _t('…')`（text 会在模块加载时烤进当前进程 lang）。

**Selection 选项（与字段标题同一作者习惯；展示路径不同）：**

- **作者规则（静态与 dynamic 共用）：** `label: string | TermReference`
  - `label: _lt('…')` → 要翻译：extract 入 pot；ORM 存 msgid + `labelText`（TermReference，仅 BE/IR）；**FieldsGet 响应**用 text `_t` 译成 `label: string`。
  - `label: 'db'`（裸字符串）→ **不翻译**：透传；不入 pot；FieldsGet **不** `_t`。
- **禁止：** 显式属性 `labelText: …`（作者应写 `label: _lt(...)`）；`label: _t('…')`（装饰器加载时无稳定用户 lang，会烤进程语言）。
- **FieldsGet 响应（给 FE）：** 永远是 `{ value, label: string }`（已译或原样），**不**返回 selection 上的 TermReference / `labelText`。
- **FE：** 不对 selection 做 `translateTerm`（只消费 FieldsGet / 静态 msgid 字符串）；codegen **剥离** IR 中的 selection `labelText`，不进 web store。
- dynamic method/callable 可返回 `_lt(...)`（由 FieldsGet 再译）、请求内已 `_t(...)` 的字符串（透传）、或裸字符串（不译）。

通用 Kind 兼容保持不变：`TranslationTerm.Kind`、唯一索引、PO 的显式 `#. kind:`、terms API 与 `$choysum.i18n.t(...[, kind])` 均可承载显式 kind；缺省为 `literal`。自动 extractor 当前只产生 `literal`。

### 5.6 种子 / demo JSON（**不属术语 i18n**；D20 否决）

模块打包数据（`package.json` → `choysum.data` / `choysum.demo`）写入的是**业务记录字段值**，生命周期与用户后续改写相同，属于 **数据 i18n**，不在本文范围。

**冻结：**

| 规则 | 约定 |
|---|---|
| JSON 字段值 | 普通标量 / `{ "ref" }` 等既有 loader 形态；**禁止** `{ "_t": "…" }` |
| Extract | **不**扫描 `data/*.json` / `demo/*.json` 入 pot |
| Loader | **不**把字段值解析为术语 msgid |
| 展示 | 读 DB 字段原文；多语言展示另见数据 i18n 方案（JSONB 等） |
| 术语 `_t` | 仅用于代码 / 静态元数据（menu、route、action、label、错误文案等） |

曾短暂采用的「JSON `{ "_t" }` + scope `data/bootstrap|demo` + 前端 `displaySeedField`」已否决（**D20**），实现不得回潮。

---

## 6. 翻译表与缓存

### 6.1 TranslationTerm

契约在本文 + 共享常量；物理表 Go ensure，**非** `@Model` / MetaModel（**D13·D14**）。

| 维度 | 约定 |
|---|---|
| **表名** | `{application}_translation_term` |
| **创建** | `internal/i18n/models` + migrator 挂钩；参照 `ensureTaskJobExecutionTable` |
| **跳过 core** | `application == "core"` → no-op（**D13**） |
| **基类** | `pkg/meta.BaseModel`（`gorm:"embedded"`） |
| **TS** | 仅门面；无 repo（**D17**） |
| **MetaModel** | MVP 不注册（**D14**） |

```text
BaseModel{…}, Application, Module, Lang, Scope, Src, Value, Kind, Source, Comments?
unique(Module, Lang, Scope, Src, Kind)
```

- `Application`：冗余诊断列，**不进唯一键**。  
- `Kind` 默认 `literal`；`Source` = `packaged` \| `override`。  
- 无业务 `res_id`。

### 6.2 Cache（Go 权威；D17）

```text
lang → module → scope → kind → src → value
```

- `{app}.I18n` 与 `$choysum.i18n` **共用** `internal/i18n/store`。  
- `TermsByModules` / Gateway `messages` 只导出 `literal`；显式非 literal kind 仍可由存储、terms API 与 bridge 通用路径承载。  
- 仅 `Language.IsActive=true` 进热缓存；启停 warm/evict。  
- 写后 invalidate + bump 该 `(lang)` 的 termHash（**D2**）。  
- TS 不持权威 cache；禁止 async DB / 自 dial `{app}.I18n`。

### 6.3 导入 / 改译

| 动作 | 行为 |
|---|---|
| install/upgrade | Go + **gotext** 解析 PO → upsert `packaged`；缺省 kind=`literal`，显式 `#. kind:` 保留通用兼容；不盖 `override`（**D18**） |
| uninstall | `DELETE WHERE Module=m` → invalidate + bump |
| 在线改译 | `UpdateTerm` → `override` → invalidate + bump |
| 缺译 | bridge miss → `_t` 回退 `src` |

gotext 只做 **Po 读写**，不用包级 `Get()` 作运行时。

### 6.4 Language（语言主数据）

| 项 | 约定 |
|---|---|
| **`base.Language`** | 唯一语言主数据（术语身份 + 格式字段；见 language-merge-optimization.md） |
| **`IsActive`** | 术语启用 / 切换项；热缓存边界 |
| **格式** | 跟 Language 行字段；`RequestContext.locale` 为 UI 键（`zh-CN`），≠ 术语 `lang` |

---

## 7. 前端运行时

### 7.1 加载

- 唯一读：`GET /web/i18n/translations?lang=&hash=`（§8.2.1）。  
- `unchanged=true`：不 merge（**D4**）。  
- 缺 key / Gateway 不可用：回退 msgid。  
- 复用 i18nStore / Element / dayjs / RTL；可选 ETag=catalogHash（**D10**）。
- 动态 merge 由单一 catalog watcher 执行，并以 `locale + catalogHash` 去重；只有非空、非 unchanged、非 gatewayError 的新 catalog 在 merge 成功后才 bump Composer message revision。locale watcher 仍负责 legacy locale 合并，但不得重复 merge Gateway catalog。
- Terminology Editor 强制重载即使 locale 不变，只要 catalogHash 改变也必须 merge + bump；相同 `locale + catalogHash` 的重复响应必须忽略，避免重复 render / watcher loop。

### 7.2 FE `_t` / `_lt`、Vue computed 与术语引用显示边界

```ts
const { _t, _lt } = createTranslate(module, { scope })
_t('Save')                         // 当前调用立即返回 string
computed(() => _t('Save'))         // locale/catalog 更新后重新求值
const titleText = _lt('Settings')  // TermReference；不查词
```

`_t` 使用 canonical identity 与同一个 `translateTerm` Composer adapter。调用方用标准 `computed(() => _t(...))` 建立响应性，不使用专用 reactive helper。若 scope 必须在 computed 创建时固定，应将 scope 放进 factory defaults，或在目标 scope 中创建 factory；不要依赖重新求值时的动态 scope 栈。静态标题用 `_lt` 钉 reference，显示仍走下方 `$t` / `translateTerm`。

术语引用（`TermReference`，wire 上仍为 `titleText` / `labelText` / `pageTitleText`）消费端只允许以下两种显示形态：

1. **Vue `<template>`：直接 `$t`。**

   ```vue
   {{
     item.titleText
       ? $t(item.titleText.key, item.titleText.src || item.title)
       : item.title
   }}
   ```

2. **非 template（TS、computed、h-render、router/document title）：统一 Composer adapter。**

   ```ts
   const composer = useI18n({ useScope: 'global' })
   translateTerm(composer, item.titleText, item.title)
   ```

`translateTerm` 调用真实 vue-i18n `t(reference.key, reference.src || fallback)`，不做 `te` 前置检查。reference / Composer / 译文不可用或抛错时，按 `reference.src || fallback || ''` 安全回退。Composer active locale 与 catalog message revision 保持 computed/render 响应；`createI18n.postTranslation` 的 revision hook 必须原样返回译文（类型对齐 `PostTranslationHandler`），未来 handler 必须组合而非覆盖。Gateway 词表在 merge 前保留 legacy nested 结构，并投影到扁平 `__terms` 命名空间供 `$t(reference.key)` 查找。

动态 merge reviewer 回归必须覆盖：异步 merge 中文 catalog、notify 后只等待 `nextTick`，native template 与 `computed(() => _t(...))` 均更新；相同 `locale + hash` 忽略；同 locale 新 hash 刷新；unchanged、gatewayError、空 payload 不 notify。

### 7.3 切语言

```text
写 User.Language（匿名 localStorage）
→ setLocale（拉 Gateway）+ UI locale
→ 更新 RequestContext.lang
→ S4-MVP：location.reload；可选 remount flag；S6 定稿（D9）
```

---

## 8. 宿主 Gateway、`{app}.I18n` 与在线改译

### 8.1 原则

与 **§3.1** 一致，补充：

- 浏览器读 / 管写入口分离（translations vs terms）。  
- 「不进 ApplicationService」指**宿主聚合**；各 app **必须**暴露 `{app}.I18n` 供 dial。  
- Gateway 包：`internal/i18n/gateway`（读+写同包；**D3**）。

### 8.2 对外 HTTP

#### 8.2.1 运行时词表（可匿名）

```http
GET /web/i18n/translations?lang=zh_CN&hash=<optional>
```

```json
{
  "lang": "zh_CN",
  "locale": "zh-CN",
  "hash": "<catalogHash>",
  "unchanged": false,
  "messages": {
    "auth": {
      "web/pages/Login@title": { "Sign in": "登录" }
    }
  }
}
```

`unchanged=true`：仍返回 `lang`/`locale`/`hash`；`messages` 省略/`null`（**D4**）。

```ts
if (!res.unchanged && res.messages) {
  i18n.mergeLocaleMessage(res.locale, res.messages);
}
```

#### 8.2.2 术语目录 API（鉴权）

```http
GET  /web/i18n/terms?lang=zh_CN&application=&module=&q=&limit=&offset=
PATCH /web/i18n/terms
```

列表形状 **D7**；All-apps / 分页 **D8**。

```json
{
  "lang": "zh_CN",
  "application": "auth",
  "module": "auth",
  "scope": "web/pages/Login@title",
  "src": "Sign in",
  "value": "登录",
  "kind": "literal"
}
```

Gateway：`IdentityFromContext`；无 token → 401；无 **Terminology Editor** → 403（**D6**）。  
写路径转发用户 identity（**D1**）。不做全会话广播，仅术语重载。

### 8.3 应用服务：`{app}.I18n`

```text
{app}.I18n/GetTranslations   # 宿主 → 运行时词表
{app}.I18n/SearchTerms       # 宿主 → 目录列表
{app}.I18n/UpdateTerm        # 宿主 → 写；Source=override
```

| | |
|---|---|
| **实现** | Go `internal/i18n/service` + TermStore（**D17**） |
| **调用方** | 仅宿主 Gateway |
| **范围** | 本 app 已装模块；未知 module 忽略 |
| **GetTranslations** | `lang`, `moduleNames[]`（宿主填）, `hash?` → `termHash` / `unchanged` / `termsByModule` |
| **空 app** | 仍注册空实现（**D5**） |
| **dial 身份** | Get=内部；Search/Update=转发用户（**D1**） |
| **与 `_t`** | 共用 Go cache；TS 经 bridge，不 dial 本服务 |

#### 8.3.0 BE `_t` 与 bridge

```text
createTranslate(module, options?) → _t(src, opts?, ...args)
  → resolveRequestLang + resolveI18nScope
  → `_lt` → TermReference；`_t` → $choysum.i18n.t(...)   // sync；学 compilerFs
  → miss → src
```

#### 8.3.1 `web.I18n`

与 `auth.I18n` 同级，仅壳层词；**不**提供跨 App API。

### 8.4 读路径聚合

```text
校验 lang → 已装目录 → 按 app 分组
→ 并行 dial GetTranslations（内部身份）
→ merge messages → catalogHash（D2）→ unchanged?（D4）
```

CE 进程内 dial；EE 远端 gRPC。注册在 **`internal/server` mux**（**D3**）。

### 8.5 术语目录路径聚合

```text
鉴权（D6）→ 解析 application（D8）
→ SearchTerms | UpdateTerm（转发用户）
→ 按 D7 返回；Update 后 bump termHash → 术语重载
```

Editor 只传 `application` 字段，零 app 路由知识。

### 8.6 在线改译（Terminology Editor）

- 用户感知统一视图；后端 N 张 per-app 表。  
- 单入口（近 Language 菜单）；禁止每 App 改译菜单。  
- 只调 §8.2.2 HTTP。

```text
Settings → 术语翻译 → Terminology Editor
  搜索（默认跨 App）· 语言 · 应用（无 q 必选具体 app，D8）· 模块
  网格：Application · Module · Scope · Src · Value · Source · Status
```

```text
保存 → PATCH /terms → {app}.I18n/UpdateTerm → Source=override → 术语重载
```

同一视觉组件可跨多 application 行；**按行所属 Application 写回**。

| 阶段 | 交付 |
|---|---|
| S5-MVP | terms API + Editor；跨 App 搜；所属模块列；术语重载 |
| S5+ | 组件提示、相关词条、缺译视图、批量 |
| S6 | 与 `status` 对齐的质量徽章 |

### 8.7 权限与 ACL（冻结）

对齐 auth 权限文档；对照 documentgateway。

#### 8.7.1 双入口

| 入口 | 分类 |
|---|---|
| `GET /translations` | HTTP **Public** |
| `GET\|PATCH /terms` | HTTP **Authenticated** + Terminology Editor |
| `GetTranslations` | **仅宿主内部** |
| `SearchTerms` / `UpdateTerm` | gRPC Authenticated；ACL default-deny |

```text
匿名 ──► GET /translations ──(内部身份)──► GetTranslations
token ──► GET|PATCH /terms ──(转发用户)──► Search / Update
```

安全边界 = HTTP Gateway + 下游 ACL（防直连）。dial 身份见 **D1**。

#### 8.7.2 禁止的偷懒做法

见 **§1.3**（entryPolicy Public 化、通配 allow GetTranslations、TranslationTerm CRUD、FieldRule 当主 ACL 等）。

#### 8.7.3 ACL 挂法（运维）

| 能力 | 建议 |
|---|---|
| 术语编辑 | Service 精确 allow Search/Update；角色 **Terminology Editor**（**D6**） |
| 多 app 翻译员 | 按 Application 各挂精确条；慎用通配 |
| break-glass | 仅平台运维 Global |
| `GetTranslations` | **不对终端用户授权** |

#### 8.7.4 Company / UI / 审计

词条不按 Company 分片；dial 仍需可信 `activeCompanyId`。菜单进 `PermissionState.ui` 仅 UX；写安全以 Gateway + UpdateTerm ACL 为准。建议 audit（user、application、module、src）。

#### 8.7.5 实现对照

1. 仅 translations 允许无 identity。  
2. terms：401 / 403。  
3. 浏览器直连 `{app}.I18n` 必须失败。  
4. dial 身份单测（**D1**）。  
5. 挂 `internal/server` mux（**D3**）。  
6. 不用 entryPolicy 整组 Public。

---

## 9. 安装与 CLI

### 9.1 流水线

| 时机 | 动作 |
|---|---|
| Install/Upgrade | PO → 表 + cache；bump termHash |
| Uninstall | 删 Module 词条；invalidate；bump |
| GlobalWebBuild | **不写**术语快照 |
| Language.IsActive | warm/evict；bump |
| 在线改译 | `override`；invalidate；bump |

### 9.2 CLI（定稿）

```text
choysum i18n extract   # S1：显式 _t 字面量 → pot（含 msgctxt）
choysum i18n sync      # S1：pot → 各语 po（对齐 msgmerge）
choysum i18n status    # S6：缺译/fuzzy/orphan；可挂 CI
```

```text
开发：extract → sync → 填 po → git
安装：PO → 本 app 表 → cache
改译：Editor → Gateway PATCH /terms → UpdateTerm
浏览器读：GET /translations?lang=&hash=
```

建议 CI：`extract` 漂移与/或 `status` 非零失败（S6）。细节 → `choysum_cli.md`。

### 9.3 规范与开源库选型

决议全文见 **§13 D18**；字段对照见 **§5.4**。摘要：

| 层 | 选用 |
|---|---|
| PO/POT 格式 | GNU gettext |
| `sync` | 语义对齐 `msgmerge` |
| Go PO 解析 | `github.com/leonelquinteros/gotext`（仅 Po 读写） |
| `_t` + `_lt`（无 `output`） | BE/FE 同形；`_t` 立刻译、`_lt` 钉 TermReference；均为自研 API（§5.1·D21） |
| extract | 自研 collector（**D16**）；禁用 xgettext 族 |
| FE | vue-i18n + Gateway；不嵌 gettext 运行时 |

---

## 10. 迁移

| 现状 | 目标 |
|---|---|
| `web/web/i18n/source` + 手写 zh-CN | `modules/web/i18n/*.po` + Gateway `messages` |
| `User.Language` 闲置 | 真实偏好 |
| RequestContext 仅 locale | 补齐 `lang` |
| `SUPPORTED_LOCALES` | 仅格式/Element |
| WebHandlers 仅静态 | + Gateway 术语路由 |

试点：**web 壳层 → auth**。反模式见 **§1.3**。

---

## 11. 落地阶段（可交付 / 可验收 / 可回滚）

```text
S0 → S1 → S2 → S3 → S4 → S5 → S6
```

S4/S5 依赖 S3。细单见 **§14**。

| Stage | 交付摘要 | 关键决议 |
|---|---|---|
| **S0** | Go DDL（D13）；TermStore + sync bridge + TS `_t`/scope；`resolveRequestLang` | D13–D15、D17 |
| **S1** | parser 公共化 + collector；`extract` / `sync`（gettext 合规） | D12、D16、D18 |
| **S2** | Go PO import（gotext）+ lifecycle；无 dist 快照 | D12a/c、D17、D18 |
| **S3** | Go `{app}.I18n` GetTranslations + Gateway translations | D1–D5、D17 |
| **S4** | setLocale←Gateway；web/auth 样板；默认 reload | D9、D12d |
| **S5** | Search/Update + terms API + Editor；术语重载 | D1、D6–D8、D11 |
| **S6** | `status` + CI；remount 定稿；体验收口 | D9 |


```text
S0-1（DDL）──► S0-2（store + bridge + _t）
           ╲──► S0-3（resolveRequestLang，可并行）

S1-0a（tsgoctx）──► S1-0b（collector）──► S1-1 extract ──► S1-2 sync
```

每阶可 flag / 未接线回滚。验收总表见 §12。

---

## 12. 总检验收清单

| # | 项 | Stage | 决议 |
|---|---|---|---|
| 1 | extract/sync + install 后 BE `_t` 出译 | S1–S2 | D16–D18 |
| 2 | `{app}.I18n` + Gateway；messages；hash；无 moduleNames | S3 | D2–D5 |
| 3 | web+auth 经 Gateway 切语言 | S4 | D9 |
| 4 | RequestContext.lang；reload/remount | S4 | D9、D12d |
| 5 | 卸载清理词条 | S2 | — |
| 6 | Language.IsActive 控 cache/切换器 | S0/S4 | — |
| 7 | Scope 规则；无 distinct | S0 | — |
| 8 | Editor + terms + Source；术语重载；无 per-app 菜单 | S5 | D6–D8 |
| 9 | 无快照；无独立 i18n 存储 App；不跨库；不 dial I18n | 全程 | §1.3 |
| 10 | `web.I18n` 仅壳层；读写经 Gateway | S3/S5 | — |
| 11 | §8.7 双边界 | S3/S5 | D1、D6 |
| 12 | Go DDL；跳过 core | S0 | D13 |
| 13 | MVP 无 MetaModel | 全程 | D14 |
| 14 | extract 复用 parser；无第二套解析 | S1 | D16 |
| 15 | Go TermStore + I18n + import；sync bridge | S0–S3 | D17 |
| 16 | gettext + gotext 读写；非 xgettext；无 gotext 运行时 | S1–S2 | D18 |

---

## 13. 已决议项（原开放问题；2026-07-16 定稿，D13–D18）

下列项已拍板；实现与评审以本节为准。若与前文示例冲突，**以本节为准**并在实现 PR 中回写前文。

### D1 — Gateway dial 身份（S3 前）

**决议：读内部 / 写转发（原选项 C）。**

| 调用 | dial 身份 |
|---|---|
| `GetTranslations` | 宿主 **内部身份**（不依赖浏览器 token；对齐 Public 读） |
| `SearchTerms` / `UpdateTerm` | **转发用户 identity**（下游 ACL default-deny 真实生效；可审计） |

须单测：匿名可读 translations；terms 无 token → 401；有 token 无角色 → 403。

### D2 — `termHash` / `catalogHash`（S3 前）

**决议：按 lang 计算；SHA-256 hex 截断 16。**

1. **termHash(app, lang)**：对该 app 该 `lang` 全部词条，按 `(module, scope, src, kind, value, source)` **字典序** canonical 序列化（字段分隔与编码实现固定并单测）→ SHA-256 → hex **截断前 16 字符**。  
2. **catalogHash(lang)**：对各已装 app 的 `termHash`，按 app 名排序后拼接 `app:termHash`（换行或固定分隔）→ 同上 SHA-256 hex16。  
3. HTTP 查询参数 `hash` / 可选 `ETag` = **catalogHash**；app 侧 `GetTranslations.hash` = 该 app **termHash**。  
4. 改译 / install / uninstall / 语言启停：只 bump 受影响 `(app, lang)` 的 termHash，进而影响 catalogHash。

### D3 — Gateway 注册点（S3 前）

**决议：** 注册在 **`internal/server` mux**（类 `documentgateway`），包路径 `internal/i18n/gateway`。**不**塞进 web `WebHandlers`（静态 dist 与术语旁路分离）。

### D4 — `unchanged` 响应（S3 前）

**决议：**

- `unchanged=true`：HTTP **200**；仍返回 `lang` / `locale` / `hash`；**`messages` 省略或 `null`**（FE 不 merge）。  
- MVP **不做** HTTP 304；`ETag` 与 catalogHash 对齐为可选增强（见 D10）。

### D5 — 无 `i18n/` 的 Application（S3）

**决议：** 仍注册空 `{app}.I18n`；`GetTranslations` / `SearchTerms` 返回空集合 + 稳定 termHash；Gateway **不跳过** dial。

> **将被废止（injectappmodel 改造）：** 取消「空 app / 无 service 宿主也强制 I18n」。无 `entryPoints.service` 且未 `EnsureServiceEntry` 则不提供术语模型/RPC；TranslationTerm 可对宿主模块虚拟补 service（不改 `package.json`）。见改造设计 §2.2 / §3。

### D6 — 术语编辑角色与种子 ACL（S5 前）

**决议：**

- 角色名：**`Terminology Editor`**（菜单「术语翻译」对齐）。  
- 种子 ACL：**Service 精确 allow** — 各已装业务 app 的 `{app}.I18n/SearchTerms` 与 `{app}.I18n/UpdateTerm`。  
- 平台运维另有 Global break-glass；**不对**终端用户 allow `GetTranslations`。

> **将被改写：** 删 I18n method allows；改绑 `TranslationTerm` Search/Read/Update（路径 α / 改造 PR-P4）。

### D7 — `GET /terms` / `SearchTerms` / `PATCH` 契约（S5 前）

**决议：MVP 用 `limit`/`offset`；响应与 PATCH 如下。**

`GET /web/i18n/terms` / `SearchTerms` 响应：

```json
{
  "lang": "zh_CN",
  "items": [
    {
      "application": "auth",
      "module": "auth",
      "scope": "web/pages/Login@title",
      "src": "Sign in",
      "value": "登录",
      "kind": "literal",
      "source": "packaged",
      "status": "translated"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

- `status`：`translated` | `missing` | `fuzzy`（缺译 ≈ `value` 空 → `missing`）。  
- **不做** cursor（S5+ 若 All-apps 分页再议）。

`PATCH /web/i18n/terms`：单对象（§8.2.2）或 `{ "items": [ … ] }`。  
- 按 `application` 分组 dial；**同一请求全成功或全失败**（任一组失败 → 整批 4xx + 错误明细；已成功组须可回滚或实现为先校验再写）。

### D8 — All-apps 列表分页（S5）

**决议：S5-MVP**

- 无搜索时：**`application` 必填**（禁止无 `q` 的 All 分页）。  
- 「All」仅当 **`q` 非空** 时允许 fan-out；**limit 封顶（如 100）且不分页**（可返回 `truncated: true`）。  
- 正式跨 app 合并分页 → **S5+**。

### D9 — 语言切换 remount（S4）

**决议：S4-MVP 默认 `location.reload`。**  
可选 feature flag 试 remount（清路由视图 + 表单/元数据 store）；**S6** 再定稿 remount 边界（原 §13.1）。

### D10 — Gateway GET 缓存（可延后）

**决议：MVP 仅客户端 `hash` 协商**；服务端短 TTL **不做**。  
可选：响应 `ETag` = catalogHash（S3 末可加）；与 D4 的 200+unchanged 并存。

### D11 — 组件提示（S5+）

**决议：S5 仅 Scope path 启发式**（不写表字段）。更丰富的组件提示须另行设计稳定、可序列化的契约，不从静态元数据自动推断。

### D12 — S1/S2 边界策略

| ID | 项 | 决议 |
|---|---|---|
| D12a | sync 后 pot 消失的 msgid | PO 标 **obsolete**；install **不删** DB 行（避免误删 `override`）；S6 `status` 报 orphan |
| D12b | 非字面量 `_t(x)` | extract **跳过 + warn**；不入 pot |
| D12c | PO 条目无 msgctxt | 导入 **拒绝该条目并记日志**（不造假 Scope）；extract 必须产出 msgctxt |
| D12d | RequestContext 仅有 `locale` | **不**把 `zh-CN` 当术语 `lang`；缺 `lang` → `User.Language` → **`en_US`** |
| D12e | Vue `<template>` extract（S1） | S1-MVP 用**正则**抽字面量 `_t`；默认 Scope=`path@template`；**S1+** 改完整 Vue template AST |

### D13 — 翻译表物理 DDL（S0 前）

**决议：Go lifecycle ensure；per-application 表名；跳过 `core`。**

1. **建表路径**：`internal/i18n/models/translation_term.go` + `ensureTranslationTermTable`；由 `internal/module/evolution/schema/migrator.go` 在 `migrator.Migrate()` 中、模型 schema 迁移完成之后调用（`evolution/schema` import `internal/i18n/models` 进行 ensure；skip `application == "core"`）。  
2. **表名**：`{application}_translation_term`（如 `auth_translation_term`）。  
3. **跳过 core**：当 manifest **`application == "core"`** 时 **no-op**，**不**创建 `core_translation_term`；判定以 `application` 字段为准，**非** module 名。  
4. **字段基类**：Go struct 直接复用 `pkg/meta.BaseModel`（`gorm:"embedded"`）；已有 `internal/module/metadata/*` 先例；**不**在 `internal/i18n/models/` 重写一份 BaseModel。  
5. **附加列**：增加 `Application` 列，记录所属 app 名；作为冗余诊断列，**不**进入唯一键。  
6. **禁止**：core `@Model('TranslationTerm')` 作为建表或迁移路径。  
7. **验收**：安装/迁移 **auth**（等业务 app）→ 对应表存在；安装/迁移 **core** → 无 `core_translation_term`。

### D14 — MetaModel 注册（MVP）

**决议：MVP 不注册 TranslationTerm 为 MetaModel。**

1. **不**进入 meta 模型目录；**不**挂公共 Model CRUD / 菜单。  
2. 改译入口为宿主 Gateway **terms API** + **Terminology Editor**（§8），非 MetaModel 权限面。  
3. 读写权威在 **Go TermStore**；TS 仅经 `$choysum.i18n` sync bridge 查词（§13 D17）；Kind/Source 保留通用存储与协议契约，缺省 Kind=`literal`。  
4. **S6+** 若需 admin introspection 可再议；**本路线不预留** MVP 任务。

> **将被改写（injectappmodel 改造）：** 允许 per-app `@Model('TranslationTerm')`（injectappmodel 注入；非 `core`）；删 `/web/i18n/terms`；Editor 路径 α = 标准模型 CRUD；业务 RPC 权威迁 TS；`_t` 查词缓存仍在 Go。见改造设计 §2 / §5 / §6.2。

### D15 — S0 落地顺序（S0）

**决议：S0-1 必须先合并；S0-2 依赖 S0-1；S0-3 可与 S0-2 并行。**

| PR | 顺序 | 依赖 | 说明 |
|---|---|---|---|
| **S0-1** | **1** | — | Go DDL + ensure + 共享类型；**须先合入** |
| **S0-2** | **2** | S0-1 | Go TermStore/cache + `$choysum.i18n` + TS `_t`/scope（§13 D17） |
| **S0-3** | **2 或并行** | 无硬依赖 S0-1 | `resolveRequestLang`；可与 S0-2 **同 Sprint 并行开 PR** |

补充：

- S0-2 内 **Scope 算法单测**可提前开发（mock），但 **PR 合并顺序**仍为 S0-1 → S0-2。  
- S0 验收拆分：S0-1 单独可验（DDL）；S0-2 + S0-3 完成后满足 §11 S0 全量验收。

### D16 — extract 解析栈（S1 前）

**决议：复用现有 parser 封装层；S1 前完成 parser 公共化 + collector 骨架；Vue `<script>` 走 AST，`<template>` S1-MVP 先用正则。**

1. **底层**：`github.com/buke/typescript-go-internal`（经 `internal/parser` / `vueparser`）。  
2. **walk 先例**：`vueparser/uiresource_parser.go` 的 `CallExpression` 全树遍历；**不**使用 `backendtsparser.Parse()` 作为 extract 入口。  
3. **S1 前置**：**PR-S1-0a**（`internal/parser/tsgoctx` 公共化）→ **PR-S1-0b**（`internal/i18n/extract` collector 骨架；依赖 S0-2 Scope）。  
4. **S1-MVP 范围**：  
   - `.ts` / `.tsx` / `.vue` **`<script>`**：AST walk（collector）。  
   - `.vue` **`<template>`**：**正则**提取字面量 `_t`（如 `{{ _t('...') }}`、属性绑定中的字面量调用）；非字面量 **skip + warn**（§13 D12b）。  
   - template 默认 Scope：`path@template`（与 script 的 auto scope 区分；无手动 `withI18nScope` 覆盖）。  
5. **后续（S1+）**：以**完整 Vue template AST** 替换 template 正则路径（精度、指令、复杂表达式）；regex 实现须集中、可替换，勿散落多处。  
6. **禁止**：`extract` 旁路实现第二套 TS/**script** 解析语义（template 正则不算第二套 TS parser）。

### D17 — Go TermStore / `{app}.I18n` / sync bridge（S0–S3 前）

**决议：表 + cache + `{app}.I18n` + PO import 均在 Go；TS 仅 `_t`/scope/lang 门面；经 sync `$choysum.i18n` bridge 读 Go cache。**

> **将被改写（injectappmodel 改造）：** 删除 Go `{app}.I18n` 业务 handlers 与 TermStore **服务 API**；保留 Go **查词 Lookup cache** + bridge；`GetTranslations` / ORM CRUD 在 TS `TranslationTerm`；PO packaged 写 → **共享 helper**（install 不 dial 模型 gRPC）；Gateway translations dial 模型方法。见改造设计 §2.1 / §2.3 / §7 / §12 PR-P5*。

合成项：

| 项 | 决议 |
|---|---|
| **A** | `{app}.I18n`（GetTranslations / SearchTerms / UpdateTerm）**Go 原生实现**；ApplicationService **不再** handler→JS |
| **B** | 术语 **cache 权威在 Go**；TS `_t` 经 **sync** `$choysum.i18n.t(module,lang,scope,src[,kind])` 查询；miss → TS 回退 `src` |
| **C** | PO import / uninstall 清理 **迁 Go**（lifecycle 调 `internal/i18n/import`）；与 UpdateTerm 共用 TermStore 写出口 |

包结构（建议）：

```text
internal/i18n/
  models/          # TranslationTerm + EnsureTranslationTermTable
  store/           # GORM + in-process cache + termHash
  service/         # {app}.I18n handlers
  import/          # PO → upsert packaged
  bridge/          # WithTerminology → $choysum.i18n.*

modules/core/service/i18n/
  translate.ts / scope.ts / request_lang.ts   # 门面 only
  # 无 app_i18n_service.ts / translation_term_repo.ts / TS 权威 cache
```

约束：

1. **`_t` 必须同步**：bridge 用 sync host function（学 `compilerFs`）；**禁止** async `$choysum.db.query` 作热路径。  
2. **`{app}.I18n` 与 bridge 共用同一 Go cache**；写后只刷 Go，无需 Go→TS notify。  
3. **termHash 权威在 Go**（与 GetTranslations 一致）。  
4. **对外 dial 拓扑不变**：Gateway 仍 dial `{app}.I18n`；浏览器不直连；`_t` 不 dial 本服务。  
5. **删除旧规划**：TS 薄 repo、TS `app_i18n_service`、TS PO import 作为权威写路径。

验收：

1. Go UpdateTerm / PO import 后，同进程 `_t` 经 bridge 立即可见新译文（同 cache）。  
2. bridge miss → `_t` 返回 src。  
3. `{app}.I18n` 冒烟测不经过 QuickJS handler。

### D18 — gettext 契约、`_t` / `_lt` 与库选型（S1 / S2 前；2026-07-20 修订）

**决议：格式跟 GNU gettext；BE/FE 门面同形返回 `_t` + `_lt`（详见 D21）；库只用于 PO 读写。已硬切删除 `output`。**

| 项 | 决议 |
|---|---|
| **格式** | pot/po 以 GNU gettext 手册为准；`msgctxt` = Scope（§5.4） |
| **`sync`** | 语义对齐 `msgmerge`；对外只 `choysum i18n sync` |
| **Go PO 解析** | `github.com/leonelquinteros/gotext`；仅用于 PO 读写 |
| **`_t`（text）** | 立即返回 string；后端走同步 bridge，前端走 Composer；Vue 用 `computed(() => _t(...))` |
| **`_lt`（reference）** | 返回 `TermReference`；不查词、禁止插值；静态 metadata 使用稳定 literal scope |
| **`output`** | **已废**；禁止 call/factory `output` 选项与双读 |
| **lazy** | 进程内需求使用显式 `() => _t(...)` 闭包；`_lt` **不是** Odoo LazyGettext |
| **extract** | 识别 `_t` / `_lt` 字面量（及 `_lt` 别名）与 factory 默认 scope；禁止 xgettext 族作为权威；不识别 `output` / `_td` / `_tr` / `mode` |
| **plural** | MVP 不做；不实现 `ngettext` |

验收：`_t`/`_lt` 类型推断精确；`TermReference` 可稳定 JSON round-trip；主干无 `output: 'text'|'reference'` 生产调用；PO 可由标准 gettext 工具读入且包含 msgctxt。

### D19 — `defineModelActions` / `entityTitle` 术语化

**决议：与 menu/route 对齐——调用侧 `_lt`，构建期写入 `Title` + `TitleText`，权限树走 §7.2。**

| 项 | 决议 |
|---|---|
| **作者 API** | `entityTitle?: ResourceTitle`；`titles?: Partial<Record<'create'|'edit'|'delete'|'copy', ResourceTitle>>` |
| **Parser** | 允许字面量字符串或 `_lt('literal'[, { scope }])`；禁止动态表达式（与 menu title 同级严格） |
| **合成** | 无 `titles[op]` 时：`Create/Edit/Delete/Copy ` + entity `src`；覆盖优先 |
| **持久化** | `MetaUiResource.Title` = 英文 fallback；`MetaUiResource.TitleText` = `TermReference` JSON（可空；旧数据仅 Title） |
| **运行时 TS** | `defineModelActions` 注册各 op 的 action declaration（`title` + `titleText`），对齐 `defineAction` |
| **显示** | 权限树等：`$t` / `translateTerm`；禁止 action 专用 display wrapper |
| **范围外** | `OListView`/`OFormView` 硬编码按钮文案；Selection label API |

验收：base 视图 `entityTitle: _lt('Country')` 可构建；`Edit Country` 等进入 pot；zh_CN 下权限树显示译文。

### D20 — 种子 / demo JSON `{ "_t" }`（**否决**）

**决议：不做。** 种子/demo JSON 字段属 **数据 i18n**，不得借用术语 `_t` / PO / `TranslationTerm`。

| 项 | 决议 |
|---|---|
| JSON `{ "_t" }` | ❌ 禁止 |
| extract 扫 data/demo JSON | ❌ 禁止 |
| loader 解析为 msgid | ❌ 禁止 |
| FE `displaySeedField` / `seed-term-module` | ❌ 禁止 |
| 正确归属 | 数据 i18n（字段 JSONB / 等价方案，另文） |

理由：业务记录值与代码术语的写入模式、生命周期、查词维度不同；把 seed `Name` 当 msgid 会把术语表拖回「按业务行翻译」的旧路。


### D21 — `_t` / `_lt` 双 helper 与硬切（2026-07-20）

**决议：`createTranslate` 同时返回 `_t` 与 `_lt`；一刀切删除 `output`；已在 `feat/i18n-lazy-translate` 落地（PR #176）。**

| ID | 决议 |
| -- | ---- |
| **L1** | BE core + FE web 门面同形返回 `{ _t, _lt }` |
| **L2** | `_t`：仅 text → `string`；允许插值；查词 |
| **L3** | `_lt`：仅 reference → `TermReference`；禁止插值；不查词（≠ Odoo LazyGettext） |
| **L4** | 静态元数据（menu / route / model action / `@Field.string`）必须用 `_lt`；禁止 text `_t` 写入静态 wire |
| **L5** | factory 默认 `scope` / `path` / `location` / `kind` 同时作用于 `_t` 与 `_lt` |
| **L6** | 删除 call/factory `output` 与相关 overload；无迁移双读、无兼容窗口 |
| **L7** | extract / parser 识别 `_lt('…')` 及 `_lt` 别名；不再认 `output: 'reference'` |
| **L8** | 硬切换：无 `@deprecated` 双 API；同一 PR 内改完导入、调用、extract、parser、测试 |
| **L9** | 可选 `fieldTerms(model)` → 预置 fields scope 的 `_lt`（app 级即可） |

开放（不阻塞）：`fieldTerms` 是否上提 core；`stringText` 是否改名（另议）。


---

## 14. 开发任务清单（按 PR / 文件 / 函数 / 测试粒度）

体例：约束重构文档的 **PR → 建议标题 → 文件/函数任务 → 测试 → DoD**；文档架构重构的 **文件表 + 分阶段退出条件**。  
路径以「建议新建 / 改造位点」标注；实现时可微调命名，但不得偏离 §1.1 冻结结论与 §10.2 不做项。

> **改造批注（草稿，P6 回写）：** [injectappmodel 设计 §12](./terminology-i18n-injectappmodel-design.md) **废止/改写** 下列依赖 Go `{app}.I18n`、`/web/i18n/terms`、EnsureI18nMeta / EnsureTranslationTermTable、以及「禁 TranslationTerm MetaModel」的任务，尤其 **PR-S3-1**（合成 I18n）、**PR-S5-1～S5-3**（terms Editor）。CLI extract/sync/status、FE translations 加载、`_t` bridge 等仍以本节为准，除非改造文明示覆盖。Gateway catalog memo 仍不做（与改造 §2.15 一致；主设计 D10 ETag 已满足）。

### 14.0 总览

**S0 合并顺序（§13 D15）：** `S0-1` → `S0-2`；`S0-3` 可与 `S0-2` 并行。  
**S1 前置（§13 D16）：** `S1-0a` → `S1-0b`（依赖 S0-2）→ `S1-1`。

| PR | Stage | 建议标题（conventional） | 依赖 |
|---|---|---|---|
| PR-S0-1 | S0 | `feat(schema): ensure per-application translation_term table on migrate` | —（**须先合入**） |
| PR-S0-2 | S0 | `feat(i18n): Go TermStore cache + sync $choysum.i18n + TS _t facade` | **S0-1**（表） |
| PR-S0-3 | S0 | `feat(core): add RequestContext.lang resolution helpers` | 无硬依赖；**可与 S0-2 并行** |
| PR-S1-0a | S1 | `refactor(parser): extract shared tsgo parse context` | —（可与 S0 并行；**须在 S1-0b / S1-1 前**） |
| PR-S1-0b | S1 | `feat(i18n): add term collector skeleton for extract` | **S1-0a**, **S0-2** |
| PR-S1-1 | S1 | `feat(cli): add choysum i18n extract` | **S1-0b**, S0-2 |
| PR-S1-2 | S1 | `feat(cli): add choysum i18n sync` | S1-1 |
| PR-S2-1 | S2 | `feat(i18n): import module PO into Go TermStore` | S0-1/S0-2（表+cache） |
| PR-S2-2 | S2 | `feat(lifecycle): wire PO import on install/upgrade/uninstall` | S2-1 |
| PR-S3-1 | S3 | `feat(i18n): add Go {app}.I18n GetTranslations` | S0-2 |
| PR-S3-2 | S3 | `feat(i18n-gateway): aggregate GET /web/i18n/translations` | S3-1 |
| PR-S4-1 | S4 | `feat(web): load terminology via Gateway and setLocale` | S3-2 |
| PR-S4-2 | S4 | `refactor(web): migrate shell copy to PO + web.I18n` | S4-1, S1-*, S3-1 |
| PR-S4-3 | S4 | `feat(auth): pilot terminology i18n and User.Language` | S4-1, S2-2 |
| PR-S4-4 | S4 | `feat(web): locale-remount (or full reload) on language switch` | S4-1 |
| PR-S5-1 | S5 | `feat(i18n): Go {app}.I18n SearchTerms/UpdateTerm + Source` | S3-*, S2-1 |
| PR-S5-2 | S5 | `feat(i18n-gateway): terms GET/PATCH for Terminology Editor` | S5-1, S3-2 |
| PR-S5-3 | S5 | `feat(web): Terminology Editor via Gateway and terminology reload` | S5-2, S4-1 |
| PR-S6-1 | S6 | `feat(cli): add choysum i18n status and CI gates` | S1-* |
| PR-S6-2 | S6 | `chore(i18n): polish locale-remount bounds and optional PO download` | S4-4, S5-* |

**本路线明确不产生任务的内容：** `dist/web/i18n/**` 快照；`choysum i18n import/export/merge/update/missing`；Translation 微服务 / 独立 i18n Application（作唯一存储层）；改译全会话广播；数据 i18n；Gateway 跨库；浏览器读路径 `moduleNames`；浏览器直连 `{app}.I18n`；每 App TranslationTerm 公共菜单；`web.I18n` 全站聚合服务；entryPolicy Public 化 `*.I18n/*`；TranslationTerm 公共 Model CRUD 作改译入口；FieldRule/RecordRule 当主写 ACL；core `@Model('TranslationTerm')` / `core_translation_term`；MVP MetaModel 注册；TS `app_i18n_service` / TS 权威 cache / TS PO import 作唯一权威写路径；以 `xgettext`/`xgotext`/`gettext-extractor` 作 extract 权威；gotext/`node-gettext` 运行时替代 TermStore；MVP plural API。

---

### 14.1 建议路径地图（文件级）

| 区域 | 建议路径 | 职责 |
|---|---|---|
| 存储 DDL | `internal/i18n/models/translation_term.go`（新） | GORM struct（嵌入 `meta.BaseModel`）+ `ensureTranslationTermTable`；per-app 表名；跳过 core |
| 迁移挂钩 | `internal/module/evolution/schema/migrator.go` | `migrator.Migrate()` 内模型迁移之后调用 `internal/i18n/models.ensureTranslationTermTable` |
| TermStore | `internal/i18n/store/`（新） | GORM 读写 + **Go 权威 cache** + termHash（§13 D17） |
| `{app}.I18n` | `internal/i18n/service/`（新） | Go GetTranslations / SearchTerms / UpdateTerm |
| PO import | `internal/i18n/import/`（新） | PO → upsert packaged；**gotext** 解析（§13 D18） |
| i18n bridge | `internal/i18n/bridge/` + `defaultengine` plugin（新） | sync `$choysum.i18n.t(...)` |
| TS 门面 | `modules/core/service/i18n/`（新） | 后端 `createTranslate`、`_t`、scope、`resolveRequestLang`；**无**权威 cache / repo / app service |
| Scope | `modules/core/service/i18n/scope.ts`（新） | extract 与 runtime 共用算法 |
| 应用服务注册 | `internal/service/` ApplicationService | 挂载 **Go** `{app}.I18n` handler（非 JS） |
| RPC 上下文 | `modules/core/rpc/context.ts` | `lang` 键约定与解析辅助 |
| 生命周期 | `internal/module/lifecycle/`（扩） | install/upgrade/uninstall 调 **Go** PO import / DeleteModuleTerms |
| CLI | `cmd/cmd_i18n.go`（新）等 | `extract` / `sync` / `status` |
| TS 解析公共层 | `internal/parser/tsgoctx/`（新；S1-0a） | 从 `vueparser` 抽出的 tsgo 上下文；供 extract / vueparser 共用 |
| extract 实现 | `internal/i18n/extract/`（新；S1-0b 起） | collector（AST）+ `template_regex`（S1-MVP）+ Scope(Go) + pot |
| path alias | `internal/vueplugin/`（既有） | `ParseTsconfigPathAlias`；extract CLI 接线 |
| Gateway | `internal/i18n/gateway/`（新） | `GET /translations`；`GET|PATCH /terms`；dial `{app}.I18n` |
| 注册 | `internal/server/server_base_routes.go` | 挂 Gateway |
| FE 壳 | `modules/web/web/stores/i18nStore/*`、`modules/web/web/app.ts` | Gateway 拉词、setLocale、RequestContext；前端 `createTranslate` 返回 `{ _t, _lt }` |
| Terminology Editor | `modules/web/web/pages/.../TerminologyEditor.vue`（新） | 只调 Gateway terms API |
| web service（薄） | `modules/web/service/`（新，若需） | 注册 `web.I18n`（**不**装载 TranslationTerm `@Model`） |
| FE 模块 i18n | `modules/web/i18n/`、`modules/auth/i18n/`（新） | pot/po |
| Language | `modules/base/service/models/language.ts` | 已有 `IsActive`；钩子 warm/evict |
| User | `modules/auth/service/models/user.ts` | `User.Language` 真实偏好 |

---

### 14.2 PR-S0-1：TranslationTerm 表结构（Go lifecycle）

目标：在 **业务 Application** install/upgrade 时 ensure `{app}_translation_term`；唯一键含 Scope；**跳过 `core`**；MVP **不**注册 MetaModel / **无** `@Model`（§13 **D13·D14**）。

建议标题：`feat(schema): ensure per-application translation_term table on migrate`

改造清单（函数级）：

1. 文件：`internal/i18n/models/translation_term.go`（新）
2. 结构：GORM struct；嵌入 `meta.BaseModel`；字段 `Application` / `Module` / `Lang` / `Scope` / `Src` / `Value` / `Kind` / `Source` / `Comments?`
3. 函数：`translationTermTableName(application string) string` → `{app}_translation_term`
4. 函数：`ensureTranslationTermTable(runtimeScope, application string) error`；`application == "core"` 时 **no-op**（§13 D13；以 manifest `application` 判定）
5. 任务：`unique(Module, Lang, Scope, Src, Kind)`；默认 `Kind=literal`，`Source=packaged`；`Application` 写当前 app 名但**不**进入唯一键
6. 文件：`internal/module/evolution/schema/migrator.go`
7. 任务：在 `migrator.Migrate()` 的模型 schema 迁移之后调用 ensure（传入当前 module 所属 application；`evolution/schema` import `internal/i18n/models`）
8. 任务：Kind/Source 常量可放 Go 包常量（或极薄 TS 镜像，仅门面用）；**无** `@Model` / TS repo
9. 任务：直接复用 `pkg/meta.BaseModel`（`gorm:"embedded"`）；不要在 `internal/i18n/models/` 重写同构 BaseModel
10. 任务：**无** `res_id`；**不**注册公共菜单 / 通用改译 UI / MetaModel

测试清单：

1. 文件：`internal/i18n/models/translation_term_test.go`（新）
2. 用例：安装 auth 模块后存在 `auth_translation_term`；安装/迁移 core **不**创建 `core_translation_term`
3. 用例：`Application` 列写入当前 app 名，但同 app 分表下不进入唯一键
4. 用例：同 `(Module, Lang, Scope, Src, Kind)` 唯一约束生效
5. 用例：同 Src、不同 Scope 可并存
6. 用例：无 TranslationTerm MetaModel 元数据（MVP）

完成定义（DoD）：

1. 表结构与 §6.1 一致；合部署下多 app 表名可并存。
2. 不接线 FE / CLI / Gateway / TS repo 时无行为变化（仅 DDL ensure）。

退出条件：§6.1 契约与物理表对齐；可独立合并且功能 flag 关闭时无行为变化。

---

### 14.3 PR-S0-2：Go TermStore cache + sync bridge + TS `_t` 门面

目标：Go 权威 cache；TS `_t` 经 **sync** `$choysum.i18n` 查询；缺译回退 src；Scope 算法可单测。**须在 S0-1 合并后合入**（§13 D15·**D17**）。

建议标题：`feat(i18n): Go TermStore cache + sync $choysum.i18n + TS _t facade`

改造清单（函数级）：

1. 文件：`internal/i18n/store/`（新）
2. 函数：`WarmLanguage` / `EvictLanguage` / `Lookup` / `InvalidateModule` / `BumpTermHash`
3. 任务：仅加载 `Language.IsActive=true`；结构 `lang → module → scope → kind → src → value`；读 `{app}_translation_term`
4. 文件：`internal/i18n/bridge/` + `internal/defaultengine` RuntimePlugin
5. 函数：sync `$choysum.i18n.t(module, lang, scope, src[, kind])`（学 `compilerFs`；**禁止** async）
6. 文件：`modules/core/service/i18n/scope.ts`（新）
7. 函数：`resolveI18nScope` / `formatScope`；§5.3 优先级；禁止行号与 `_t.distinct`
8. 文件：`modules/core/service/i18n/translate.ts`（新）
9. 函数：`createTranslate` → `{ _t, _lt }`；`withI18nScope`；插值（仅 `_t`）；调用 bridge；miss → `src`；进程内延迟用显式闭包（§13 D18·D21）
10. 任务：**不**实现 TS 权威 cache / `translation_term_repo.ts` / `app_i18n_service.ts`；**不**引入 gotext/`node-gettext` 运行时；**不**保留 `output` / `_td` / `_tr` / `LazyTranslate`
11. 文件：导出点（`modules/core/service/i18n/index.ts` 等）

测试清单：

1. 文件：`internal/i18n/store/*_test.go`、`internal/i18n/bridge/*_test.go`
2. 用例：warm 后 Lookup 命中；`IsActive=false` 不进热缓存；invalidate 后 bump termHash
3. 用例：sync bridge 返回译文；miss 空 → TS `_t` 回退 src
4. 文件：`modules/core/service/i18n/scope.test.ts`、`translate.test.ts`
5. 用例：手动 scope / withI18nScope；插值；不上 `_t.distinct`
6. 用例：`_lt` 返回可 JSON round-trip 的 `TermReference` 且不查词；`_lt` 拒绝插值；无 `output` 选项（§13 D18·D21）

完成定义（DoD）：

1. Go store + bridge + TS `_t` 单测全绿。
2. 无 FE 接线、无跨服务查词；不依赖 MetaModel；符合 §13 D17。

退出条件：§11 S0 中 S0-2 验收满足；未接线 install 时可用 fixture 灌表 + warm。

---

### 14.4 PR-S0-3：RequestContext.lang

目标：统一 `lang`（术语）与 `locale`（格式）键；解析链文档化并有 helper。**可与 S0-2 并行**（§13 D15）；无硬依赖 S0-1 表。

建议标题：`feat(core): add RequestContext.lang resolution helpers`

改造清单（函数级）：

1. 文件：`modules/core/rpc/context.ts`
2. 任务：文档化 KV：`lang` / `locale`（§5.2）；示例改为同时携带
3. 函数：新增 `resolveRequestLang(kv, fallbacks?)`（建议）
4. 任务：解析序：baggage `lang` → `User.Language` → Company 默认 → `en_US`（Company 接入可后移 S4；**禁止**把 `locale` 当 `lang`，§13 D12d）
5. 文件：`modules/core/service/i18n/translate.ts`
6. 任务：`_t` 读当前 RequestContext.lang（与 cache 键一致，内部码 `zh_CN`）

测试清单：

1. 文件：`modules/core/rpc/context.test.ts`（扩）或 `i18n/request_lang.test.ts`
2. 用例：仅有 `locale=zh-CN` 时不误当术语 lang，回退 `en_US`（或 User.Language）（§13 D12d）
3. 用例：baggage lang 优先于默认

完成定义（DoD）：

1. typecheck 无新增错误；辅助函数有单测。
2. 未设 `lang` 时回退链与 §13 D12d 一致（最终 `en_US`）。

---

### 14.5 extract 解析能力评估（S1 前置结论）

**结论：现有 `internal/parser` / `vueparser` 足以作为 S1 extract 的基础，但不能直接复用 `backendtsparser.Parse()`；须在 S1 前完成 parser 公共化，并新增 `internal/i18n/extract` collector。**

| 能力 | 现状 | 对 `_t` extract 的含义 |
|---|---|---|
| TS AST 解析 | ✅ `internal/parser/ast.go`（`ParseTSTree`）+ `vueparser/tsgo_parse_ctx.go`；底层 `typescript-go-internal` | 可直接解析 `.ts` / `.tsx` / Vue `<script>` |
| 全树遍历 + `CallExpression` | ✅ 先例：`vueparser/uiresource_parser.go`（`ForEachChild` + `parseUiResourceCall`） | extract 应复用同一 walk 模式，而非 class/model parser |
| 字符串 / object literal | ✅ `parseStringLiteral`、`tsgoIsLiteralNode`、object property 提取 | 支持 `_t('msgid')`、`_t('msgid', { scope: 'x' })` |
| import / path alias | ✅ `tsParseCtx.imports`、`resolveModuleSpec`、`parser.ApplyPathAlias`；CLI 侧接 `internal/vueplugin.ParseTsconfigPathAlias` | 支持别名 import 与模块路径归一 |
| Vue SFC script | ✅ `vuefileparser` + `vuesfchtmlparser` 拆 `<script>` / `<script setup>` | script 内 `_t` 走 AST collector |
| `backendtsparser` 模型解析 | ⚠️ 仅扫 class / field / decorator / method 签名 | **不能**用于 extract；不遍历 method body 普通调用 |
| Scope 推导 | ❌ 未实现 | 需新建：`withI18nScope` 栈、enclosing symbol、file path 组合（对齐 `scope.ts`） |
| `createTranslate` 绑定 | ✅ 已实现 | `const { _t, _lt } = createTranslate('mod', options?)`；`_lt` 别名；无 `output`（§5.1·D21） |
| Vue `<template>` | ⚠️ 无 template AST | **S1-MVP**：`vuesfchtmlparser` 拆 template 文本 + **正则**抽字面量；默认 Scope=`path@template`；**S1+**：完整 Vue template AST 替换 regex |

**明确不做 / 不推荐：**

- ❌ 在 `extract` 内旁路重造第二套 TS/**script** 解析链（与 runtime / 现有 build parser 漂移）。
- ❌ 直接调用 `backendtsparser.Parse()` 期望得到 `_t` 调用点。
- ❌ S1-MVP 为 template 引入完整 Vue template AST（过重；**先用 regex，S1+ 再换 AST**）。

**推荐落点：**

```text
internal/i18n/extract/          # collector(AST) + template_regex + scope(Go) + pot
  ↑ 复用
internal/parser/tsgoctx/        # S1-0a：从 vueparser 抽出的公共 TS 解析上下文
internal/parser/vueparser/      # SFC 拆分、walk 先例（uiresource_parser）
internal/vueplugin/             # tsconfig path alias
```

---

### 14.6 PR-S1-0：parser 重构与 extract collector 骨架（S1 前置）

目标：在 `choysum i18n extract` 之前，完成 parser 公共化与 term collector 骨架；避免 `internal/i18n/extract` 复制 `vueparser` 私有 `tsParseCtx` 逻辑。

建议拆为两个可独立合并的子 PR（§14.0 **PR-S1-0a / PR-S1-0b**）。

#### PR-S1-0a：公共 TS 解析上下文

建议标题：`refactor(parser): extract shared tsgo parse context`

改造清单（函数级）：

1. 文件：`internal/parser/tsgoctx/`（新）或等价公共包
2. 任务：从 `internal/parser/vueparser/tsgo_parse_ctx.go` 抽出可复用能力：`Parse`（含 `ScriptKind`）、`imports` / `exports`、`resolveModuleSpec`、`convertReferenceWithModuleSpec`、`nodeText`、行列号
3. 任务：`vueparser` 改为调用公共层；行为与现有 `uiresource_parser` / `vuefileparser` 测试保持一致
4. 任务：**不**引入 i18n 业务逻辑；仅 parser 基础设施重构

测试清单：

1. 迁移/扩展现有 `vueparser/tsgo_parse_ctx_test.go`、`uiresource_parser_test.go` 覆盖
2. 用例：import / export / path alias / default export 行为不退化

完成定义（DoD）：

1. `internal/i18n/extract` 可直接依赖公共 tsgo 上下文，无需复制 `vueparser` 私有实现。
2. 现有 vueparser 相关测试全绿。

#### PR-S1-0b：term collector 骨架

建议标题：`feat(i18n): add term collector skeleton for extract`

依赖：**PR-S1-0a**；**S0-2**（Scope 算法与 golden 对齐）。

改造清单（函数级）：

1. 文件：`internal/i18n/extract/collector.go`（新）
2. 任务：参照 `vueparser/uiresource_parser.go` 实现 `ForEachChild` 全树 walk + `CallExpression` 识别
3. 函数：识别 `_t` / `_lt` 字面量调用（及 `_lt` 别名）；非字面量 **skip + warn**（§13 D12b）；忽略已废弃 `mode` / `output` / `_td` / `_tr`
4. 文件：`internal/i18n/extract/scope.go`（新）
5. 任务：Go 侧 Scope 推导骨架：`withI18nScope` 栈、enclosing symbol、file path；与 `modules/core/service/i18n/scope.ts` **同规则**（golden 对齐）
6. 任务：识别 `const { _t, _lt } = createTranslate('module', options?)` 绑定（含默认 scope，无 `output`），并校验 module 名与文件所属模块一致
7. 文件：`internal/i18n/extract/vue.go`（新）
8. 任务：`.vue` **script**：`vuesfchtmlparser` 拆块后喂 AST collector
9. 文件：`internal/i18n/extract/template_regex.go`（新）
10. 任务：`.vue` **template**：拆出 template 文本后**正则**匹配字面量 `_t`（如 `{{ _t('...') }}`、`:attr="_t('...')"` 等约定模式）；默认 Scope=`path@template`（§13 D12e）；非字面量 skip + warn
11. 任务：template regex 逻辑**集中单文件**，便于 S1+ 整体替换为完整 Vue template AST
12. 任务：抽取公共 `parseStringLiteral` / literal 判断（可自 `uiresource_parser` 提升到 `internal/parser` 或 `tsgoctx`）

测试清单：

1. 文件：`internal/i18n/extract/collector_test.go`、`template_regex_test.go`
2. 用例：`.ts` / `.tsx` / `.vue` script 中 `_t('literal')`、`_t('literal')`、`_t('literal')` 可收集
3. 用例：`.vue` template 中 `{{ _t('literal') }}` 等正则模式可收集；Scope 为 `path@template`
4. 用例：`{ scope: 'manual' }` / `withI18nScope` 覆盖 auto scope（script）
5. 用例：非字面量 `_t(x)` 跳过且有 warn（script 与 template 均适用）
6. 文件：`modules/core/service/i18n/scope_extract_golden.test.ts`（或 Go golden）与 runtime Scope 一致

完成定义（DoD）：

1. collector 可独立单测产出 `{ module, scope, src }` 列表（尚未要求写 pot / CLI）。
2. 未引入第二套 TS/Vue 解析实现；底层仍经 `typescript-go-internal`。

退出条件：PR-S1-1 可直接在此基础上接 `WritePot` 与 `cmd i18n extract`。

---

### 14.7 PR-S1-1：`choysum i18n extract`

目标：在 **PR-S1-0b collector** 基础上接 pot 写出与 CLI；扫描 `_t` 字面量；msgctxt=Scope；与 runtime Scope **同算法**（§13 D16）。

建议标题：`feat(cli): add choysum i18n extract`

改造清单（函数级）：

1. 文件：`cmd/cmd_i18n.go`（新）
2. 函数：`newI18nCmd` → 子命令 `extract`；挂到 `cmd/cmd.go` `rootCmd.AddCommand`
3. 文件：`internal/i18n/extract/pot.go`（新）
4. 函数：`ExtractModule(modulePath)` / `WritePot`（调用 S1-0b collector；输出 **GNU gettext** 合规 pot，含 `msgctxt`；§5.4·§13 D18）
5. 任务：模块扫描覆盖 `.ts` / `.tsx` / `.vue`（**script**=AST collector；**template**=regex，§13 D12e·D16）
6. 任务：CLI 侧接 `internal/vueplugin.ParseTsconfigPathAlias`；path alias 语义与现有 build parser 一致
7. 任务：非字面量 `_t(x)` **跳过 + warn**（§13 D12b；collector / template regex 均适用）
8. 任务：**不**调用 `xgettext` / `xgotext` / Node `gettext-extractor` 作为权威路径（§13 D18）
9. 任务：文档注明 template regex 为 **临时方案**；S1+ 迁移完整 Vue template AST（§13 D16）
10. 文件：`.dev/docs/infra/cmd/choysum_cli.md`
11. 任务：补充 `i18n extract` 用法

测试清单：

1. 文件：`internal/i18n/extract/extract_test.go` 与/或 `modules/core/service/i18n/scope_extract_golden.test.ts`
2. 用例：样例源码 → pot 含正确 msgid/msgctxt
3. 用例：手动 `{ scope }` / `withI18nScope` 覆盖 auto
4. 用例：Vue / TS / TSX 样例经现有 parser 抽取后，Scope 与 runtime 规则一致
5. 用例：Vue template 正则样例（`{{ _t('...') }}`）产出 `path@template` Scope
6. 用例：非字面量 `_t(x)` 不入 pot，且有 warn（§13 D12b）

完成定义（DoD）：

1. `go run . i18n extract` 对样例模块写出可检 pot。
2. `extract` 复用现有 parser 封装层，底层依赖 `typescript-go-internal`；不引入第二套 TS/Vue 解析实现。
3. 产出 pot 含 `msgid` + `msgctxt`，可被 gotext / 标准 gettext 工具读入（§13 D18）。
4. 不引入 `import`/`export` CLI；不以 `xgettext` 族为权威。

退出条件：§11 S1 中 extract 验收；CLI 无副作用于运行时。

---

### 14.8 PR-S1-2：`choysum i18n sync`

目标：用最新 pot 同步各语言 po：保留 msgstr、补新条、标 fuzzy/过期；语义对齐 **msgmerge**；**不**单独提供 merge/update（§13 D18）。

建议标题：`feat(cli): add choysum i18n sync`

改造清单（函数级）：

1. 文件：`cmd/cmd_i18n.go`
2. 函数：子命令 `sync`
3. 文件：`internal/i18n/sync/`（新）；实现可自写或封装系统 `msgmerge`
4. 函数：`SyncModulePo(module, lang)`
5. 任务：对外只暴露 `sync`；行为对齐 gettext `msgmerge`（保 msgstr、补新、obsolete）
6. 文档：更新 `choysum_cli.md`

测试清单：

1. 文件：`internal/i18n/sync/sync_test.go`
2. 用例：旧 msgstr 保留；新 msgid 空白出现
3. 用例：pot 中消失的 msgid → PO 标 **obsolete**（§13 D12a；不删 msgstr 历史）

完成定义（DoD）：

1. `extract` → `sync` 闭环在 fixtures 可复现。
2. 无并列 `merge`/`update` 命令。
3. 产出 po 仍为 gettext 合规文本（§5.4）。

---

### 14.9 PR-S2-1：PO → Go TermStore 导入器

目标：**Go** 用 **gotext** 解析 PO（msgctxt→Scope）写入本库；upgrade 不覆盖 `Source=override`；写后刷 Go cache（§13 D17·**D18**）。

建议标题：`feat(i18n): import module PO into Go TermStore`

改造清单（函数级）：

1. 文件：`internal/i18n/import/`（新）
2. 依赖：`github.com/leonelquinteros/gotext`（`go.mod`）；仅用 `Po` 解析，**不用**包级 `Get()` 作查词
3. 函数：`ImportModulePo({ application, module, lang, poText })` / `UpsertPoTerms`
4. 任务：`Source=packaged` upsert；若已有 `Source=override` 则跳过覆盖；**不因** PO obsolete 删除 DB 行（§13 D12a）
5. 函数：`DeleteModuleTerms(application, module)`（仅 uninstall）
6. 任务：导入后 Go cache invalidate + bump termHash（§13 D2）
7. 任务：无模块 `i18n/` 时 skip，不算错误；条目无 msgctxt → **拒绝该条 + 日志**（§13 D12c）

测试清单：

1. 文件：`internal/i18n/import/*_test.go`
2. 用例：msgctxt 映射 Scope；无 msgctxt 条目被拒绝（§13 D12c）
3. 用例：已有 `override` 不被 packaged upsert 覆盖
4. 用例：PO obsolete 不 DELETE 已有行（§13 D12a）
5. 用例：`DeleteModuleTerms` 清空该 Module；bridge/`_t` 不再命中旧译
6. 用例：gotext 解析多行 msgstr / 转义样例与 §5.4 一致

完成定义（DoD）：

1. 导入器单测全绿；不写任何 `dist/web/i18n`；无 TS PO import 权威路径。
2. 无 gotext 运行时查词路径（§13 D18）。

---

### 14.10 PR-S2-2：Lifecycle 接线

目标：install/upgrade 导入；uninstall 删词条；GlobalWebBuild **不**写术语快照；依赖 S0-1 已 ensure 表。

建议标题：`feat(lifecycle): wire PO import on install/upgrade/uninstall`

改造清单（函数级）：

1. 文件：`internal/module/lifecycle/modulemanager.go`（或既有 hook 扩展点）
2. 任务：Install/Upgrade 的 schema 阶段已通过 `ensureTranslationTermTable` 建表（§14.2）；本 PR 接 PO 导入
3. 任务：Install/Upgrade 成功后调用 **Go** `internal/i18n/import`（按模块 `i18n/*.po`）
4. 任务：Uninstall 调用 Go `DeleteModuleTerms` + Go cache invalidate
5. 文件：Language 变更钩子（base 模型或订阅点 → Go TermStore）
6. 任务：`IsActive` true→Go warm / false→Go evict + bump termHash
7. 任务：确认 `internal/module/artifact/build/web` **无**术语 JSON 写出

测试清单：

1. 文件：`internal/module/lifecycle/*_i18n*_test.go`（新）或扩展 `module_index_sync_test.go` 同类夹具
2. 用例：装模块后 TranslationTerm 有行；卸后清空
3. 用例：upgrade 再导不覆盖 `override`（夹具预置 `Source=override`）
4. 用例：构建路径无 `dist/web/i18n` 产物断言（可选）

完成定义（DoD）：

1. §11 S2 验收；回滚开关可跳过导入。
2. 无快照物化代码路径。

退出条件：装卸载闭环可测；与 meta 已装模块目录一致。

---

### 14.11 PR-S3-1：Go `{app}.I18n` / GetTranslations

目标：各 Application 只读本库；对外全名 `{app}.I18n/GetTranslations`；**Go 原生实现**（§13 D17）。

建议标题：`feat(i18n): add Go {app}.I18n GetTranslations`

改造清单（函数级）：

1. 文件：`internal/i18n/service/`（新）
2. 函数：`GetTranslations({ lang, moduleNames[], hash? })`（读 Go TermStore/cache）
3. 任务：过滤本 app 已装模块子集；未知 module 忽略
4. 任务：返回 `termsByModule` + termHash（§13 D2）；hash 命中则 `unchanged` 且 `termsByModule` 为 null（§13 D4）
5. 文件：`internal/service/` ApplicationService 注册面
6. 任务：为每个 Application 注册 `{app}.I18n` ServiceDesc（**含空实现**，§13 D5）；**Go handler**（**不再** handler→JS）
7. 任务：**仅宿主可 dial**（§8.7 / §13 D1）；**不做**跨库；**不**对终端用户 ACL allow `GetTranslations`

测试清单：

1. 文件：`internal/i18n/service/*_test.go` + 注册冒烟测
2. 用例：只返回请求 moduleNames 交集
3. 用例：termHash 一致 → unchanged 且 termsByModule null
4. 用例：无数据 / 空实现 → 空 termsByModule + 稳定 termHash（§13 D5）
5. 用例：全名形如 `auth.I18n/GetTranslations`；handler **不进** QuickJS
6. 用例：浏览器/gRPC-Web 直连被拒或不可达（§8.7）
7. 用例：termHash 算法与 §13 D2 夹具一致

完成定义（DoD）：

1. CE 单进程可 dial 自测；无 HTTP Gateway 亦可测。
2. 契约与 §8.3 / §8.7 / §13 D2·D4·D5·**D17** 一致。

---

### 14.12 PR-S3-2：宿主i18n Gateway（运行时词表）

目标：浏览器唯一读路径；聚合各 `{app}.I18n`；对外 vue-i18n `messages`；无 `moduleNames` 查询参数。

建议标题：`feat(i18n-gateway): aggregate GET /web/i18n/translations`

改造清单（函数级）：

1. 文件：`internal/i18n/gateway/gateway.go`（新）
2. 函数：`RegisterHandlers(mux, scopes...)`（对照 `documentgateway.RegisterSkeletonHandlers`；挂 **`internal/server` mux**，§13 D3）
3. 函数：`translationsHandler`：校验 lang → 已装模块目录 → 按 app 分组 → 并行 dial `{app}.I18n/GetTranslations`（**内部身份**，§13 D1）→ merge → `catalogHash`（§13 D2）
4. 任务：查询仅 `lang` + 可选 `hash`；**拒绝或忽略** `moduleNames`
5. 任务：响应 JSON §8.2 / §13 D4；可选 `ETag`=catalogHash（§13 D10，非必须）
6. 文件：`internal/server/server_base_routes.go`
7. 任务：`i18ngateway.RegisterHandlers(s.mux, s.runtimeScope)`
8. 任务：**宿主聚合**不进各业务 ApplicationService；**不**注册进 `WebHandlers`
9. 任务：`GET /translations` 为 HTTP Public（§8.7）；dial **不**要求浏览器 token

测试清单：

1. 文件：`internal/i18n/gateway/gateway_test.go`
2. 用例：mock 多 app 返回合并 messages
3. 用例：catalogHash 命中 → `unchanged=true` 且 messages 省略/null（§13 D4）
4. 用例：断言无跨库 SQL；仅 gRPC/进程内 dial；dial 身份为内部
5. 用例：匿名 GET 可读；带非法 moduleNames 被忽略
6. 用例：下游 GetTranslations 不对浏览器暴露（与 PR-S3-1 联测）
7. 用例：catalogHash 算法与 §13 D2 夹具一致

完成定义（DoD）：

1. §11 S3 验收；可卸路由回滚。
2. `WebHandlers` 仍只服务静态 dist；术语旁路在 server mux（§13 D3）。
3. 与 §8.7 / §13 D1·D2·D4 一致。

退出条件：与 documentgateway 同层次清晰；浏览器契约冻结。

---

### 14.13 PR-S4-1：FE setLocale ← Gateway

目标：i18nStore 改为 Gateway 拉词；`mergeLocaleMessage`；同步 RequestContext.lang。

建议标题：`feat(web): load terminology via Gateway and setLocale`

改造清单（函数级）：

1. 文件：`modules/web/web/stores/i18nStore/index.ts`
2. 函数：`setLocale` / `loadVueI18nMessages`
3. 任务：改为 `GET /web/i18n/translations?lang=&hash=`；处理 `unchanged`
4. 文件：`modules/web/web/stores/i18nStore/loader.ts`（扩）或新 `terminology_loader.ts`
5. 函数：`fetchWebTranslations(lang, hash?)`
6. 文件：`modules/web/web/app.ts`
7. 任务：成功时 `mergeLocaleMessage`；失败回退 msgid
8. 任务：`setGlobalRequestContextProvider` 写入 `lang`（内部码）与既有 `locale`
9. 任务：语言切换器仅列出 `Language.IsActive`（API 或配置源，剔除硬编码统一术语视图）
10. 文件：`modules/web/web/i18n/translate.ts`
11. 任务：导出唯一 `translateTerm(composer, reference, fallback)`；`_t` 复用；以真实 vue-i18n fallback 测试证明不需要 `te`
12. 任务：setup consumer 使用 global-scope Composer；router 由 `app.ts` 注入 `i18n.global`

测试清单：

1. 文件：`modules/web/web/stores/i18nStore/*.test.ts`（新/扩）
2. 用例：unchanged 不 merge 空对象
3. 用例：Gateway 失败时 UI 仍显示英文 msgid
4. 用例：setLocale 后 RequestContext 含正确 lang
5. 用例：adapter 缺 reference / Composer / key 安全回退；catalog merge 与 locale 变化保持响应

完成定义（DoD）：

1. 不依赖 `dist/web/i18n` 文件。
2. 旧 `source` 手写 messages 可暂时共存，但 setLocale 主路径走 Gateway。
3. FE 术语引用显示只存在 §7.2 两种标准形态，无对象专用翻译 wrapper。

---

### 14.14 PR-S4-2：web 壳层 PO 化 + `web.I18n`

目标：试点将 `modules/web/web/i18n/source` 等迁到 `modules/web/i18n/*.po`；壳层由 **`web.I18n`** 服务（非全站聚合）。

建议标题：`refactor(web): migrate shell copy to PO + web.I18n`

改造清单（函数级）：

1. 文件：`modules/web/i18n/web.pot`、`zh_CN.po`（新）
2. 任务：`extract`/`sync` 产物纳入 git
3. 文件：壳层 Vue/TS（`App.vue`、header、欢迎页等）逐步 `_t`
4. 文件：`modules/web/service/`（新，极薄）+ 注册 `web.I18n`
5. 任务：装载本 app `TranslationTerm`；**禁止** `web.I18n` 跨 App API
6. 文件：删除或停用旧手写 `zh-CN` 术语源（迁完后）
7. 任务：`SUPPORTED_LOCALES` 仅保留格式/Element，非术语存储权威
8. 任务：Vue template 直接 `$t(reference.key, reference.src || fallback)`；TS / h-render / router 统一 `translateTerm`

测试清单：

1. 手动/ e2e：中英切换壳层文案
2. 用例：`web.I18n/GetTranslations` 仅返回 web 模块词
3. 用例：pot 漂移 CI（可先手工，S6 再硬门禁）
4. 用例：header / breadcrumb 直接 template 翻译；menu h-render、Selection、document title 走 adapter；静态门禁禁止 semantic wrapper

完成定义（DoD）：

1. 壳层可见文案来自 Gateway；无业务译文堆在 `web/web/i18n`。
2. `web.I18n` 与 §8.3.1 一致。
3. 不直接显示 reference `src`，不保留 `menuTitle` / `displayTitle` 等中间投影。

---

### 14.15 PR-S4-3：auth 试点 + User.Language

目标：auth 模块 PO + 错误 message `_t`；偏好写入 `User.Language`。

建议标题：`feat(auth): pilot terminology i18n and User.Language`

改造清单（函数级）：

1. 文件：`modules/auth/i18n/`（新）
2. 文件：`modules/auth/service/**` 用户可见错误：`{ code, message: _t(...) }`
3. 文件：`modules/auth/service/models/user.ts` / FE 偏好写入
4. 任务：登录用户切语言写 `User.Language`；匿名 localStorage
5. 文件：`modules/auth/web/**` 登录页等 `_t`
6. 任务：FE 错误展示约定：code 不译、message 展示术语

测试清单：

1. 文件：auth 既有 service/web 测试扩
2. 用例：切语言后错误 message 随 lang
3. 用例：User.Language 持久化并在下次进入生效

完成定义（DoD）：

1. §12 清单项 3–4 对 auth/web 成立。
2. 可回滚为显示 msgid。

---

### 14.16 PR-S4-4：locale remount / full reload

目标：切语言后避免 UI 新、缓存/RPC 旧；**S4-MVP 默认 `location.reload`**（§13 D9）。

建议标题：`feat(web): locale-remount (or full reload) on language switch`

改造清单（函数级）：

1. 文件：`modules/web/web/stores/i18nStore/index.ts`（或新建 `locale_remount.ts`）
2. 函数：`afterLocaleChange()`
3. 任务：**默认** `location.reload`；可选 flag 试 remount（清路由视图 / 表单 store）
4. 任务：边界对齐 §13 D9；PR 描述写清；S6 再收紧

测试清单：

1. 用例/手工：切语言后显式标记的动态文案与 RequestContext 一致
2. 默认 reload 路径可测；flag 开 remount 时行为可测

完成定义（DoD）：

1. 默认策略为 reload（§13 D9）；remount 边界可延后 S6。

---

### 14.17 PR-S5-1：Go `{app}.I18n` 写路径 + Source

目标：鉴权 `SearchTerms` / `UpdateTerm`（供 Gateway dial）；`Source=override`；PO upgrade 不覆盖；bump termHash（§13 D2）；**无**广播；**Go 实现**（§13 D17）。

建议标题：`feat(i18n): Go {app}.I18n SearchTerms/UpdateTerm + Source`

改造清单（函数级）：

1. 文件：`internal/i18n/service/` + `internal/i18n/store/`（扩展）
2. 函数：`UpdateTerm({ module, lang, scope, src, value, kind? })` → `Source=override` → 刷 Go cache
3. 函数：`SearchTerms({ lang, modules?, q?, limit, offset, … })` → 响应对齐 §13 D7
4. 任务：挂 ACL **default-deny**；种子角色 **`Terminology Editor`** Service 精确 allow（§13 D6）
5. 任务：确认 Go import 路径跳过 `override`（与 S2-1 合并验收）
6. 任务：**不做** TranslationTerm 公共 CRUD；**不**用 FieldRule/RecordRule 当主 ACL
7. **不做**：全会话 WebSocket/SSE；entryPolicy Public 化 `*.I18n/*`；TS `app_i18n_service` 写路径

测试清单：

1. 用例：改译后同进程 `_t`（经 bridge）立即新值
2. 用例：再 import PO 不覆盖 `override`
3. 用例：无权限 / 非法 module → PermissionDenied
4. 用例：SearchTerms 分页与 module 过滤；响应含 status（§13 D7）
5. 用例：`Terminology Editor` 角色 Service 精确 allow 后可写（§13 D6）

完成定义（DoD）：

1. §11 S5 后端契约就绪；与 §8.7 / §13 D6·D7·**D17** 一致；浏览器入口由 PR-S5-2 承接。

---

### 14.18 PR-S5-2：Gateway terms 读/写

目标：宿主统一 `GET|PATCH /web/i18n/terms`；鉴权；fan-out dial `{app}.I18n`；与读 translations 同包。

建议标题：`feat(i18n-gateway): terms GET/PATCH for Terminology Editor`

改造清单（函数级）：

1. 文件：`internal/i18n/gateway/terms_handler.go`（新）等
2. 函数：`termsListHandler` / `termsPatchHandler`
3. 任务：HTTP Authenticated + **Terminology Editor**（§13 D6）；校验 `application`；dial 时 **转发用户 identity**（§13 D1）
4. 任务：列表/写契约对齐 §13 D7；All-apps 约束对齐 §13 D8（无 q 时 application 必填；带 q 的 All limit 封顶不分页）
5. 任务：注册到与 `translations` 同一 `RegisterHandlers`（server mux，§13 D3）；**不** Public
6. **禁止**：跨库 SQL；匿名 GET/PATCH terms；跨 app 无搜索分页

测试清单：

1. 文件：`gateway_terms_test.go`
2. 用例：PATCH 路由到正确 app mock；批量全成功或全失败（§13 D7）
3. 用例：未鉴权拒绝；未知 application 拒绝；无 Terminology Editor 拒绝
4. 用例：无 q + 无 application → 4xx；带 q 的 All fan-out + truncated（§13 D8）
5. 用例：dial 转发用户 identity（对照 GetTranslations 内部身份）
6. 用例：匿名可读 translations、不可读 terms

完成定义（DoD）：

1. Terminology Editor 可只依赖 HTTP；浏览器不 dial `{app}.I18n`。
2. 与 §8.7 / §13 D1·D6·D7·D8 一致。

---

### 14.19 PR-S5-3：Terminology Editor + 术语重载

目标：单入口 Terminology Editor；**只调 Gateway** terms API；术语重载；**无** per-app 改译菜单。

建议标题：`feat(web): Terminology Editor via Gateway and terminology reload`

改造清单（函数级）：

1. 文件：`modules/web/web/pages/.../TerminologyEditor.vue`（新）+ 路由/菜单（靠近 base Language）；菜单权限绑 **Terminology Editor**
2. 任务：语言 / Application / Module 筛选；专用网格（§8.6）；**非** `createStoreByModel(TranslationTerm)`
3. 任务：列表 `GET /web/i18n/terms`；保存 `PATCH /web/i18n/terms`；UX 对齐 §13 D8（无搜索须选 application；跨 App 搜靠 `q`）
4. 任务：保存成功 → 术语重载；`reloadTerminology()` → `GET /web/i18n/translations`
5. 任务：网格展示 Application/Module/Scope；组件提示仅 Scope path 启发式（§13 D11）
6. **禁止**：FE 直连 `{app}.I18n`；各业务 App TranslationTerm 菜单；`web.I18n` 跨 App API

测试清单：

1. 手工：改译 → 本会话重拉可见；其他会话不自动推送
2. 用例：请求只打到 `/web/i18n/terms`
3. 用例：reloadTerminology 走 Gateway catalogHash
4. 用例：搜索「Log In」在 All+q 下命中 `application=auth`（OHeader patch 夹具）
5. 用例：仅筛 `web` 时不出现 auth 补丁 msgid

完成定义（DoD）：

1. 无需 rebuild；无广播；与 §8.2.2 / §8.6 / §12 #8–#11 / §13 D8·D11 一致。

---

### 14.20 PR-S6-1：`status` + CI

目标：缺译/fuzzy/orphan Scope 汇总；CI 可红。

建议标题：`feat(cli): add choysum i18n status and CI gates`

改造清单（函数级）：

1. 文件：`cmd/cmd_i18n.go` 子命令 `status`
2. 文件：`internal/i18n/status/`（新）
3. 函数：`StatusReport(modules) → exitCode`
4. 任务：CI：`extract` 漂移（pot dirty）与/或 `status` 非零失败
5. 文档：`choysum_cli.md`；可选 workflow 片段
6. **不做**：独立 `missing` 命令

测试清单：

1. 文件：`status_test.go`
2. 用例：缺译 → 非零；干净树 → 0
3. 用例：orphan Scope / obsolete 仍留 DB 的行可检测（§13 D12a）

完成定义（DoD）：

1. §11 S6 验收；关 CI 不影响运行时。

---

### 14.21 PR-S6-2：体验收口

目标：语言切换 remount 边界定稿；可选管理端「导出当前语言 PO 下载」（HTTP/UI，**不是** `choysum i18n export`）。

建议标题：`chore(i18n): polish locale-remount bounds and optional PO download`

改造清单（函数级）：

1. 收紧 PR-S4-4 的 remount 范围或文档冻结
2. 可选：settings「下载 PO」生成文件（只读导出，CLI 仍无 export）
3. 对照 §12 总检逐项勾选

测试清单：

1. 回归：web+auth 切语言；装卸载；`override` 保护；Gateway unchanged
2. 跑相关 unit / 必要 e2e

完成定义（DoD）：

1. §12 全部勾选；§13 已决议项已落地或明确延期（如 remount 边界、All-apps 分页）。

---

### 14.22 执行顺序与并行建议

```text
S0-1 ──► S0-2 ──┬──► S1-0a ──► S1-0b ──► S1-1 ──► S1-2 ──► S6-1（可晚）
                │      ▲
                │      └──（S1-0b 依赖 S0-2 Scope）
                ├──► S2-1 ──► S2-2
                ├──► S3-1 ──► S3-2 ──┬──► S4-1 ──► S4-2 / S4-3 / S4-4
                │                     └──► S5-1 ──► S5-2 ──► S5-3
                └──► S0-3（可与 S0-2 并行）
S1-0a 可与 S0 并行，但须在 S1-0b / S1-1 前合入
S6-2：主路径稳定后
```

1. **阻塞线：** S0-2 → S1-0b → S1-1；S0-2 → S3 → S4/S5；无 Gateway 不做 FE 主路径。  
2. **可并行：** S1-0a 与 S0；S1 与 S2；S0-3 与 S0-2；S4-2 与 S4-3；S5-1 可与 S3-2 尾部并行，但 S5-2 依赖 S5-1。  
3. **必须最后收口类：** S6-2；S5-3 依赖 S5-2（Terminology Editor 等 Gateway terms）。  
4. 每个 PR 保持可回滚（flag 或未接线）。

---

### 14.23 每个 PR 的 Reviewer 检查项

1. 是否引入 `dist/web/i18n/**` 或 GlobalWebBuild 写术语快照？→ **必须否**。  
2. 是否新增 `choysum i18n import|export|merge|update|missing`？→ **必须否**（能力只在 sync/status）。  
3. Gateway 是否跨 Application 读库或让浏览器读路径传 moduleNames？→ **必须否**。  
4. `_t` 热路径是否 RPC 跨服务查词？→ **必须否**。  
5. Scope 是否用行号 / `_t.distinct`？→ **必须否**。  
6. 缓存是否按 `Language.IsActive`（而非 Locale.IsActive）？→ **必须是**。  
7. PO upgrade 是否覆盖 `Source=override`？→ **必须否**（S2 起预留，S5 验收）。  
8. 改译后是否做全会话广播？→ **必须否**（仅术语重载）。  
9. 对外读 `messages` 是否可 `mergeLocaleMessage`；对内是否 canonical { module, scope, src, kind }（kind 缺省 literal）？→ **必须是**。  
10. 数据 i18n / 业务 `res_id` 是否混入 TranslationTerm？→ **必须否**。  
11. 是否引入独立 `i18n` Application（作唯一存储层）或「唯一存储层 + 副本同步」管线？→ **必须否**。  
12. 对外契约是否为 `{app}.I18n/...`（而非 TranslationTerm 常规 CRUD）？→ **必须是**。  
13. 是否为每个业务 App 挂 TranslationTerm 公共改译菜单？→ **必须否**（仅 Terminology Editor）。  
14. `web.I18n` 是否出现跨 App Search/Update？→ **必须否**。  
15. `{app}.I18n` 是否仍 handler→JS / TS 权威 cache？→ **必须否**（§13 D17：Go handler + Go TermStore/cache；`_t` 仅 sync bridge）。  
16. Terminology Editor / FE 是否直连 `{app}.I18n`（绕过 Gateway terms/translations）？→ **必须否**。  
17. 术语读、写是否都经宿主 `/web/i18n/**` Gateway？→ **必须是**（S3 读 + S5 写）。  
18. 视图 patch / IMD 扩展中的 `_t` 是否归入**声明模块**（而非基座模块）？→ **必须是**（§4.1）。  
19. Terminology Editor 是否仅靠「先选 Application」才能找到补丁词条、且无跨 App 搜索？→ **必须否**（S5-MVP 起默认跨 App 搜）。  
20. `GET /web/i18n/translations` 是否 Public、且 `GET|PATCH /web/i18n/terms` 是否 Authenticated？→ **必须是**（§8.7）。  
21. `{app}.I18n/GetTranslations` 是否对浏览器/gRPC-Web 开放，或被 entryPolicy Public 化？→ **必须否**。  
22. `SearchTerms`/`UpdateTerm` 是否 ACL default-deny；是否用 FieldRule/RecordRule 或 TranslationTerm Model CRUD 当主写 ACL？→ **前者必须是，后者必须否**。  
23. `extract` 是否旁路实现第二套 TS/**script** 解析，或直接用 `backendtsparser.Parse()` 抽 `_t`？→ **必须否**（§13 D16；script 走 S1-0 walk；template S1-MVP 允许 regex，S1+ 换 template AST）。  
24. Vue `<template>` 是否在 S1-MVP 完全跳过？→ **必须否**（须 regex 覆盖常见字面量；§13 D12e）。  
25. `_t` 热路径是否走 async `$choysum.db.query` 或自 dial `{app}.I18n`？→ **必须否**（须 sync `$choysum.i18n`；§13 D17）。  
26. PO import 是否仍以 TS 为唯一权威写路径？→ **必须否**（Go import；§13 D17）。  
27. pot/po 是否偏离 GNU gettext（缺 msgctxt / 非标准字段当查词键）？→ **必须否**（§5.4·§13 D18）。  
28. extract 是否以 `xgettext`/`xgotext`/`gettext-extractor` 为权威？→ **必须否**（§13 D18）。  
29. 是否用 gotext/`node-gettext` 运行时 `Get()` 替代 TermStore？→ **必须否**（gotext 仅 PO 读写；§13 D18）。  
30. `createTranslate` 是否同形返回 `{ _t, _lt }`，且 `_t`→string、`_lt`→`TermReference`（`_lt` 禁插值；无 `output`）？→ **必须是**；Vue lazy/reactive 是否分别使用显式闭包 / `computed(() => _t(...))`？→ **必须是**（§5.1·§13 D18·D21）。
31. Vue template 中术语引用是否直接使用 `$t(reference.key, reference.src || fallback)`，且 optional reference 仅以 plain string 条件回退？→ **必须是**（§7.2）。  
32. TS / computed / h-render / router 是否统一调用 `translateTerm(composer, reference, fallback)`，并通过 global-scope Composer 或组装层注入保持响应？→ **必须是**（§7.2）。  
33. 是否直接显示 reference `src`，或新增 menu/breadcrumb/selection/router 等对象专用翻译 wrapper / `displayTitle` 投影？→ **必须否**（§5.5·§7.2）。
34. `createI18n.postTranslation` 是否保留 revision dependency 且原样返回译文；未来 handler 是否组合而非覆盖？→ **必须是**（§7.2）。
35. Gateway catalog 是否由单一路径按 `locale + catalogHash` 去重，并仅在成功 merge 新的非空 catalog 后 notify；Editor 同 locale 新 hash 是否仍刷新？→ **必须是**（§7.1）。
36. 是否仍使用已废弃的 `output` / `mode: 'descriptor'` / `_td` / `_tr` / `TextDescriptor`，或把 `_lt` 当 Odoo LazyGettext？→ **必须否**（§5.1·§13 D18·D21）。
37. `defineModelActions` 的 `entityTitle`/`titles` 是否使用 `_lt`，并持久化 `Title` + `TitleText`、权限树走 §7.2？→ **必须是**（§5.5·§13 D19）。
38. 种子/demo JSON 是否**禁止** `{ "_t" }` / extract 入 pot / 术语展示适配器？→ **必须禁止**（属数据 i18n；§5.6·§13 D20 否决）。

---

### 14.24 与 §11 / §12 映射速查

| §11 Stage | PRs | §12 验收项 |
|---|---|---|
| S0 | S0-1..3 | #6（部分）、#7、#12、#13、#15（bridge/cache 骨架） |
| S1 | S1-0a..2 | #1（工具侧）、#14、#16（pot 合规） |
| S2 | S2-1..2 | #1（装后 BE）、#5、#9、#15、#16（gotext import） |
| S3 | S3-1..2 | #2、#9、#10（读）、#11、#15 |
| S4 | S4-1..4 | #3、#4、#6 |
| S5 | S5-1..3 | #8、#10（写）、#11（terms Authenticated / UpdateTerm ACL） |
| S6 | S6-1..2 | 质量门禁 + §12 收口 |
