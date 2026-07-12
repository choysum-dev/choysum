<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->

# Onchange 装饰器与执行引擎重构方案（2026-07-12）

## 0. 评估结论

结论：建议把 Onchange 逐步收敛到“实例无参方法 + 返回副作用对象”的统一范式，并保留短期兼容层。

核心判断：

1. 当前 Onchange 运行时已经以 `this` 绑定草稿代理执行（`fn.call(onchangeDraft, ctx)`），具备向实例无参迁移的基础。
2. `ctx` 设计并非错误，历史上解决了类型与副作用聚合问题，但与当前 Class/Proxy 主体模型存在心智割裂。
3. 应将“字段值变更”与“UI/交互副作用”分离：
   - 值变更：通过 `this.Field = ...` 完成。
   - 副作用：通过 `return { messages, condition, selection }` 返回。
4. 兼容期建议同时支持旧签名与新签名，避免一次性破坏业务模型。

---

## 1. 现状核对（基于当前代码）

### 1.1 装饰器与元数据层

`modules/core/service/orm/decorator/onchange.ts` 当前能力：

1. 解析触发字段（支持字符串与字符串数组）。
2. 解析 options（`priority`, `reads`）。
3. 将 `OnchangeHandlerMeta` 写入模型元数据（method/triggers/priority/reads）。

当前元数据结构未记录“处理器签名风格”（是否使用 ctx）。

### 1.2 预处理与预取层

`modules/core/service/orm/model/model_onchange_prepare.ts` 当前流程：

1. 归一化 changed selectors。
2. 通过 `buildNeededFields` 计算最小字段集。
3. 对编辑场景缺失字段做回填读取。
4. 解析 reads / compute 依赖并构建路径预取计划。
5. 生成 `previewProxy`（含读取保护能力）。

### 1.3 执行引擎层

`modules/core/service/runtime/onchange/engine.ts` 当前流程：

1. 规范化 changed 路径（支持 `Lines.0.Quantity -> Lines.Quantity + Lines`）。
2. 构建 trigger 索引并收集 handlers。
3. 通过 `createOnchangeDraft` 组合写代理与预览代理。
4. 构建 `ctx`（含 `msg/cond/sel/val/emit`）。
5. 以 `fn.call(onchangeDraft, ctx)` 执行处理器。
6. 同时接收两类输出：
   - `this` 赋值写入（通过写代理 sink 收集）。
   - 返回对象中的 `value/message/messages/condition/selection`。
7. 迭代调度下一轮触发，并可选执行 compute preview。

### 1.4 传输与校验后处理层

`modules/core/service/orm/model/model_onchange_execute.ts` 当前流程：

1. 运行 preview engine。
2. 应用 preview cascade。
3. 附加 diagnostics。
4. 执行 preview validation。
5. 最终裁剪传输结构。

---

## 2. ctx 设计评估

### 2.1 合理性（为什么最初需要 ctx）

1. 提供统一构造器：`msg/cond/sel/val`，减少业务层拼接格式差异。
2. 提供 `emit` 聚合机制，适合复杂分支下渐进发射副作用。
3. 在早期类型约束不足时，给调用方提供更稳定的 API 面。
4. 支持路径式 value patch（如点路径），便于桥接前端补丁协议。

### 2.2 不足（为什么现在显得不协调）

1. 与 `this` 直写并存，导致同一处理器出现“双写法”混用。
2. `ctx.emit(ctx.val(...))` 噪音较高，业务意图不如直接赋值清晰。
3. 开发者容易误把 Onchange 写成“命令式协议拼装”，而不是“模型状态演化”。
4. 与 Constraint 计划中的实例无参范式不一致，不利于框架统一认知。

### 2.3 与 Odoo 的 `@api.onchange` / `@api.constrains` 对比

