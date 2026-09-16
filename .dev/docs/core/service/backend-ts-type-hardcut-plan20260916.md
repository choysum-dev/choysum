<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# 后端 TS 类型硬切方案（`modules/*/service`）

更新时间：2026-09-16（同日修订：取消 BT-3 / 原 HC4 policy pipeline）

**状态：** 执行稿（硬切；**不考虑类型向后兼容**）。架构结论对齐 [type-generics-optimization-design.md](./type-generics-optimization-design.md) 的 **GT\***；本文把它收成可排期的 PR 切片，并按 2026-09-16 生产盘点冻结实现选择。

**范围：** `modules/*/service/**/*.ts`（后端模型 / mixin / ORM / runtime / rpc）。  
**不在本文：** Vue / `modules/web/web/**` store（仍走原文 **Track B**）。

**一句话目标：** 让 `Model.Search/Create/Update` 的 `T` 来自**这个类**，让生产 CRUD override 用 **HC2 签名**写清不变式与领域变换，从而让生产路径不再需要 `as any`。

**硬切含义：** 无 deprecated 别名、无双签名过渡窗、无 `UntypedModelService` 默认重载、无 `{ new (...args: any[]) }` 逃生 ctor。类型破了就同 PR 改调用点。  
**不是硬切：** gRPC / 作者面动词 PascalCase（`Create`/`Search`）、环境轴语义、soft-delete / company / sudo 行为——那些是产品合同，见 [basemodel-author-api-design.md](./orm/basemodel-author-api-design.md) **A3**。

---

## 0. 怎么读

| 读者 | 路径 |
| --- | --- |
| 拍板 | §1 冻结 → §2 目标架构 → §7 分期 |
| 实现 | §3 现状 → §4 目标签名 → §5 删除清单 → **§7（PR / 文件 / 函数 / 测试）** |
| 验收 | §8 门禁 |

**相关：**

| 文档 | 关系 |
| --- | --- |
| [type-generics-optimization-design.md](./type-generics-optimization-design.md) | 多轨设计 SSOT（含 FE Track B）；本文 = 后端硬切执行 |
| [basemodel-author-api-design.md](./orm/basemodel-author-api-design.md) | 静态动词白名单；不改 PascalCase |
| [field-defaults-design.md](./orm/field-defaults-design.md) | `DefaultGet` pipeline（引擎挂钩样板；与 write-policy 无绑定） |
| [business_models_layout_refactor_plan20260831.md](./business_models_layout_refactor_plan20260831.md) | 模型文件布局；`AuthzMutationModel` **保留** CRUD override（HC2），不与已取消的 BT-3 合流 |

---

## 1. 冻结结论（冲突时以本表为准）

### 1.1 总则（HC\*）

| ID | 结论 |
| --- | --- |
| **HC1** | 只保留一套完整构造器类型：`ModelCtor<T>`（`ModelClass<T> & typeof BaseModel`）。删除 `BaseModelCtor` / `InstantiableModelCtor` / `RuntimeModelCtor`。metadata `field.ts` 只 re-export 这一套，禁止第三套定义。 |
| **HC2** | 集合入口用 **构造器多态**：`Create<C extends ModelCtor>(this: C, …): … InstanceType<C>`。废除方法级 `T extends BaseModel`。 |
| **HC3** | 写载荷是 `Partial<Insertable<InstanceType<C>>>` / `Partial<Updateable<InstanceType<C>>>`。废除 `T & BaseModel`。 |
| **HC4** | **取消原「policy pipeline」硬切。** IMD 下用户本就可 override CRUD / 改 `@Model`；metadata policy **约束不住**下游，也不比 override 更「正确」。append-only、stamp actor、normalize、authz cache invalidate、`_prepareValues` 等 **允许**留在模型 / mixin 的 CRUD override 上，但必须 **HC2 签名**（与 HC5 同一纪律）。不把「消灭平台 CRUD override」当硬切 DoD。 |
| **HC5** | 领域写变换（如 Role 的 UI grant 同步）可以留在模型上，必须用 **HC2 签名**，payload 用 `Insertable<Role>`，**禁止** `value as any` / `returnFields as any`。 |
| **HC6** | `ModelService<C>` 的 CRUD 显式绑 `InstanceType<C>`。`createServiceByModel` / `BaseModel.dial` **必须**带模型类型参数；删除无泛型的 untyped 重载。 |
| **HC7** | 带 `fields` / `returnFields` 时返回 **诚实投影** `Projected<T, F>`（V1 = 顶层字面量 `Pick`；`'*'` / 省略 = 满 `T`）。禁止继续标满 `T` 却只选出子集。 |
| **HC8** | 动态拼 domain 走已有 `BaseQueryCondition`，在进 `Search` 前用 `condition<T>(…)` helper 收口。禁止 `QueryCondition<any>` 和调用点 `as any`。 |
| **HC9** | `$choysum` 只在 `modules/core/types/$choysum.d.ts` 声明一次。删除 `runtime/context/source.ts` 的 `unknown` 再声明。 |
| **HC10** | 生产 `modules/*/service`（排除测试）以 `no-explicit-any` 为终态。测试另用 testkit，**不设归零 KPI**。 |
| **HC11** | `ModelAPI<T>` 保持 **internal**（引擎实现）。作者面仍是静态 PascalCase 动词。 |
| **HC12** | 不引入第二套 `any` escape hatch；边界用 `unknown` + type guard。 |

### 1.2 相对 2026-08-17 设计稿拍死的开放题

原文 §15 对后端仍有效的选择，本文硬切为：

| 原题 | 硬切选择 |
| --- | --- |
| Q1 Projected | V1 浅 `Pick`；嵌套路径后置（BT-4） |
| Q2 appendOnly | 模型 CRUD override throw（现 `FieldChange`）；**不做** `@Model({ appendOnly })` 引擎硬切；不做窄 `AppendOnlyModelAPI` |
| Q3 ModelAPI 可见性 | 长期 internal |
| Q8 prepare/stamp | 留在模型 override / helper；要 IO 可走 override 或已有 create pipeline 步（对标 DefaultGet）。**不**强制 sync-only metadata prepare |
| Q9 计数 | `scripts/` 可复现脚本 + PR 贴数；**不** CI hard fail |

FE 的 Q4/Q6/Q7 仍归原文 Track B，本文不拍。

---

## 2. 目标架构

```text
@Model class Role extends BaseModel
        │
        ├─ 字段 → Selectable<Role> / Insertable<Role> / Updateable<Role>
        │
        └─ 可选 HC2 CRUD override（领域变换 / 平台不变式均可）
                        │
                        ▼
              内部 ModelAPI<Role>     ← 引擎 only
                        │
                        ▼
              BaseModel 静态 thin facade   ← 作者 + gRPC 同形
                        │
                        ▼
              ModelService<typeof Role>    ← createServiceByModel / dial
                   CRUD 参数 = InstanceType<typeof Role> = Role
```

三层数据，不再用 `any` 当第四层：

| 层 | 类型 | 谁用 |
| --- | --- | --- |
| 调用方 | `Insertable<T>` / `Updateable<T>` / `QueryCondition<T>` / `FieldSelection<T>` | 业务模型、RPC 入参 |
| 投影 | `Projected<T, F>` / `Selectable<T>` | Search/Browse 返回 |
| 内部袋 | `unknown` + guard；动态树用 `BaseQueryCondition` | 规则引擎、override 里改 key 之前 |

---

## 3. 现状盘点（2026-09-16）

**方法：** `modules/*/service/**/*.ts` 扫 `\bas any\b`；生产 = 非 `*.test.ts` / 非 `**/tests/**`。

