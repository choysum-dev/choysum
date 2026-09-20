// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type {
  SelectResult,
  RepositoryMutationPayloadDefaultsDepsLike,
  RepositoryMutationPayloadEncodeDepsLike,
  RepositoryMutationPayloadGuardDepsLike,
  RepositoryMutationPayloadValidateDepsLike,
} from '../types/engine';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryMutationPayloadMode = 'create' | 'update';

export type RepositoryMutationPayloadGuardDeps = RepositoryMutationPayloadGuardDepsLike<SelectResult>;

export type RepositoryMutationPayloadDefaultsDeps = RepositoryMutationPayloadDefaultsDepsLike<SelectResult>;

export type RepositoryMutationPayloadValidateDeps<TMode extends RepositoryMutationPayloadMode> = RepositoryMutationPayloadValidateDepsLike<
  SelectResult,
  TMode,
  ObjectRecord
>;

export type RepositoryMutationPayloadEncodeDeps = RepositoryMutationPayloadEncodeDepsLike<SelectResult>;

export async function assertRepositoryMutationPayloadsAllowed(params: RepositoryMutationPayloadGuardDeps, payloads: SelectResult[]): Promise<void> {
  for (const payload of payloads || []) {
    await params.assertFieldRuleWriteAllowed(payload);
  }
}

export function applyRepositoryMutationDefaultValues(params: RepositoryMutationPayloadDefaultsDeps, payloads: SelectResult[]): SelectResult[] {
  return (payloads || []).map(payload => params.applyDefaultMutationValues(payload));
}

export async function validateRepositoryMutationPayload<TMode extends RepositoryMutationPayloadMode>(
  params: RepositoryMutationPayloadValidateDeps<TMode>,
  payload: SelectResult,
  mode: TMode,
  validationContexts?: Array<ObjectRecord | undefined>
): Promise<void> {
  const contexts = validationContexts && validationContexts.length > 0 ? validationContexts : [undefined];
  for (const current of contexts) {
    await params.validateFields(payload, mode, current);
  }
}

export function encodeRepositoryMutationPayloads(params: RepositoryMutationPayloadEncodeDeps, payloads: SelectResult[]): SelectResult[] {
  return (payloads || []).map(payload => params.encodeForDb(payload));
}