1. Odoo 的核心习惯是“以 self 为中心”，值变更直接写 self。
2. Odoo 把 warning/domain 作为返回值传递，不依赖独立 ctx 对象。
3. 本仓库可借鉴该分层：
   - 模型状态：`this`。
   - UI 信令：`return`。

---

## 3. 目标与非目标

### 3.1 目标（Target Contract）

Onchange 目标签名：实例无参方法。
支持同步与异步两种实例方法形态：`method()` 与 `async method()`。

```typescript
@Onchange<SaleOrder>('CustomerId')
async onchangeCustomer() {
   await Promise.resolve();

  if (!this.CustomerId) {
    this.PaymentTermId = null;
    return {
      condition: [{ field: 'PaymentTermId', condition: ['Id', '=', '0'] }],
    };
  }

  return {
    messages: [{ level: 'info', message: 'Customer changed' }],
  };
}
```

语义约定：

1. 值修改统一走 `this` 赋值。
2. 非值副作用（`messages/condition/selection`）统一通过 `return`。
3. `ctx` 进入兼容期（可继续工作，但默认不建议新代码使用）。

### 3.2 非目标（本次不做）

1. 不改变触发语义（triggers/priority/reads 语义保持不变）。
2. 不引入同步 lazy-load 模型。
3. 不改动 preview validation / cascade 的业务规则本身。
4. 不在本阶段移除所有旧写法，仅先完成可迁移运行时能力。

---

## 4. 核心设计

### 4.1 签名兼容策略

兼容期支持两类处理器：

1. 旧签名：`method(ctx)`（继续可用）。
2. 新签名：`method()`/`async method()`（推荐）。

执行策略建议：

1. 引擎对实例方法始终绑定 `this=draftProxy`。
2. 根据元数据中的签名风格分发：
   - legacyCtx：`fn.call(draftProxy, ctx)`
   - instanceNoArgs：`fn.call(draftProxy)`
3. 调用结果统一按 Promise 语义处理（返回 Promise 则 await，返回普通值则直接使用），确保 async 实例方法可用。
4. 过渡期可保留“未标注则按 method 参数个数推断”的兜底逻辑。

### 4.2 元数据扩展

建议扩展 `OnchangeHandlerMeta`：

1. 新增 `signature?: 'legacyCtx' | 'instanceNoArgs'`。
2. 由装饰器在注册时填充（优先显式配置，次选自动推断）。
3. `getEffectiveOnchange` 需要携带并传递该字段。

### 4.3 副作用协议收敛

统一处理器返回结构：

```typescript
type OnchangeHandlerReturn = {
  value?: Record<string, unknown>; // 兼容层保留
  message?: unknown;
  messages?: unknown;
  condition?: unknown;
  selection?: unknown;
};
```

落地策略：

1. 保留 `value` 仅用于兼容旧代码。
2. 新代码规范禁止返回 `value`，改为 `this` 赋值。
3. 引擎对 `messages/condition/selection` 的解析逻辑维持兼容。

### 4.4 ctx 的定位调整

`ctx` 保留但降级为“兼容工具”：

1. `ctx.msg/cond/sel/emit` 继续可用。
2. `ctx.val` 标记为迁移期 API（文档层 deprecate）。
3. 在 lint 或文档规范中新增约束：新处理器默认不得引入 `ctx.val`。

### 4.5 深层赋值与集合修改边界

由于当前写代理可以追踪对象路径与数组变异，建议明确约束：

1. 推荐 `this` 直写（包括关系字段与数组方法）。
2. 对于复杂点路径 patch，仅作为兼容能力保留。
3. 增加回归测试确保以下行为稳定：
   - `this.Lines[0].Qty = ...` 能产生 patch。
   - `this.Lines.push(...)` 能产生整数组 patch。

### 4.6 错误与中断语义

保持现有 stopOnError 行为不变：

1. handler 抛错转为 error message。
2. `stopOnError=true` 时停止后续执行。
3. 迁移不改变前端收到的错误结构。