| 范围 | 文件 | `as any` |
| --- | --- | --- |
| 生产 | 87 | **1080** |
| 测试 | 273 | **11816** |
| 生产 `: any` 标注 | 53 | **236** |
| 生产 override `this: { new (...args: any[]) }` | — | **47** |
| 生产 `options?: any` | — | **18** |

生产按模块：`auth` 439 · `core` 215 · `meta` 126 · `base` 118 · `document` 104 · 其余 &lt; 80。

生产模式（近似，有交叉）：

| 模式 | 约次数 | 根因 |
| --- | --- | --- |
| CRUD 参数 / `(this as any).Search` | 150+ | `T` 绑不到类；ctor 不互通 |
| record bag `(row as any).Id` | 150 | 返回标满 `T`、实际 Partial；或 `unknown` 无 guard |
| `super.Create(... as any)` | 64 处 × 多参数 | override `this` 与 `BaseModelCtor` 双轨 |
| 对象字面量 payload | 77 | Insertable 推断失败（连带 HC2） |
| `this as any` 当 ctor | 57 | Instantiable vs ModelCtor |
| `$choysum as any` | 28 | 双重 declare |
| `getModelMetadata(ctor as any)` | 17 | ctor 家族分裂 |
| `null as any` | 12 | 互斥 FK 清空 |

热点文件几乎全是 **CRUD override 模板**：`role.ts`、`role_method_access.ts`、`AuthzMutationModel`、`property_definition_base_model.ts`、`translation_term_base_model.ts`、`field_default_base_model.ts`、`user.ts`、`ui_resource.ts`、`attachment_binding.ts`。

**根因不是「少写了类型」，是 T 在三个边界丢掉：**

1. **override `this` 与 `BaseModel.Create` 的 `this` 不是同一种 ctor。** 子类一 override，`super.*` 全军 `as any`。`AuthzMutationModel` 把这套复制进所有 Role\*。
2. **方法级泛型 `Search<T extends BaseModel>` 经 `ModelService` mapped type 抽取后，`T` 塌成 `BaseModel`。** `createServiceByModel<typeof MetaModel>(...)` 的 `Search(['Application','=',x])` 非法。
3. **ctor 四套并存**（`ModelCtor` / `BaseModelCtor` / `RuntimeModelCtor` / `InstantiableModelCtor`），metadata / mixin / abstract 对不上。

只改调用点、不改这三处，生产 1080 处清不掉。

---

## 4. 目标签名（实现抄这里）

名称以落地 PR 为准，语义不许漂。

### 4.1 唯一 ctor

文件：`modules/core/service/orm/model/types.ts`

```ts
import type BaseModel from './model';
import type { SelectResult } from '../repository/types/common';
import type { FieldSelection } from '../repository/types/selection';

export type ModelFactoryArgs<T extends BaseModel = BaseModel> = [
  factoryToken: symbol,
  entity: SelectResult,
  fields?: FieldSelection<T>,
];

/** 可 new 出 T 的模型类（含 abstract mixin 的 subclass）。 */
export type ModelClass<T extends BaseModel = BaseModel> = abstract new (
  ...args: ModelFactoryArgs<T>
) => T;

/** 集合入口 / metadata / getModelMetadata 的 this。 */
export type ModelCtor<T extends BaseModel = BaseModel> = ModelClass<T> & typeof BaseModel;
```

删除：`BaseModelCtor`、`InstantiableModelCtor`、`RuntimeModelCtor`。  
`orm/metadata/field.ts` 的旧 `ModelCtor<T> = new (...args: never[]) => T` 改为 **re-export** `orm/model/types.ts` 的 `ModelCtor`（禁止第三套定义）。

`getModelMetadata(target: ModelCtor)`；`getEffectiveConstraints(this: ModelCtor)` 不再 `this as any`。

### 4.2 集合入口（BaseModel）

```ts
static Create<C extends ModelCtor>(
  this: C,
  value: Partial<Insertable<InstanceType<C>>>,
): Promise<InstanceType<C>>;
static Create<C extends ModelCtor, F extends FieldSelection<InstanceType<C>>>(
  this: C,
  value: Partial<Insertable<InstanceType<C>>>,
  returnFields: F,
): Promise<Projected<InstanceType<C>, F>>;

static Search<C extends ModelCtor>(
  this: C,
  condition?: QueryCondition<InstanceType<C>> | [],
  options?: SearchOptions<InstanceType<C>>,
): Promise<InstanceType<C>[]>;
static Search<C extends ModelCtor, F extends FieldSelection<InstanceType<C>>>(
  this: C,
  condition: QueryCondition<InstanceType<C>> | [],
  options: SearchOptions<InstanceType<C>> & { fields: F },
): Promise<Array<Projected<InstanceType<C>, F>>>;
```

`CreateMany` / `Browse` / `BrowseMany` / `Update` / `UpdateById` / `Count` / `ReadGroup*` / `Delete*` 同构。  
`BrowseMany` 的 fields 与 `Browse` 对齐为 `FieldSelection`（废 `(keyof Selectable<T>)[]`）。  
`OrderBy` 的 `field` 用 `Extract<keyof Selectable<T>, string>`。

内部委托：

```ts
return getModelApi<InstanceType<C>>(this).create(value, returnFields);
```

`getModelApi` / `ModelAPI<T>` **不导出到** `service/api`。

### 4.3 投影（V1）

文件：`orm/repository/types/selection.ts`（与 FieldSelection 同层）

```ts
type FieldName<T> = Exclude<FieldSelection<T>[number], object | '*'>;

export type Projected<T, F extends FieldSelection<T>> =
  Extract<F[number], '*'> extends never
    ? Pick<Selectable<T>, Extract<F[number], FieldName<T>>>
    : Selectable<T>;
```

V1 不展开 `DeepRelationSelection`（嵌套仍标关系字段的完整 Selectable 或保持关系对象类型）。嵌套诚实投影另开 PR，不阻塞 BT-4。

### 4.4 ModelService / dial

文件：`modules/core/rpc/types/model.ts`、`service/rpc/service_factory.ts`

```ts
type Row<C extends ModelCtor> = InstanceType<C>;

type CrudService<C extends ModelCtor> = {
  Create: typeof BaseModel.Create extends (this: C, ...a: infer A) => infer R
    ? (...a: A) => Promise<ClientModel<Awaited<R>>>
    : never;
  Search(...): Promise<Array<ClientModel<Row<C>>>>;
  // CreateMany / Browse / Update / … 手写绑定，不靠 Parameters<泛型函数>
};

export type ModelService<C extends ModelCtor> = CrudService<C> & {
  [K in Exclude<ModelServiceMethodKey<C>, keyof CrudService<C>>]:
    C[K] extends RpcServiceFn ? ClientModelService<C[K]> : never;
};

export function createServiceByModel<C extends ModelCtor>(
  modelName: string,
): ModelService<C>;
```

删除无类型参数重载。调用约定：

```ts
const MetaModel = createServiceByModel<typeof MetaModelModel>('meta.MetaModel');
await MetaModel.Search(
  { And: [['Application', '=', appName], ['Name', '=', modelName]] },
  { fields: ['Id'], limit: 1 },
);
```

`BaseModel.dial` 同样强制 `dial<typeof Partner>('partner.Partner')`，默认不再是 `Record<string, (...args: unknown[]) => unknown>`。

### 4.5 写路径不变式：留在 HC2 CRUD override（原「Model policy」已取消）

**2026-09-16 修订：** 不再落地 `@Model` write policy pipeline（原 BT-3 / 旧 HC4）。

