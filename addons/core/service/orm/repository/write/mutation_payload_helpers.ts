// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type {
  Entity,
  RepositoryMutationPayloadDefaultsDepsLike,
  RepositoryMutationPayloadEncodeDepsLike,
  RepositoryMutationPayloadGuardDepsLike,
  RepositoryMutationPayloadValidateDepsLike,
} from '../types';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryMutationPayloadMode = 'create' | 'update';

export type RepositoryMutationPayloadGuardDeps = RepositoryMutationPayloadGuardDepsLike<Entity>;

export type RepositoryMutationPayloadDefaultsDeps = RepositoryMutationPayloadDefaultsDepsLike<Entity>;

export type RepositoryMutationPayloadValidateDeps<TMode extends RepositoryMutationPayloadMode> = RepositoryMutationPayloadValidateDepsLike<
  Entity,
  TMode,
  ObjectRecord
>;

export type RepositoryMutationPayloadEncodeDeps = RepositoryMutationPayloadEncodeDepsLike<Entity>;

export async function assertRepositoryMutationPayloadsAllowed(params: RepositoryMutationPayloadGuardDeps, payloads: Entity[]): Promise<void> {
  for (const payload of payloads || []) {
    await params.assertFieldRuleWriteAllowed(payload);
  }
}

export function applyRepositoryMutationDefaultValues(params: RepositoryMutationPayloadDefaultsDeps, payloads: Entity[]): Entity[] {
  return (payloads || []).map(payload => params.applyDefaultMutationValues(payload));
}

export async function validateRepositoryMutationPayload<TMode extends RepositoryMutationPayloadMode>(
  params: RepositoryMutationPayloadValidateDeps<TMode>,
  payload: Entity,
  mode: TMode,
  validationContexts?: Array<ObjectRecord | undefined>
): Promise<void> {
  const contexts = validationContexts && validationContexts.length > 0 ? validationContexts : [undefined];
  for (const current of contexts) {
    await params.validateFields(payload, mode, current);
  }
}

export function encodeRepositoryMutationPayloads(params: RepositoryMutationPayloadEncodeDeps, payloads: Entity[]): Entity[] {
  return (payloads || []).map(payload => params.encodeForDb(payload));
}