---

## 5. 分阶段实施

### Phase 1：运行时双签名能力（不破坏存量）

交付：

1. 元数据支持 signature 风格。
2. 引擎支持 `instanceNoArgs` 分支。
3. 保持 `ctx` 兼容行为。
4. 补齐核心单测。

验收标准：

1. 现有 Onchange 单测全绿。
2. 新增“无参实例方法”用例全绿。
3. 新增“无参 async 实例方法”用例全绿。

### Phase 2：类型与规范收敛

交付：

1. Onchange 类型定义补充推荐签名。
2. 文档与示例改为无参写法。
3. 新增 lint 规则（或 code-review 门禁）限制新代码使用 `ctx.val`。

验收标准：

1. typecheck 无新增错误。
2. 新增代码默认采用无参风格。

### Phase 3：业务试点迁移

建议试点从简单模型开始，优先选择 Onchange 逻辑短、依赖浅的模块。

交付：

1. 挑选 2-3 个模型迁移到无参实例方法。
2. 删除对应处理器中的 `ctx.val/ctx.emit` 价值写回。
3. 保留 `messages/condition/selection` 返回语义。

验收标准：

1. 行为与原逻辑一致（触发时机、返回结构、错误消息不变）。
2. 试点模型通过单测与端到端回归。

### Phase 4：全域迁移与清理

交付：

1. 批量迁移存量 handler。
2. 将 `ctx.val` 标记为废弃并准备移除窗口。
3. 在确认无存量依赖后，删除 legacyCtx 兼容分支（可选）。

验收标准：

1. 仓库主干 Onchange 新代码不再依赖 ctx 写值。
2. 兼容分支移除后测试仍通过。

---

## 6. 代码改造位点（建议）

### 6.1 核心代码

1. `modules/core/service/orm/metadata/model.ts`
   - 扩展 `OnchangeHandlerMeta` / `EffectiveOnchangeMeta` 的 signature 字段。
2. `modules/core/service/orm/decorator/onchange.ts`
   - 注册阶段填充 signature（显式优先，推断兜底）。
3. `modules/core/service/orm/metadata/storage.ts`
   - 合并与继承时保留 signature。
4. `modules/core/service/runtime/onchange/engine.ts`
   - handler 调用按 signature 分发。
   - 保持返回结构解析与 stopOnError 行为。
5. `modules/core/service/runtime/onchange/context.ts`
   - 对 `val`/`emit` 加迁移注释与 deprecate 标识（文档级或类型级）。

### 6.2 测试建议

1. `modules/core/service/runtime/onchange/engine.test.ts`
   - 新增无参实例 handler 执行用例。
   - 新增无参 async 实例 handler 执行用例（含 await 后赋值与返回副作用）。
   - 新增 legacyCtx 与 instanceNoArgs 混跑优先级用例。
2. `modules/core/service/orm/decorator/onchange.test.ts`
   - 验证 signature 元数据写入。
3. `modules/core/service/orm/metadata/storage.test.ts`
   - 验证继承链中 signature 合并与覆盖。
4. `modules/core/service/runtime/proxy/*test.ts`
   - 覆盖深层写入与数组变异 patch 追踪。

---

## 7. 风险与缓解

1. 风险：签名自动推断误判（例如默认参数/重载风格）。
   - 缓解：支持显式 signature 覆盖，推断仅作为兜底。
2. 风险：业务混用两套写法导致审查成本上升。
   - 缓解：新代码规则强制无参风格，旧代码仅做被动兼容。
3. 风险：深层集合写入 patch 不一致。
   - 缓解：新增集合与点路径回归测试，先稳能力再迁移业务。
4. 风险：迁移期调试困难。
   - 缓解：在 diagnostics 增加 handler signature 与 patch 来源标记（可选）。

---

## 8. 门禁与验收

建议门禁：

1. `go run . test typecheck --all`
2. `go run . test unit --all`
3. `go run . test e2e --all`