Choysum 是编译期 **IMD**：下游可再声明同名模型并 override CRUD，也可改写 `@Model` options。metadata 开关 **约束不住**用户；在此前提下「禁止平台自己用 CRUD override」没有安全或架构优势，只是风格偏好。

| 不变式 | 继续怎么写 | 类型要求 |
| --- | --- | --- |
| append-only | 模型 `Update*` / `Delete*` override throw（如 `FieldChange`） | HC2 |
| stamp actor | `Create` / `CreateMany` override 盖 `getUserId()`（如 `FieldChange.ActorUid`、`Message.AuthorUid`） | HC2；payload `Partial<Insertable<…>>` |
| sync normalize | `_prepareValues` / `prepareCreatePayload` 仍可由 Create/Update override 调用 | 入参勿 `as any`（BT-2 已收口） |
| authz cache invalidate | `AuthzMutationModel`（或等价 mixin）包一层 Create/Update/Delete*，或模型自行 invalidate | HC2；保留 mixin 可以 |

领域变换（Role `AccessUiResourceIds` 同步）本来就是 override：与上表同一纪律——**允许 override，不允许 `as any`**（HC5）。

可选后续（**非硬切**）：若仅想减少样板，可再评估横切 helper；不作为硬切门禁，也不引入未定义继承语义的 `@Model` policy。

### 4.6 unknown 边界

```ts
export type RefLike = { Id?: unknown; id?: unknown };

export function isRefLike(value: unknown): value is RefLike { /* … */ }

export function normalizeRefId(value: unknown): string | null {
  if (value == null) return null;
  if (typeof value === 'string') { /* trim */ }
  if (isRefLike(value)) { /* Id ?? id */ }
  return null;
}
```

动态 condition：

```ts
export function condition<T>(tree: BaseQueryCondition): QueryCondition<T> {
  return tree as QueryCondition<T>;
}
```

这是**唯一**允许的断言函数，集中在 `orm/repository/types/query.ts`，调用点写 `condition<Role>(dynamicTree)` 而不是 `as any`。

互斥 FK 清空：

```ts
export function clearExclusive<T>(row: T, keys: Array<keyof T>): void {
  for (const k of keys) (row as Record<keyof T, unknown>)[k] = null;
}
```

禁止 `this.MetaServiceId = null as any`。

---

## 5. 删除清单（硬切，同 PR 改调用点）

| 删除 | 理由 |
| --- | --- |
| `this: { new (...args: any[]): T } & typeof BaseModel` | HC1/HC2 |
| `options?: any`（CRUD override） | 用 `UpdateOptions` / `SearchOptions<InstanceType<C>>` |
| `Partial<Insertable<T & BaseModel>>` | HC3 |
| `createServiceByModel(name): UntypedModelService` | HC6 |
| `static dial<T = Record<string, …>>` 的默认 `T` | HC6 |
| `export type Entity` / `Queryable` 兼容别名 | 改用 `SelectResult` / `Selectable` |
| `declare const $choysum: unknown`（source.ts） | HC9 |
| PropertyDefinition / TranslationTerm / FieldDefault 基类上「只为过类型」的 `as any` 转发 | 换成 HC2 后应直接 `super.Create(value, returnFields)` |
| 生产代码新增 `as any` / `: any` / `Record<string, any>` | HC10 / HC12 |

**不删（相对旧稿）：** `AuthzMutationModel` 的 Create/Update/Delete override、FieldChange/Message/规则模型上的 stamp / append-only / `_prepareValues` override——它们是合法扩展点；BT-2 只要求 HC2 签名。

---

## 6. 作者合同（override 还剩什么）

| 允许 | 禁止 |
| --- | --- |
| 领域写变换：`Role.Create` 同步 UI grant，签名 = HC2，payload 为 `Insertable<Role>` | CRUD override 里 `value as any` / `returnFields as any` / `this: { new (...args: any[]) }` |
| 平台不变式：append-only / stamp / prepare / authz invalidate 的 CRUD override（HC2） | 为「只改 this 形状」复制一整份 CRUD（应用 HC2 多态，而不是再抄一份 any this） |
| mixin 增加**新**静态方法（`SearchByRecord`）或 HC2 包装 CRUD（如 `AuthzMutationModel`） | `_prepareValues(value as any, …)`（应用 `Record` / `Partial<Insertable<…>>`） |
| 测试里 `as unknown as Role` + 一行原因 | 测试里 `} as any, ['Id'] as any` 连打（新代码） |

`PolymorphicRecordModel.SearchByRecord`：`this: ModelCtor`，内部 `this.Search({ And: [['Model','=',m], ['ResId','=',id]] }, { fields, orderBy })`，删 `(this as any)` 与 `FieldSelection<any>`。

---

## 7. 分期落地（BT-0 … BT-7）

原则：**先改类型力学，再收投影与边界。** 一个 BT 可拆多 PR；单 PR 不与无关功能混车；硬切 = 无双签名。  
**不**再把「迁 write policy / 删平台 CRUD override」当作必经波次（原 BT-3 已取消）。

```text
BT-0  ctor SSOT
  └─ BT-1  集合入口 HC2（返回仍满 InstanceType<C>）
        ├─ BT-2  生产 override 对齐签名          ──┐
        ├─ BT-5  ModelService / dial 绑 InstanceType ┘ 可并行
        └─ BT-4  Projected 重载
              └─ BT-6  unknown 边界
                    └─ BT-7  eslint / 计数 / 业务残留
```

**拆 PR 规则：** 编译失败必须同 PR 改调用点；typecheck 守卫文件（`*.typecheck.ts`）与实现同 PR。行为语义（invalidate 时机等）若改，单独说明；**不**为「搬到 metadata」单独开硬切 PR。

每 PR 描述回链 **HC\* + PR-BT-n**，并贴 `as any` 计数（脚本在 BT-7 才正式入库；此前可用 §3 同口径临时命令）。

---

### 7.0 公共命令（每个 PR 都跑）

```bash
./choysum test typecheck core
# 另加本 PR 触及的 App，见各 PR「测试」表
go fmt ./...   # 若动到 Go；纯 TS PR 可省略
```

临时计数（BT-7 前）：

```bash
rg -g '!*.test.ts' -g '!**/tests/**' --count-matches '\bas any\b' modules/*/service
```

---

### 7.1 BT-0 — ctor 收口（1 PR）

**目标：** 只留 `ModelClass<T>` / `ModelCtor<T>`。删除 `BaseModelCtor` / `InstantiableModelCtor` / `RuntimeModelCtor`。**不改** Create/Search 的方法级 `T`、不改运行时。

#### PR-BT-0 `refactor(core): unify ModelCtor types`

| 项 | 内容 |
| --- | --- |
| 行为 | 无（纯类型重命名 + 参数类型并集） |
| 风险 | `ModelCtor = new (...args: never[])` 与 `ModelClass = abstract new (symbol, entity, fields?)` 不完全等价，需让 metadata 接受 `ModelCtor` |

**定义（改）：**

| 文件 | 函数 / 类型 |
| --- | --- |
| `modules/core/service/orm/model/types.ts` | 删除 `BaseModelCtor` / `RuntimeModelCtor` / `InstantiableModelCtor`；新增 `ModelFactoryArgs` / `ModelClass` / `ModelCtor`（§4.1） |
| `modules/core/service/orm/model/index.ts` | re-export 改为 `ModelCtor` / `ModelClass` |
| `modules/core/service/index.ts` | `export type { BaseModelCtor }` → `ModelCtor` |
| `modules/core/service/orm/metadata/field.ts` | `ModelCtor<T>` **改为** re-export `orm/model/types.ts` 的 `ModelCtor`（禁止第三套定义） |
| `modules/core/service/api/metadata.ts` | 若仍导出 `ModelCtor`，与上对齐 |

