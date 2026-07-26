// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { RepositoryPermissionDeniedFn } from '../authz/types';
import type {
  ConditionEnvelope,
  Entity,
  RepositoryMutationPayloadEncodeDepsLike,
  RepositoryMutationPayloadGuardDepsLike,
  RepositoryMutationPayloadValidateDepsLike,
} from '../types';
import { asObjectRecord } from '@/core/utils/object';
import {
  applyRepositoryMutationDefaultValues,
  assertRepositoryMutationPayloadsAllowed,
  encodeRepositoryMutationPayloads,
  validateRepositoryMutationPayload,
} from './mutation_payload_helpers';
import { _t } from '@/core/service/i18n_binder';
import { applyTranslatedFieldsForWrite } from '../projection/translated_field_codec';
import { stampMonetaryScalesForWriteMany } from '../projection/monetary_scale';

export type RepositoryCreateWriteAuthzDeps = {
  meta: ModelMetadata;
  getRecordRuleEnvelope: (op: 'create') => Promise<ConditionEnvelope>;
  permissionDenied: RepositoryPermissionDeniedFn;
};

export type RepositoryCreateWritePrepareDeps = {
  meta: ModelMetadata;
  generateId: () => string;
  applyDefaultCompanyIdOnCreate: (entity: Entity) => Entity;
} & RepositoryMutationPayloadGuardDepsLike<Entity> &
  RepositoryMutationPayloadValidateDepsLike<Entity, 'create'> &
  RepositoryMutationPayloadEncodeDepsLike<Entity>;

export async function ensureRepositoryCreateAllowed(params: RepositoryCreateWriteAuthzDeps): Promise<ConditionEnvelope> {
  const recordRuleEnvelope = await params.getRecordRuleEnvelope('create');
  if (recordRuleEnvelope.kind !== 'false') {
    return recordRuleEnvelope;
  }

  throw params.permissionDenied(
    'record_rule_denied',
    _t('record rule denied', { scope: 'service/orm/repository/write/create_helpers' }),
    {
      model: params.meta.fullModelName || params.meta.modelName || params.meta.name,
      op: 'create',
      reason: recordRuleEnvelope.reason || 'denied',
    }
  );
}

export async function prepareRepositoryCreateEntities(params: RepositoryCreateWritePrepareDeps, value: Entity[]): Promise<Entity[]> {
  await assertRepositoryMutationPayloadsAllowed(params, value || []);

  const entitiesWithId = (value || []).map(entity => {
    const id = asObjectRecord(entity)?.Id;
    return id ? entity : { ...entity, Id: params.generateId() };
  });

  const preparedEntities = applyRepositoryMutationDefaultValues(
    {
      applyDefaultMutationValues: entity => params.applyDefaultCompanyIdOnCreate(entity),
    },
    entitiesWithId
  );
  const stampedEntities = await stampMonetaryScalesForWriteMany(
    params.meta,
    preparedEntities.map(entity => ({ input: entity }))
  );
  for (const entity of stampedEntities) {
    await validateRepositoryMutationPayload(
      {
        validateFields: (input, mode) => params.validateFields(input, mode),
      },
      entity,
      'create'
    );
  }

  const entitiesForEncode = stampedEntities.map(entity =>
    applyTranslatedFieldsForWrite(params.meta, entity, { mode: 'create' })
  );
  return encodeRepositoryMutationPayloads(params, entitiesForEncode);
}