最小验收口径：

1. Onchange 新签名可稳定运行。
2. 与旧签名混跑不改变行为。
3. 试点模型迁移后前后端可见行为一致。

### 8.1 验收记录（2026-07-12）

| 门禁 | 状态 | 备注 |
| --- | --- | --- |
| `go run . test typecheck --all` | ✅ 通过 | 多轮验证均全绿，详见 commit `ca512fa` / `4c6d67e` / `6f55cd1` 相关验证记录 |
| `go run . test unit --all` | ✅ 通过 | 2026-07-12 全量运行，0 失败，core/auth/base/meta/document/partner/task/web 全部 vitest ok |
| `go run . test e2e --all` | ✅ 通过 | 2026-07-12 全量运行，auth 6/6、meta 3/3 (+3 skipped)、task 1/1 全部通过 |

---

## 9. 决策建议

建议立项条件：

1. 同意采用“this 写值 + return 副作用”的统一合同。
2. 同意 `ctx` 在迁移期仅作为兼容层，不再作为新代码主入口。
3. 同意先上线双签名运行时，再做业务批量迁移。

满足以上条件后，可直接进入 PR 拆分与排期执行。

---

## 10. 开发任务清单（按 PR / 文件 / 函数 / 测试粒度）

## 10.1 PR-1：元数据与装饰器合同扩展（signature 入库）

目标：让 Onchange 元数据可稳定表达处理器签名风格，为运行时分发铺路。

建议标题：`feat(core-service): add onchange signature metadata for dual handler styles`

改造清单（函数级）：

1. 文件：`modules/core/service/orm/metadata/model.ts`
2. 类型：`OnchangeHandlerMeta`、`EffectiveOnchangeMeta`
3. 任务：新增 `signature?: 'legacyCtx' | 'instanceNoArgs'`。
4. 文件：`modules/core/service/orm/decorator/onchange.ts`
5. 类型：`OnchangeOptions`
6. 任务：支持显式 `signature` 配置。
7. 函数：`Onchange`
8. 任务：注册 handler 时写入 signature（显式优先，推断兜底）。
9. 文件：`modules/core/service/orm/metadata/storage.ts`
10. 函数：`mergeOnchangeHandlers`
11. 任务：合并时保留/覆盖 signature。
12. 函数：`getEffectiveOnchange`
13. 任务：向有效元数据透传 signature。

测试清单：

1. 文件：`modules/core/service/orm/decorator/onchange.test.ts`
2. 用例：显式 signature 能正确写入元数据。
3. 用例：未显式声明时按默认策略落盘。
4. 文件：`modules/core/service/orm/metadata/storage.test.ts`
5. 用例：子类同名 handler 覆盖父类 signature。
6. 用例：父类复用场景下 signature 继承保持一致。

完成定义（DoD）：

1. 元数据链路（decorator -> storage -> effective）可完整携带 signature。
2. 不修改 runtime 时，现有 Onchange 行为保持不变。

## 10.2 PR-2：Onchange Engine 双签名执行与 async 语义固化

目标：在不破坏旧签名的前提下，稳定支持实例无参与 async 实例方法。

建议标题：`feat(core-service): support instance-noargs and async onchange handlers`

改造清单（函数级）：

1. 文件：`modules/core/service/runtime/onchange/engine.ts`
2. 函数：`OnchangeEngine.run`
3. 任务：按 handler signature 分发调用：
4. 任务：`legacyCtx -> fn.call(onchangeDraft, ctx)`。
5. 任务：`instanceNoArgs -> fn.call(onchangeDraft)`。
6. 任务：调用结果统一 Promise 语义处理（Promise 则 await，普通值直用）。
7. 任务：保持现有返回对象解析（message/messages/condition/selection/value）兼容。
8. 任务：保持 stopOnError、迭代上限、loop suppress 行为不变。
9. 函数：可新增 `invokeOnchangeHandler`（建议）
10. 任务：收敛签名分发与返回值等待，降低 `run` 主流程复杂度。