**消费方机械替换（生产，按目录）：**

- **facade：** `model_create_service_facade.ts` 的 `createModel` / `createManyModels`；`model_update_service_facade.ts` 的 `updateModels` / `updateModelById`；`model_delete_service_facade.ts`；`model_read_facade.ts` 的 `browseModel` / `browseManyModels` / `searchModels` / `countModels` / `readGroupedModels` / `countGroupedModels`；`model_runtime_service_facade.ts`；`model_edge_facade.ts`；`model_internal_facade.ts` 的 `hydrateModel` / `getModelRepository`。本地 alias（`ModelCreateServiceFacadeCtor` 等）一律改 `ModelCtor<T>`。
- **operations：** `model_create.ts` `CreateOperations.Create` / `CreateMany`；`model_update.ts` `UpdateOperations.*`；`model_delete.ts` `DeleteOperations.*`；`model_read.ts` `ReadOperations.*`；`model_instance.ts`；`model_copy.ts` `copyModel`；`model_default.ts` / `model_default_get_pipeline.ts` `runDefaultGetPipeline`；`model_namesearch.ts` / `model_namecreate.ts`；`model_fields_get_facade.ts`；`model_field_translations.ts`；`model_field_company_values.ts`；`model_onchange*.ts`；`model_hydration.ts`；`model_runtime.ts`；`model_registry.ts`；`model_ctor_lookup.ts`；`model_for_field_condition.ts`；`model_soft_delete_scope.ts`；`field_tracking.ts`。
- **平台基类（只改 ctor 参数，不改 CRUD override 签名）：** `field_default_base_model.ts` 的 `storeMeta` / `ensureScopeUniqueIndex` / `Set` / `Get` / `GetEffective` / `Unset`；`app_setting_base_model.ts` 的 `storeMeta` / `Get` / `Set`；`translation_term_base_model.ts` 的 `storeMeta` / `ensureTermUniqueIndex` / `GetTranslations` / `ImportPackaged`；`property_definition_base_model.ts` 的 `storeMeta` / `ensureDefinitionUniqueIndex` / `assertUniqueDefinitionScope`；`properties_lookup.ts` / `field_default_lookup.ts`；`properties_resolve.ts` `resolveProperties`；`properties_write.ts`；`properties_definition_acl.ts` `assertPropertyDefinitionParentWritable`；`properties_definition_purge.ts`。
- **decorator / metadata：** `decorator/model.ts` `Model` / `RegisteredModelCtor` / `ApplicationModelPool`；`decorator/field.ts`；`decorator/service.ts`；`decorator/{compute,constraint,search,inverse,sqlcompute}.ts`；`metadata/storage.ts` `getModelMetadata` / `getEffectiveConstraints` / `getEffectiveOnchange`；`metadata/constraint.ts`。
- **relation / repository / runtime / converter：** `relation/{types,factory,processor,many-to-one,many-to-many,one-to-many,relation_model_service_facade}.ts`；`repository/{repository_factory,repository_runtime_bridge}.ts`；`repository/authz/field_rule_helpers.ts`；`repository/validation/bridge.ts`；`repository/query/{condition_compiler,select_context}.ts`；`repository/projection/relation_projection.ts`；`repository/read/read_aggregate_helpers.ts`；`runtime/{compute/engine,compute/graph,compute/cascade,validation/engine,proxy/proxy,runtime_repository_facade}.ts`；`runtime/onchange/plan/{index,executor,preload,cache,shared}.ts`；`orm/utils/converter.ts` `EntityConverter.*`。
- **跨模块 import：** `message/service/models/message.ts`；`audit/service/models/field_change.ts`；`document/service/models/_binding_field_limits.ts`；`auth/service/models/user/_lifecycle_auth.ts`；`auth/service/models/app_setting.ts`。

`model.ts` 本 PR **只改 import 名**：`this: BaseModelCtor<T>` 暂改 `this: ModelClass<T>`（仅工厂构造签名，避免现有 override 与 `typeof BaseModel` 冲突）；facade / metadata / hydrate 用 `ModelCtor<T>`。**保留**方法级 `T extends BaseModel` 与 `this as unknown as ModelCtor<T>` 委托（BT-1 再删）。

**测试（改 import / 桩 ctor，无新用例）：**

| 文件 | 注意 |
| --- | --- |
| `model_ctor_lookup.test.ts` | ctor 查找 |
| `model_create_service_facade.test.ts` / `model_update_service_facade.test.ts` / `model_delete_service_facade.test.ts` / `model_read_facade.test.ts` / `model_runtime_service_facade.test.ts` | facade 桩 |
| `model.test.ts` / `model_read.test.ts` / `model_default.test.ts` / `model_copy.test.ts` |  |
| `field_default_base_model.test.ts` / `app_setting_base_model.test.ts` | `InstantiableModelCtor` 参数 |
| `decorator/service.test.ts` |  |
| 关系测试：`relation/{factory,many-to-one,one-to-many,many-to-many,processor,relation_model_service_facade}.test.ts` | 测试里 override 的 `this` **先不动**（BT-2） |

**跑：** `./choysum test typecheck core`；`./choysum test unit core --be`。

**DoD：**

```bash
rg -n 'BaseModelCtor|InstantiableModelCtor|RuntimeModelCtor|ModelStatic' modules --glob '!*.md'
# 仅允许：本设计文档
```

---

### 7.2 BT-1 — 集合入口多态 this（1 PR）

**目标：** HC2 + HC3。`Create<C extends ModelCtor>(this: C, …): Promise<InstanceType<C>>`。返回 **先仍满行**（Projected 留给 BT-4）。废除 `Insertable<T & BaseModel>`。

**依赖：** PR-BT-0。

#### PR-BT-1 `refactor(core): polymorphic ModelCtor collection APIs`

**`model.ts` 静态方法全部改签名**（删除方法级 `T extends BaseModel`）：

| 方法 | 今日 | 目标（BT-1） |
| --- | --- | --- |
| `DefaultGet` | `this: BaseModelCtor<T>`, `Partial<Insertable<T & BaseModel>>` | `this: C`, `Partial<Insertable<InstanceType<C>>>` |
| `FieldsGet` | 同上 | `this: C` |
| `GetFieldTranslations` / `UpdateFieldTranslations` | 同上 | `this: C` |
| `GetFieldCompanyValues` / `UpdateFieldCompanyValues` | 同上 | `this: C` |
| `ResolveProperties` | 同上 | `this: C` |
| `Copy` / `NameSearch` / `NameCreate` | 同上 | `this: C` |
| `Create` / `CreateMany` | `Partial<Insertable<T & BaseModel>>` → `Promise<T>` | `Partial<Insertable<InstanceType<C>>>` → `Promise<InstanceType<C>>` |
| `Browse` | `FieldSelection<T>` → `T` | `FieldSelection<InstanceType<C>>` → `InstanceType<C>` |
| `BrowseMany` | `(keyof Selectable<T>)[]` | **改为** `FieldSelection<InstanceType<C>>` |
| `Search` / `Count` | `QueryCondition<T>` | `QueryCondition<InstanceType<C>>` |
| `ReadGroup` / `ReadGroupCount` | `GroupBySpec<T>` | `GroupBySpec<InstanceType<C>>` |
| `Update` / `UpdateById` | `Partial<Updateable<T & BaseModel>>` | `Partial<Updateable<InstanceType<C>>>` |
| `Delete` / `DeleteById` | `QueryCondition<T>` | `QueryCondition<InstanceType<C>>` |
| `Onchange` / `withSavepoint` / `hydrate` | `this: BaseModelCtor<T>` | `this: C` |
| `getEffectiveConstraints` / `getEffectiveOnchange` | `this: { new (...args: any[]): T }` | `this: ModelCtor`；内部 `getEffectiveConstraints(this)` **无** `as any` |
| 实例 `copy` | `this.constructor as RuntimeModelCtor<this>` | `this.constructor as ModelCtor<this>` |

