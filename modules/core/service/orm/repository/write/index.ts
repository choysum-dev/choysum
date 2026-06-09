// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export {
  createRepositoryCreateMutationPayloadDeps,
  createRepositoryUpdateMutationPayloadDeps,
  createRepositoryMutationWriteTargetDeps,
  createRepositoryMutationWriteConditionDeps,
  createRepositoryMutationWriteFacadeDeps,
  createRepositoryCreateFacadeDeps,
  createRepositoryUpdateFacadeDeps,
  createRepositoryDeleteWriteDeps,
  createRepositoryCreateWriteDeps,
  createRepositoryUpdateWriteDeps,
} from './deps';
export type { RepositoryCreateWriteRuntimeDeps, RepositoryCreateWritePostWriteDeps } from './create';
export { insertRepositoryCreateEntities, applyRepositoryCreatePostWrite, executeRepositoryCreate } from './create';
export type { RepositoryCreateWriteAuthzDeps, RepositoryCreateWritePrepareDeps } from './create_helpers';
export { ensureRepositoryCreateAllowed, prepareRepositoryCreateEntities } from './create_helpers';
export type {
  RepositoryDeleteWriteRuntimeDeps,
  RepositoryDeleteWritePostWriteDeps,
  RepositoryDeleteSoftDeletePreWriteDeps,
  RepositoryDeleteQueryPrepareDeps,
} from './delete';
export {
  prepareRepositorySoftDeleteWrite,
  prepareRepositoryDeleteQuery,
  executeRepositoryDeleteRuntime,
  applyRepositoryDeletePostWrite,
  executeRepositoryDelete,
  executeRepositoryHardDelete,
} from './delete';
export type { RepositoryDeleteChild } from './delete_child_factory';
export { createRepositoryDeleteChild } from './delete_child_factory';
export type { RepositoryDeleteWriteTargetDeps, RepositoryDeleteWriteConditionDeps } from './delete_helpers';
export { resolveRepositoryDeleteTargetIds, applyRepositoryDeleteCondition } from './delete_helpers';
export type {
  RepositoryMutationPayloadMode,
  RepositoryMutationPayloadGuardDeps,
  RepositoryMutationPayloadDefaultsDeps,
  RepositoryMutationPayloadValidateDeps,
  RepositoryMutationPayloadEncodeDeps,
} from './mutation_payload_helpers';
export {
  assertRepositoryMutationPayloadsAllowed,
  applyRepositoryMutationDefaultValues,
  validateRepositoryMutationPayload,
  encodeRepositoryMutationPayloads,
} from './mutation_payload_helpers';
export type { RepositoryMutationWriteOp, RepositoryMutationWriteTargetDeps, RepositoryMutationWriteConditionDeps } from './mutation_write_helpers';
export { resolveRepositoryMutationWriteTargetIds, applyRepositoryMutationWriteCondition } from './mutation_write_helpers';
export type {
  RepositoryUpdateWriteRuntimeDeps,
  RepositoryUpdateWritePostWriteDeps,
  RepositoryUpdateWriteQueryPrepareDeps,
  RepositoryUpdateWriteTargetResolveDeps,
  RepositoryUpdateWriteSanitizedPayloadDeps,
} from './update';
export {
  resolveRepositoryUpdatePayloadTargets,
  prepareRepositoryUpdateSanitizedPayload,
  prepareRepositoryUpdatePayload,
  prepareRepositoryUpdateQuery,
  executeRepositoryUpdateRuntime,
  applyRepositoryUpdatePostWrite,
  executeRepositoryUpdate,
} from './update';
export type { RepositoryUpdateWriteTargetDeps, RepositoryUpdateWriteConditionDeps, RepositoryUpdateWriteCurrentRowsDeps } from './update_helpers';
export { resolveRepositoryUpdateTargetIds, applyRepositoryUpdateCondition, loadRepositoryUpdateValidationCurrentRows } from './update_helpers';
