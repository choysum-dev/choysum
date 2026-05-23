// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { BaseModel } from '../orm/model';
export type { ModelOptions, modelOptions, HookPhase, MigrationPhase, MigrationOptions } from '../orm/decorator';
export {
  Model,
  defineModelOptions,
  Constraint,
  Field,
  isTopLevelGrpcRequest,
  Onchange,
  HookPreInit,
  HookPostInit,
  HookPreUpgrade,
  HookPostUpgrade,
  HookPreUninstall,
  HookPostUninstall,
  Migration,
} from '../orm/decorator';