委托：`return createModel(this, value, returnFields)` —— **禁止** `this as unknown as ModelCtor<T>`。若 facade 参数已是 `ModelCtor<T>`，`this` 应直接可传。

**facade / operations 载荷：**

| 文件 | 函数 | 改 |
| --- | --- | --- |
| `model_create_service_facade.ts` | `createModel` / `createManyModels` | 参数 `ModelCtor<T>`；`Partial<Insertable<T>>`（删 `& BaseModel`）；删 `as T` 若可推断 |
| `model_update_service_facade.ts` | `updateModels` / `updateModelById` | 删 `& BaseModel` |
| `model_read_facade.ts` | `browseManyModels` | `fields?: FieldSelection<T>`；删 `{ fields } as SearchOptions<T>` |
| `model_create.ts` | `CreateOperations.Create` / `CreateMany` | ctor 已是 `ModelCtor`（BT-0）；确认载荷无 `& BaseModel` |
| `model_default_get_pipeline.ts` | `runDefaultGetPipeline` | 入参与 `DefaultGet` 对齐 |

**新增编译守卫：**

| 文件 | 断言 |
| --- | --- |
| `orm/repository/types/insertable_nullability.typecheck.ts` | **保持**可空标量仍 Insertable（回归） |
| **新建** `orm/model/model_static_this.typecheck.ts` | 定义 `class Probe extends BaseModel { Name!: string }`；`Probe.Create({ Name: 'x' })` 推断为 `Probe`；`Probe.Search(['Name','=','x'])` 合法；`Probe.Search(['NoSuch','=',1])` 应能写成负例（`// @ts-expect-error`） |

**测试：**

| 跑 | 覆盖 |
| --- | --- |
| `./choysum test typecheck core` | 守卫文件编进 typecheck |
| `./choysum test unit core --be` | facade / model CRUD |
| 重点文件：`model_create_service_facade.test.ts`、`model_update_service_facade.test.ts`、`model_read_facade.test.ts`、`model_read.test.ts`、`model.test.ts`、`model_default.test.ts`、`model_namecreate.test.ts`、`model_namesearch.test.ts` | 桩上的 `this: { new (...args: any[]) }` **本 PR 改成 `ModelCtor`**（测试桩也硬切） |

**DoD：**

```bash
rg -n 'T & BaseModel' modules/core/service/orm
rg -n 'as unknown as ModelCtor|as unknown as RuntimeModelCtor' modules/core/service/orm/model/model.ts
rg -n 'new \(\.\.\.args: any\[\]\)' modules/core/service/orm/model/model.ts
```

三处均为 0。`Role.Create` 此时**还不必**零断言（override 仍是旧 `this`，BT-2 处理）。

**BT-1 spike（可做在 PR 内第一 commit）：** 临时把 `Role.Create` 的 `this` 改成 `C extends ModelCtor`，确认 `super.Create(value, returnFields)` 通过；再还原 Role，只合 core。失败则停在 BT-0/BT-1，不铺 override。

---

### 7.3 BT-2 — 生产 override 对齐 HC2（3 PR）

**目标：** 所有生产 CRUD override 使用 HC2 签名；`super.*(value, returnFields)` 零 `as any`。**保留** cache invalidate / append-only / stamp / prepare 的 override 语义（不迁 policy）。

**依赖：** PR-BT-1。测试里同类 override 一并改，避免 `model_namesearch.test.ts` 等继续教错误模板。

#### PR-BT-2a `refactor(core): HC2 overrides for platform base models`

| 文件 | 函数 | 做法 |
| --- | --- | --- |
| `property_definition_base_model.ts` | `Create` / `CreateMany` / `Update` / `UpdateById` / `Delete` / `DeleteById` | `this: C extends ModelCtor`；`super.Create(value, returnFields)`；内部 `Search` 用 `this.Search` 而非 `(this as any).Search`；`assertUniqueDefinitionScope(this, rec)` 的 ctor 参数已是 `ModelCtor` |
| 同上 | `storeMeta` / `ensureDefinitionUniqueIndex` / `assertUniqueDefinitionScope` | 已在 BT-0 改 ctor；本 PR 删残留 `as any` |
| `translation_term_base_model.ts` | 同上六套 CRUD | 同 PropertyDefinition |
| `field_default_base_model.ts` | **无 CRUD override**；`Set` / `Get` / `GetEffective` / `Unset` | `this: ModelCtor<FieldDefaultBaseModel>`；内部 `Search`/`UpdateById`/`Create` 去 `(this as any)` / `{ Value } as any` |
| `app_setting_base_model.ts` | `Get` / `Set` | 同 FieldDefault |
| `mixins/polymorphic_record_model.ts` | `SearchByRecord` | `this: ModelCtor`；`fields?: FieldSelection<InstanceType<typeof this>>`；内部 `this.Search({ And: [['Model','=',m],['ResId','=',id]] }, { fields, orderBy: { field: this.polymorphicOrderByField(), order: 'asc' } })`；删 `FieldSelection<any>` / `(this as any).Search` |

**测试：**

| 文件 | 断言 |
| --- | --- |
| `property_definition_base_model` 若无独立 test | 走 `properties_coverage.test.ts` / properties ACL 相关 |
| `translation_term_base_model_coverage.test.ts` / `translation_term_cache.test.ts` | CRUD + unique index |
| `field_default_base_model.test.ts` / `field_default_base_model_coverage.test.ts` / `field_default_pipeline_wire.test.ts` / `field_default_lookup.test.ts` | Set/Get/Unset |
| `app_setting_base_model.test.ts` / `app_setting_base_model_coverage.test.ts` | Get/Set |
| `message` / `audit` 的 `SearchByRecord` 测试 | 签名变化后仍能搜 |

**跑：** `typecheck core`；`unit core --be`；`typecheck message` + `typecheck audit`（PolymorphicRecordModel 子类）。

#### PR-BT-2b `refactor(auth): HC2 Role* and AuthzMutationModel signatures`

**mixin（签名对齐，行为仍 wrap invalidate）：**

| 文件 | 函数 |
| --- | --- |
| `auth/service/mixins/authz_mutation_model.ts` | `Create` / `CreateMany` / `Update` / `UpdateById` / `Delete` / `DeleteById`：`this: C extends ModelCtor`；`super.Create(value, returnFields)` **无** `as any`；删 `options?: any` |
| 同上 | `mutateThenInvalidateAllAuthzCaches` / `mutateThenInvalidateAuthzCachesForUsers` / `userIdsFromUserRolePayloads`：后者入参 `unknown` → `Partial<Insertable<UserRole>> \| Array<Partial<Insertable<UserRole>>>`（UserRole 循环 import 则放 `_request_cache_invalidation.ts` 旁、用最小 `{ UserId?: IdRelationItem }`） |

**Role 领域变换（仍 override，payload 改 typed）：**