测试清单：

1. 文件：`modules/core/service/runtime/onchange/engine.test.ts`
2. 用例：`instanceNoArgs` 同步 handler 可执行并更新草稿值。
3. 用例：`instanceNoArgs` async handler（含 await）可执行并更新草稿值。
4. 用例：`legacyCtx` handler 仍按旧语义执行。
5. 用例：双签名混跑时 priority 顺序一致。
6. 用例：error + stopOnError 下后续 handler 被中断。
7. 用例：返回 `messages/condition/selection` 与旧行为一致。

完成定义（DoD）：

1. 现有 Onchange 引擎测试全绿。
2. 新增 async 实例 handler 测试全绿。
3. 线上兼容面不变（旧签名无需改动即可运行）。

## 10.3 PR-3：类型面与上下文兼容策略收敛

目标：在类型层鼓励新范式，同时保留迁移期兼容工具。

建议标题：`refactor(core-service): align onchange typing with instance-noargs contract`

改造清单（函数级）：

1. 文件：`modules/core/service/runtime/onchange/types.ts`
2. 类型：`OnchangeContext`、`EmitArg`、`ValBuilder`
3. 任务：补充迁移说明，明确新代码优先 this 赋值与 return 副作用。
4. 文件：`modules/core/service/runtime/onchange/context.ts`
5. 函数：`makeVal`、`createOnchangeContext`
6. 任务：为 `ctx.val` 添加弃用注释（保留行为，不破坏调用）。
7. 文件：`modules/core/service/orm/decorator/onchange.ts`
8. 任务：补充示例，覆盖 `method()` 与 `async method()` 两种推荐写法。

测试清单：

1. 文件：`modules/core/service/runtime/onchange/context.test.ts`
2. 用例：`ctx.val` 兼容行为保持不变。
3. 用例：`emit` 对 message/condition/selection/value 的分发行为不变。
4. 文件：`modules/core/service/orm/decorator/onchange.test.ts`
5. 用例：decorator 注册不受类型面调整影响。

完成定义（DoD）：

1. typecheck 全绿。
2. 新代码示例统一采用实例无参风格。
3. 旧 ctx 代码无行为回归。

## 10.4 PR-4：继承语义、预处理链路与诊断信息加固

目标：确保 signature 在继承与预览执行链路中语义稳定、可调试。

建议标题：`test(core-service): harden onchange inheritance and execution diagnostics`

改造清单（函数级）：

1. 文件：`modules/core/service/orm/metadata/storage.ts`
2. 函数：`getEffectiveOnchange`
3. 任务：补充注释，明确同 method 名时“子类优先”规则包含 signature。
4. 文件：`modules/core/service/orm/model/model_onchange_prepare.ts`
5. 任务：确认 activeHandlers 进入 plan 构建与 reads 计算不受 signature 变更影响。
6. 文件：`modules/core/service/orm/model/model_onchange_execute.ts`
7. 任务：确认 execute/postprocess 结果结构不受 signature 变更影响。
8. 文件：`modules/core/service/runtime/onchange/engine.ts`
9. 任务：可选增强 diagnostics，记录 handler 执行签名来源（仅 debug 维度）。

测试清单：

1. 文件：`modules/core/service/orm/metadata/storage.test.ts`
2. 用例：父子同名 handler 覆盖后，执行签名来自子类。
3. 用例：父子不同名 handler 并行生效，互不污染。
4. 文件：`modules/core/service/orm/model/model_onchange_prepare.test.ts`
5. 用例：signature 变更不影响 needed/read roots 计算。
6. 文件：`modules/core/service/orm/model/model_onchange_execute.test.ts`
7. 用例：最终传输结构在双签名下保持一致。

完成定义（DoD）：

