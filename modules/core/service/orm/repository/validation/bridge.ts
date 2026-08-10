// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { validateRuntimeOrThrow } from '../../../runtime/runtime_validation_facade';
import type BaseModel from '../../model/model';
import type { ModelCtor } from '../../metadata/field';
import type { ModelMetadata } from '../../metadata';
import { ValidationPipelineError, type ConstraintMode } from '../../metadata';
import type { Entity } from '../types';
import type { Repository } from '../repository';
import { wrapRepositoryValidationError } from './error_helpers';
import { throwRepositorySqlWriteError } from './sql_helpers';
import type { ObjectRecord } from '../../../../utils/types';
import {
  recordRepositoryPlatformCreateWhitelistAudit,
  resolveRepositoryPlatformCreateWriteWhitelist,
  resolveRepositoryPlatformRejectUnknownFields,
} from './platform_helpers';

export { selectPrimaryValidationIssue, wrapRepositoryValidationError } from './error_helpers';
export { throwRepositorySqlWriteError } from './sql_helpers';
export {
  recordRepositoryPlatformCreateWhitelistAudit,
  resolveRepositoryPlatformCreateWriteWhitelist,
  resolveRepositoryPlatformRejectUnknownFields,
} from './platform_helpers';

export async function validateRepositoryWrite(params: {
  meta: ModelMetadata;
  repository: Repository;
  requestContext: unknown;
  getValidationBypassDepth: () => number;
  input: Entity;
  mode: ConstraintMode;
  current?: ObjectRecord;
}): Promise<void> {
  const { meta, repository, requestContext, getValidationBypassDepth, input, mode, current } = params;

  try {
    const internalValidationBypass = getValidationBypassDepth() > 0;
    const internalComputedWriteWhitelist = internalValidationBypass ? Array.from(meta.computeGraph?.persistedComputeFields || new Set<string>()) : [];
    const configuredCreateWhitelist = mode === 'create' ? resolveRepositoryPlatformCreateWriteWhitelist(meta, requestContext) : [];
    const platformCreateWriteWhitelist = Array.from(new Set([...configuredCreateWhitelist, ...internalComputedWriteWhitelist]));
    await validateRuntimeOrThrow(
      {
        mode,
        model: meta.type as ModelCtor<BaseModel> & typeof BaseModel,
        metadata: meta,
        current,
        values: input as ObjectRecord,
        changedFields: new Set(Object.keys(input || {})),
        repository,
        requestContext,
      },
      {
        platformCreateWriteWhitelist,
        platformRejectUnknownFields: resolveRepositoryPlatformRejectUnknownFields(requestContext),
        onPlatformCreateWhitelistHit: (fields: string[]) => recordRepositoryPlatformCreateWhitelistAudit(meta, requestContext, mode, fields),
      }
    );
  } catch (error) {
    if (error instanceof ValidationPipelineError) {
      throw wrapRepositoryValidationError(meta, error, mode);
    }
    throw error;
  }
}