| 文件 | 函数 |
| --- | --- |
| `models/role.ts` | `Browse` / `BrowseMany` / `Search` / `Create` / `CreateMany` / `Update` / `UpdateById` |
| `models/_role_ui_projection.ts` | `applyAccessWriteTransformOnCreate(values: Partial<Insertable<Role>>)`；`applyAccessWriteTransformOnUpdate`；`syncAllowResourceGrants`；`hydrateAccessUiResourceIds(records: Array<Partial<Role> \| Role>)`；`wantsAccessField(selection: FieldSelection<Role> \| undefined)` |
| `models/user_role.ts` | `Create` / `CreateMany`（仍 targeted invalidate；签名 HC2） |

**规则模型 `_prepareValues` 入参收口（语义留在 override 里，不迁 metadata policy）：**

| 文件 | 函数 |
| --- | --- |
| `role_field_rule.ts` | `_validatePerms` / `_prepareValues` / `Create` / `CreateMany` / `Update` / `UpdateById` |
| `role_record_rule.ts` | `_prepareValues` + 四套写 |
| `role_method_access.ts` | `_prepareValues` + 四套写 |
| `role_ui_resource.ts` | `_prepareValues` + 四套写 |
| `_rule_scope_helpers.ts` | `assertExclusiveScope(values: Record<string, unknown>, …)` |

`_prepareValues` 目标类型：`Partial<Insertable<Concrete>>`（create）/ `Partial<Updateable<Concrete>>`（update），禁止 `Record<string, any>`。

**测试：**

| 文件 | 覆盖 |
| --- | --- |
| `authz_mutation_crud_coverage.test.ts` | mixin CRUD + cache |
| `authz_mutation_helpers.test.ts` | `userIdsFromUserRolePayloads` |
| `role_access_ui_resource_ids.test.ts` / `role_ui_projection_hydrate.test.ts` / `role_ui_resource_sync.test.ts` | Role grant 同步 |
| `field_rule.test.ts` / `record_rule.test.ts` / `permission_state.test.ts` | 规则写入仍合法 |

**跑：** `./choysum test typecheck auth`；`./choysum test unit auth --be`。

#### PR-BT-2c `refactor: HC2 leftover overrides (meta/message/audit/tests)`

| 文件 | 函数 | 做法 |
| --- | --- | --- |
| `meta/service/models/module_index.ts` | `Search`（约 L183）及邻近 `Search`/`readGroup` 辅助 | `this: C extends ModelCtor`；`condition: QueryCondition<InstanceType<C>> \| []`（今日 `any[] \| Record<string, any>` 删掉）；`options?: SearchOptions<InstanceType<C>>`；`getModelRepository(this)` 无 `as any`。动态拼树走 §4.6 `condition<MetaModuleIndex>(…)` |
| `meta/service/models/_module_index_query.ts` | `assertSearchCondition` / `normalizeFields` / `parseSortSpecs` / `buildSortPushdownPlan` / `extractGroupedModuleNames` | 入参从 `any` 收到 `unknown` / `FieldSelection` / `OrderBy` |
| `message/service/models/message.ts` | `Create` / `CreateMany` | `this: C`；`prepareCreatePayload` 入参 `Partial<Insertable<Message>>`；`super.Create.call(this, payload, returnFields)` 无 `as T & BaseModel` |
| `audit/service/models/field_change.ts` | `Create` / `CreateMany` / `Update*` / `Delete*` | 仅对齐 HC2；**保留** append-only / stamp override |
| 测试桩：`orm/relation/{many-to-one,one-to-many,many-to-many,processor,relation_model_service_facade}.test.ts`；`model_namesearch.test.ts`；`model_namecreate.test.ts` | 测试 override `Create`/`UpdateById` | 换成 `ModelCtor`，禁止再复制 `{ new (...args: any[]) }` |

**测试：** `typecheck meta message audit`；`unit meta --be`；`unit message --be`；`unit audit --be`；`meta/service/tests/module_index.test.ts`；`audit/service/tests/field_change.test.ts`。

**BT-2 总 DoD：**

```bash
rg -g '!*.test.ts' -g '!**/tests/**' -n 'new \(\.\.\.args: any\[\]\)' modules/*/service
rg -g '!*.test.ts' -g '!**/tests/**' -n 'options\?: any' modules/auth/service modules/core/service/orm/model modules/meta/service/models
```

生产命中 0。Role\* `super.Create(` 行不得再含 `as any`。

---

### 7.4 BT-3 — ~~policy pipeline~~（**已取消**）