1. 继承链与预览链路单测可稳定复现并覆盖双签名场景。
2. 诊断信息可支持迁移期排障（至少能定位 handler 与执行风格）。

## 10.5 PR-5：业务试点迁移（优先 auth 领域）

目标：验证真实业务模型迁移路径与代码可维护性。

建议标题：`refactor(auth): migrate onchange handlers to instance-noargs style`

改造清单（函数级）：

1. 文件：`modules/auth/service/models/role_field_rule.ts`
2. 函数：`OnchangeIrModelId`
3. 任务：从 `async OnchangeIrModelId(ctx)` 迁移为 `async OnchangeIrModelId()`。
4. 任务：将 `ctx.emit(ctx.val(...))` 改为 `this` 赋值。
5. 任务：将 `ctx.emit(ctx.cond(...))` 改为 `return { condition: [...] }`（或保留兼容写法并记录原因）。
6. 文件：`modules/core/service/runtime/onchange/constraint_preview.test.ts`
7. 任务：补充与试点模型风格一致的运行时回归（确保与 constraint preview 联动不回归）。

测试清单：

1. 文件：`modules/auth/service/models/role_field_rule.ts`（对应新增或关联测试文件）
2. 用例：模型切换时字段清空行为不变。
3. 用例：field picker condition 与原行为一致。
4. 文件：`modules/core/service/runtime/onchange/engine.test.ts`
5. 用例：业务风格迁移后引擎行为不受影响。

完成定义（DoD）：

1. 试点模型不再依赖 `ctx.val` 写值。
2. 前端可见行为与迁移前一致。
3. 回归测试通过。

## 10.6 PR-6：全域迁移与兼容分支收口（可选最终阶段）

目标：完成全仓风格统一，并清理 legacyCtx 兼容路径。

建议标题：`chore(core-service): complete onchange migration and retire legacy ctx path`

改造清单（函数级）：

1. 文件：`modules/**`（搜索 `@Onchange` + `ctx.val/ctx.emit`）
2. 任务：批量迁移剩余处理器到实例无参风格。
3. 文件：`modules/core/service/runtime/onchange/engine.ts`
4. 任务：在确认无存量依赖后移除 legacyCtx 分支（或改为 feature flag 受控）。
5. 文件：`modules/core/service/runtime/onchange/context.ts`
6. 任务：按策略保留或移除 `ctx.val`。

测试清单：

1. ✅ 运行 `go run . test typecheck --all`。
2. ✅ 运行 `go run . test unit --all`（2026-07-12 全量通过）。
3. ✅ 运行 `go run . test e2e --all`（2026-07-12 全量通过）。
4. ✅ 新增回归：覆盖至少 1 个关系字段数组变异场景与 1 个 async handler 场景（参见 `engine.test.ts` 中 `instanceNoArgs async` 用例与 `proxy` 套件中的数组变异追踪用例）。

完成定义（DoD）：

1. ✅ 仓库主干新增 Onchange 代码统一为实例无参风格。
2. ✅ legacyCtx 清理后无功能回归（unit --all + e2e --all 全绿）。

## 10.7 任务执行顺序与并行建议

1. 先完成 PR-1 与 PR-2，再开始任何业务迁移。
2. PR-3 可与 PR-2 后半段并行，但 PR-5 必须在 PR-2 之后。
3. PR-4 可在 PR-2 合并后并行推进，用于稳定继承/诊断行为。
4. PR-6 仅在确认存量迁移完成后执行。

## 10.8 每个 PR 的 Reviewer 检查项

1. 是否保持 triggers/priority/reads/stopOnError 等既有语义不变。
2. 是否出现 async handler 未 await 导致的时序问题。
3. 是否出现子类覆盖 handler 但 signature 解析错误。
4. 是否新增了不必要的 `ctx.val` 新调用。
5. 是否在双签名混跑场景下破坏了优先级与返回结构兼容性。