**状态：** 取消。未合入 main；[#408](https://github.com/choysum-dev/choysum/pull/408) 已关闭。

**取消理由（相对旧 HC4）：**

1. **IMD：** 下游可 override CRUD，也可改写 `@Model`；metadata policy **不是**不可绕过约束。  
2. **必要性：** 类型硬切（HC1–HC3、HC2 签名）不依赖「消灭平台 CRUD override」；BT-0～2 已覆盖力学问题。  
3. **合理性：** append-only / stamp / prepare / authz invalidate 写在模型 override 上是 IMD 下的正常扩展点；强行迁引擎只增加间接层与未定义的继承合并语义。

**保留做法：** §4.5；AuthzMutationModel / FieldChange / Message / 规则模型继续用 **HC2** CRUD override。

**编号：** BT-3 空号，后续排期直接 **BT-2 → BT-4 / BT-5**，不再插入 3a–3d。

---

### 7.5 BT-4 — 诚实投影（1 PR）

**目标：** HC7。有 `fields`/`returnFields` 时返回 `Projected<T,F>`。

**依赖：** PR-BT-1（重载建在正确的 `this` 上）。建议 **BT-2 之后**再合，减少 override 与重载打架。

#### PR-BT-4 `feat(core): Projected return types for Search/Browse/Create`

| 文件 | 改 |
| --- | --- |
| `orm/repository/types/selection.ts` | 新增 `Projected<T, F>`（§4.3）；可选 `fields<T>()(...names)` helper 保证字面量 |
| `orm/repository/types/query.ts` | `OrderBy<T>` 的 `field` 改为 `Extract<keyof Selectable<T>, string>` |
| `service/api/selection.ts` | re-export `Projected` |
| `model.ts` | `Create`/`CreateMany`/`Browse`/`BrowseMany`/`Search`/`Update`/`UpdateById` 按 §4.2 **两组重载**（无 fields → 满行；有 fields → Projected） |
| `model_read_facade.ts` / `model_create_service_facade.ts` / `model_update_service_facade.ts` | 返回类型跟上；实现可用重载或单一实现 + 断言在 facade **内部一次** |

**调用点试点（必须改，用来证明投影）：**

| 文件 | 函数 | 今日 | 目标 |
| --- | --- | --- | --- |
| `auth/service/models/user/_method_access.ts` | `metaModelId` / `metaApplicationId` | `Search(..., { fields: ['Id'] } as any)` 然后 `rows?.[0]?.Id` | 去掉 `as any`；结果类型有 `Id` 无 `Name` |
| `auth/service/models/user/_permission_state_acl.ts` | 同类 `fields: ['Id','Name']` 汇总 | `(r as any).Name` | `r.Name` 合法 |
| `auth/service/models/user/_record_rule_eval.ts` | 动态 And/Or | `{ And: parts } as any` | `condition<…>(…)`（若 BT-6 未合，本 PR 可先内联一个本地 helper，BT-6 再升格） |

**新增守卫：** `orm/repository/types/projected.typecheck.ts`

```ts
type Row = Projected<Probe, ['Id']>;
type ExpectId = ExpectTrue<'Id' extends keyof Row ? true : false>;
// @ts-expect-error Name was not selected
type NoName = Row['Name'];
```

**测试：** 现有 Search 单测行为不变（运行时仍返回所选列）。`typecheck --all`（重载影响所有 App）。

**DoD：** `projected.typecheck.ts` 进入 typecheck；试点三文件对应 Search 调用无 `as any`；访问未选字段在守卫里 `@ts-expect-error`。

---

### 7.6 BT-5 — ModelService / dial（2 PR）

**目标：** HC6。CRUD 不靠 `Parameters<泛型函数>`。

**依赖：** PR-BT-1。与 BT-2 **可并行**；若并行，call site PR 等 BT-2 后再清 `as any` 更干净。

#### PR-BT-5a `refactor(core): bind ModelService CRUD to InstanceType`

| 文件 | 函数 / 类型 |
| --- | --- |
| `modules/core/rpc/types/model.ts` | `ModelService<C>` = `CrudService<C> & 自定义方法 mapped`（§4.4）；`ModelConstructor` 改为 `ModelCtor` 或删除 |
| `service/rpc/service_factory.ts` | **删除** `createServiceByModel(name): UntypedModelService`；只保留 `createServiceByModel<C extends ModelCtor>(name: string): ModelService<C>` |
| `orm/model/model_pool.ts` | `dial<C extends ModelCtor>(fullModelName: string): ModelService<C>`；删除默认 `T = Record<string, …>`；`pool` 的默认 `T = typeof BaseModel` 改为强制泛型 `pool<C extends ModelCtor>(app, short): C` |
| `model.ts` `static dial` | 转调 `dial`，同样强制泛型 |
| `model_create.ts` / `model_update.ts` / `model_read.ts` | `createServiceByModel('document.AttachmentBinding') as unknown as AttachmentBindingServiceLike` → `createServiceByModel<typeof AttachmentBinding>(...)`（core 若不能 import document 模型，保留窄 `AttachmentBindingServiceLike` **显式 interface**，禁止 `unknown as` 扩成整袋 any） |

**测试：** `service/rpc/service_factory.test.ts` —— 调用改为 `createServiceByModel<typeof FakeCtor>(name)`；缺工厂仍 throw。`app_setting_base_model.test.ts` 里 `dial(modelName)` 补泛型。

**跑：** `typecheck core`；`unit core --be`。此时 **业务 App 会红**（无泛型调用），必须紧接 5b 或同一波合并。

#### PR-BT-5b `refactor: require createServiceByModel type arguments`

生产调用点（均已有 `<typeof X>` 的保持；**无泛型的补上**）：

| 文件 | 符号 |
| --- | --- |
| `auth/.../user/_method_access.ts` | `MetaApplication` / `MetaModel` / `MetaService` / `MetaUiResource`；`metaModelId` / `metaApplicationId` / `resolveMethodAccessMeta` 的 `Search` 去 `as any` |
| `auth/.../user/_permission_state_acl.ts` / `_permission_state_ui.ts` | 同上 |
| `auth/.../user/_field_rule_eval.ts` / `_record_rule_eval.ts` | 同上 |
| `auth/.../user/_lifecycle_auth.ts` / `user.ts` | `CompanyService` / `Language` |
| `meta/.../module.ts` / `module_index.ts` | `Job` |
| `partner_bank/.../bank_account.ts` | `Bank` |
| `document/service/hook/post_init.ts` / `meta/service/hook/post_init.ts` | `ScheduleService` |
| 全部 `createServiceByModel('…')` 无 `<…>` 的测试 | 补 `typeof` |

**测试：** `method_access_meta_lookup.test.ts`；`permission_state_acl_source.test.ts`；`field_rule_effective_resolve.test.ts`；`record_rule_eval_edges.test.ts`。

**跑：** `typecheck auth meta document partner_bank`；`unit auth --be`。

**DoD：**

```bash
rg -n 'createServiceByModel\([^<]' modules --glob '!*.md'
# 实现过载已无无泛型签名；调用处必须带类型实参（或从 typed 别名转调）
rg -n 'dial<T = Record' modules/core
# 0
```

`_method_access.ts` 的 `Search({ And: [['Application', '=', …]] } as any)` 为 0。

---

### 7.7 BT-6 — unknown 边界（3 PR）

**依赖：** BT-0（`$choysum.db` 类型已可用）；BT-2a（平台基类不再靠 `as any` 调 db 更干净，可并行）。

#### PR-BT-6a `fix(core): single $choysum declaration`

| 文件 | 改 |
| --- | --- |
| `modules/core/types/$choysum.d.ts` | 保持 `declare var $choysum`；补 `interface GlobalThis { $choysum?: typeof $choysum }`（若需要 `globalThis.$choysum`） |
| `runtime/context/source.ts` | **删除** `declare const $choysum: unknown \| undefined`；`resolveRuntimeRoot` 直接用 `$choysum` / `globalThis.$choysum` |
| `field_default_base_model.ts` `ensureScopeUniqueIndex` | `$choysum.db.dialectName` / `$choysum.db.execute` 无 `as any` |
| `property_definition_base_model.ts` `ensureDefinitionUniqueIndex` | 同上 |
| `translation_term_base_model.ts` `ensureTermUniqueIndex` | 同上 |

**测试：** 上述 coverage tests；`typecheck core`。

**DoD：** `rg -g '!*.test.ts' -n '\$choysum as any' modules` = 0。

#### PR-BT-6b `refactor(core): RefLike guards for normalizeRefId`

| 文件 | 函数 |
| --- | --- |
| `core/service/utils/normalization.ts` | 新增 `isRefLike`；改 `readRefId` / `normalizeRefId` / `normalizeRefIdList`：禁止 `(value as any).Id` |
| **新建** `utils/normalization_ref.test.ts` 或扩展现有 normalization 测试 | string / `{Id}` / `{id}` / null / 无 Id 对象 |

被大量引用，行为必须与今日一致（trim、空串 → null、单元素升数组）。

**跑：** `typecheck core`；`unit core --be`；`unit auth --be`（UserId 规范化）。

#### PR-BT-6c `refactor(auth): clearExclusive and condition() helper`

| 文件 | 函数 |
| --- | --- |
| **新建** `orm/repository/types/query.ts` 旁或 `utils/condition.ts` | `condition<T>(tree: BaseQueryCondition): QueryCondition<T>`（§4.6）；从 `api/query` re-export |
| **新建** `orm/model/clear_exclusive.ts` | `clearExclusive<T>(row: T, keys: Array<keyof T>): void` |
| `role_method_access.ts` | `this.MetaServiceId = null as any` 等 → `clearExclusive(this, ['MetaServiceId','MetaModelId','MetaApplicationId', …])` |
| `_record_rule_eval.ts` / `_method_access.ts` | `{ And: parts } as any` → `condition<…>(…)` |

**测试：** 规则互斥字段清空的现有 onchange/unit；`typecheck auth`。

**DoD：**

```bash
rg -g '!*.test.ts' -g '!**/tests/**' -n 'null as any' modules/*/service
# 0，或 PR 内例外表 ≤ 2 行并注明原因
```

---

### 7.8 BT-7 — 门禁与残留清扫（4+ PR）

**依赖：** BT-2～BT-6 主体已合。本阶段按模块清剩余生产 `as any`，再开 eslint。

#### PR-BT-7a `chore: as-any inventory script`

| 文件 | 内容 |
| --- | --- |
| **新建** `scripts/dev/count_as_any.py` | 口径 = §3（`modules/*/service/**/*.ts`；生产排除 `*.test.ts` / `**/tests/**`）；输出：模块合计、Top 文件、粗分桶（`super.` / `Search(` / `$choysum` / `as any`） |
| 文档 | 本文件 §3 注明「以脚本为准」 |

**不做：** CI hard fail。

**测：** `python3 scripts/dev/count_as_any.py` 退出 0；数字与人工抽查同量级。

#### PR-BT-7b `refactor(auth): remaining eval/user bags`

热点（§3 Top）：`user/user.ts`、`user/_method_access.ts`、`user/_permission_state_acl.ts`、`user/_permission_state_ui.ts`、`user/_authz_context.ts`、`user/_record_rule_eval.ts`、`user/_field_rule_eval.ts`、`user/_lifecycle_auth.ts`。

按函数收：`GetPermissionState`、`CheckMethodAccess`、`evaluateRecordRuleCondition`、`evaluateFieldRules`、`buildAuthzContext` —— 行类型用 BT-4 投影或显式 DTO（`type MetaNameRow = { Id: string; Name: string }`）。

**测试：** `permission_state.test.ts`；`check_method_access_*.test.ts`；`field_rule.test.ts`；`record_rule.test.ts`。

#### PR-BT-7c `refactor: remaining app service bags`

按模块可再拆 PR，每 PR 不跨 App：

| 建议 PR | 文件 |
| --- | --- |
| document | `attachment_binding.ts`、`attachment_object.ts`、`_attachment_gc.ts`、`_upload.ts`、`upload_session.ts` |
| meta | `ui_resource.ts`、`module.ts`、`module_index.ts`、`_module_index_query.ts` |
| base | `_sequence_next.ts`、`_currency_convert.ts`、`language.ts`、`company.ts`、`city.ts` |
| task | `schedule.ts`、`data_transfer_job_queue.ts`、`job.ts` |

**测试：** 各模块 `test unit <app> --be` + `typecheck <app>`。

#### PR-BT-7d `chore: no-explicit-any on production service + testkit`

| 文件 | 改 |
| --- | --- |
| 仓库 TS eslint 配置（若尚无则新增仅覆盖 `modules/*/service` 的规则文件） | 生产 glob 启用 `@typescript-eslint/no-explicit-any` error；`**/*.test.ts`、`**/tests/**` 关闭 |
| `modules/core/service/testing.ts` 或 `testing/fixtures.ts` | `fakeCtx(partial)`、`fakeRow<T>(partial: Partial<T>): T`（`as unknown as T` **集中一处**）、`asChoysumError(err: unknown)` |
| 测试新代码 | 禁止 `} as any, ['Id'] as any` 连打；改走 testkit |

**跑：** 全模块 `typecheck --all`；抽样 `unit auth --be`。

**BT-7 总 DoD：** 脚本生产 `as any` 相对 2026-09-16 **1080 下降 ≥ 70%**；生产 eslint any 为 error；测试不设归零。

---

### 7.9 PR 一览（排期）

| PR | 标题（建议） | 可并行 | 必跑 typecheck | 必跑 unit |
| --- | --- | --- | --- | --- |
| BT-0 | unify ModelCtor types | — | core | core `--be` |
| BT-1 | polymorphic collection APIs | 后于 0 | core | core `--be` |
| BT-2a | platform base HC2 overrides | 后于 1 | core, message, audit | core / message / audit `--be` |
| BT-2b | Role\* HC2 signatures | 后于 1；可与 2a 并行 | auth | auth `--be` |
| BT-2c | meta/message/audit leftover + test stubs | 后于 1 | meta, message, audit | 同左 `--be` |
| ~~BT-3a–3d~~ | ~~write policy / 删平台 CRUD override~~ | **已取消**（见 §7.4） | — | — |
| BT-4 | Projected returns | 后于 1，建议后于 2 | `--all` | core `--be` + auth Search 试点 |
| BT-5a | ModelService InstanceType | 后于 1；**紧接 5b** | core | core `--be` |
| BT-5b | createServiceByModel type args | 后于 5a | auth, meta, document, partner_bank | auth `--be` |
| BT-6a | single `$choysum` | 可与 2+ 并行 | core | core `--be` |
| BT-6b | RefLike / normalizeRefId | 可并行 | core | core + auth `--be` |
| BT-6c | clearExclusive + `condition()` | 建议后于 2b | auth | auth `--be` |
| BT-7a | count script | 随时 | — | 脚本自身 |
| BT-7b/c/d | 残留 + eslint + testkit | 后于 2–6 | 触及 App / `--all` | 触及 App `--be` |

推荐波次：`BT-0 → BT-1 → (2a∥2b∥2c∥5a) → 5b → BT-4 → (6a∥6b∥6c) → BT-7*`（**跳过**已取消的 BT-3）。

---

## 8. 验收门禁

每个 BT PR：

| 门禁 | 要求 |
| --- | --- |
| Typecheck | `./choysum test typecheck core` + 本 PR 触及的 App |
| Unit | 触及包 `--be` |
| 计数 | 运行与 §3 同口径脚本，PR 写清生产 `as any` 前后 |
| 禁止混入 | 不顺手改 FE store、不改 gRPC 动词名、不改 sudo 语义；**不**再开 write-policy 硬切 PR |
| 文档 | PR 回链 **HC\* + BT-n** |

终态（全部 BT 合并后）另加：生产路径 eslint `no-explicit-any` error（测试目录 override 关闭）。

---

## 9. 收益与非目标

相对 §3 生产 1080：

| 步骤 | 预期 |
| --- | --- |
| BT-1+2 | override / `super.* as any` 成片消失（auth 最明显）；平台不变式仍可留 HC2 override |
| ~~BT-3~~ | **已取消**（不把「删平台 CRUD override」当收益项） |
| BT-5 | 跨 app `createServiceByModel` Search 断言消失 |
| BT-4+6 | `(r as any).Id` / `$choysum` / `null as any` |
| BT-7 | 收残；生产 ≥70% 下降 |
| 测试 11816 | **不归零**；随生产类型变严自然降一截 |

**非目标：**

- 改 Kysely / 查询引擎
- 改 soft-delete / company / sudo **运行时语义**
- Vue 模板零断言、web store 泛型（原文 Track B）
- 测试 `as any` 归零
- 把 `ModelAPI` 做成作者面
- protobuf / OIDC 全量类型生成
- 嵌套 `Projected`（Q1.R，BT-4 之后）
- **用 `@Model` write policy 消灭平台 CRUD override**（原 BT-3 / 旧 HC4；IMD 下不成立）

---

## 10. 风险

| 风险 | 缓解 |
| --- | --- |
| `InstanceType<C>` 在 abstract mixin 上推成基类 | `ModelCtor<T>` 让 subclass `this` 带具体 T；BT-1 先在 Role / PropertyDefinition 做编译实验再铺 |
| HC2 重载导致 facade 内部推断爆炸 | BT-1 先单一返回 `InstanceType<C>`；BT-4 再加重载 |
| `createServiceByModel` 强制泛型改动面大 | BT-5 单独 PR；可用 `rg` 列全调用点一次改完 |
| Projected V1 字面量拓宽成 `string` | 调用点用 `as const` fields 或 `fields<Role>()('Id','Name')` helper |
| IMD 下游 override 拆掉上游不变式 | **接受**：扩展点即 override；文档与代码审查约定，不做虚假的 metadata「硬约束」 |

回滚：按 BT 独立 PR 回退。硬切 = 无旧签名可依赖，回滚就是 revert 该 PR，不留双轨。

---

## 11. 开工清单（BT-0 当天）

1. 冻结评审：§1 HC\* 有无反对（反对须改表，不口头漂）。
2. 开 **PR-BT-0**（§7.1）：只动 ctor 类型与引用，不动 CRUD 行为。
3. **PR-BT-1** 内第一 commit 做 Role.Create spike：`super.Create(value, returnFields)` 零断言；失败则停，不铺 BT-2。
4. 计数脚本可延到 **PR-BT-7a**；BT-0…6 用 §7.0 临时 `rg` 即可。
