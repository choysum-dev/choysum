// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { Entity, FieldSelection, SoftDeleteOptions, UpdateOptions, DeleteOptions } from '../repository/types';
import { MODEL_SYMBOLS } from '../../runtime/proxy';
import { LockUtils } from '../utils/lock';
import { EntityConverter } from '../utils/converter';
import { RelationFactory } from '../relation';
import type { RelationChangesCollection } from '../relation/types';
import type { RuntimeModelCtor } from './types';
import { resolveRepositoryWithSoftDeleteOptions } from './model_soft_delete_scope';
import { toTransportObject as toTransportObjectImpl } from './model_runtime';
import { collectModelUpstreamInverseFields, getModelRuntimeMetadata, recomputeModelMetadata, triggerModelUpstream } from './model_runtime_service_facade';
import { getRuntimeErrorMessage, runWithValidationBypass } from './model_write_helpers';
import type { ValidationBypassCapable } from './model_write_helpers';
import { AuditUidUtils } from '../utils/audit_uid';
import type { UnknownRecord } from '../../../utils/types';
import { _t } from '@/core/service/i18n_binder';

type InstanceRepositoryLike = ValidationBypassCapable & {
  update(values: UnknownRecord, condition: unknown): Promise<Array<unknown>>;
  search(condition: unknown, options?: { fields?: unknown }): Promise<UnknownRecord[]>;
  delete(condition: unknown): Promise<Array<unknown>>;
};

type InstanceState = Record<PropertyKey, unknown>;

function stateOf(instance: BaseModel): InstanceState {
  return instance as unknown as InstanceState;
}

function invokeSymbol<T>(instance: BaseModel, symbolKey: symbol): T {
  const handler = stateOf(instance)[symbolKey];
  if (typeof handler !== 'function') {
    throw new Error(`model symbol handler is missing: ${String(symbolKey)}`);
  }
  return (handler as () => T)();
}

function resetModelChanges(instance: BaseModel): void {
  invokeSymbol<void>(instance, MODEL_SYMBOLS.resetChanges);
  invokeSymbol<void>(instance, MODEL_SYMBOLS.resetRelationChanges);
}

function resolveInstanceReadRepository(instance: BaseModel, options?: SoftDeleteOptions): InstanceRepositoryLike {
  return resolveRepositoryWithSoftDeleteOptions(instance.constructor as RuntimeModelCtor, options) as unknown as InstanceRepositoryLike;
}

function mergeFields<T extends BaseModel>(instance: T, newFields: FieldSelection<T>): void {
  const state = stateOf(instance);
  if (!Array.isArray(state.fields)) {
    state.fields = [];
  }
  const currentFields = Array.isArray(state.fields) ? state.fields : [];
  state.fields = [...currentFields, ...newFields];
}

export async function updateModelInstance<T extends BaseModel>(instance: T, options?: UpdateOptions): Promise<T> {
  if (!instance.Id) {
    throw new Error('Cannot update an instance without Id');
  }

  const changedFields = invokeSymbol<string[]>(instance, MODEL_SYMBOLS.getChangedFields);
  const relationChanges = invokeSymbol<RelationChangesCollection>(instance, MODEL_SYMBOLS.collectRelationChanges);
  const hasRelationChanges = Object.keys(relationChanges).length > 0;

  if (changedFields.length === 0 && !hasRelationChanges) {
    return instance;
  }

  const updateObj: UnknownRecord = {};
  const instanceState = stateOf(instance);
  for (const field of changedFields) {
    updateObj[field] = instanceState[field];
  }

  try {
    const repository = resolveInstanceReadRepository(instance, options);
    const updateWithValidationBypass = async (values: UnknownRecord, condition: unknown) => {
      return await runWithValidationBypass(repository, async () => await repository.update(values, condition));
    };

    const upstreamInverseFields = collectModelUpstreamInverseFields(instance.constructor as RuntimeModelCtor);
    const currentFields = Array.from(new Set<string>(['Id', 'UpdatedAt', ...upstreamInverseFields]));
    const currentEntities = await repository.search(['Id', '=', instance.Id], {
      fields: currentFields,
    });
    if (!currentEntities.length) {
      throw new Error(_t('Record with Id %s no longer exists', { scope: 'service/orm/model/model_instance' }, instance.Id));
    }
    const currentUpdatedAt = currentEntities[0].UpdatedAt as Date | undefined;

    const { processedValue, relations } = await RelationFactory.prepareForUpdate(instance.constructor as RuntimeModelCtor, updateObj, Object.keys(updateObj));

    if (hasRelationChanges) {
      RelationFactory.prepareRelationChanges(instance.constructor as RuntimeModelCtor, instance, relationChanges, relations);
    }

    const now = new Date();
    processedValue.UpdatedAt = now;
    AuditUidUtils.applyOnUpdate(processedValue as UnknownRecord);

    const condition = LockUtils.buildOptimisticLockCondition(instance.Id, currentUpdatedAt);
    const result = await updateWithValidationBypass(processedValue, condition);
    if (result.length === 0) {
      throw LockUtils.formatLockError(new Error('Update failed: This record has been modified by another user. Please reload the record and try again.'));
    }

    if (relations.oneToManyRelations.length > 0 || relations.manyToManyRelations.length > 0) {
      const relResults = await RelationFactory.batchProcessToManyRelations(instance.constructor as RuntimeModelCtor, [instance.Id], [relations]);

      const relErrors: string[] = [];
      for (const relationResult of relResults || []) {
        const errors = Array.isArray(relationResult?.errors) ? relationResult.errors : [];
        for (const error of errors) {
          relErrors.push(getRuntimeErrorMessage(error));
        }
      }
      if (relErrors.length) {
        throw new Error(`[update] relation handling failed for ${relErrors.length} item(s). Example: ${relErrors[0]}`);
      }

      if (relations.touchedCollections?.size) {
        const meta = getModelRuntimeMetadata(instance.constructor as RuntimeModelCtor);
        if (meta.computeGraph) {
          const collChanged = new Set<string>(relations.touchedCollections);
          await recomputeModelMetadata(meta, instanceState as UnknownRecord, collChanged, 'persist');
          const followUp: UnknownRecord = {};
          for (const changedField of collChanged) {
            if (changedField in instanceState && instanceState[changedField] !== currentEntities[0][changedField]) {
              followUp[changedField] = instanceState[changedField];
            }
          }
          if (Object.keys(followUp).length) {
            followUp.UpdatedAt = new Date();
            AuditUidUtils.applyOnUpdate(followUp);
            await updateWithValidationBypass(followUp, ['Id', '=', instance.Id]);
          }
        }
      }
    }

    instance.UpdatedAt = now;
    resetModelChanges(instance);

    const allChangedFields = Object.keys(updateObj);
    if (allChangedFields.length) {
      const beforeEntityForUpstream = { ...(currentEntities[0] || {}) } as UnknownRecord;
      const afterEntityForUpstream = {
        ...(currentEntities[0] || {}),
        ...processedValue,
        ...updateObj,
        Id: instance.Id,
      } as UnknownRecord;

      try {
        await triggerModelUpstream({
          childCtor: instance.constructor as RuntimeModelCtor,
          operation: 'update',
          changedFields: allChangedFields,
          beforeEntity: beforeEntityForUpstream,
          afterEntity: afterEntityForUpstream,
        });
      } catch (error) {
        if (typeof console !== 'undefined') {
          console.warn('[update] parent compute trigger failed:', error);
        }
      }
    }

    return await reloadModelInstance(instance, options);
  } catch (error) {
    if (LockUtils.isOptimisticLockError(error as Error)) {
      throw error;
    }
    throw LockUtils.formatLockError(error as Error);
  }
}

export async function deleteModelInstance(instance: BaseModel, options?: DeleteOptions): Promise<void> {
  if (!instance.Id) {
    throw new Error('Cannot delete an instance without Id');
  }

  const repository = resolveInstanceReadRepository(instance, options);
  const upstreamInverseFields = collectModelUpstreamInverseFields(instance.constructor as RuntimeModelCtor);
  const oldRows = await repository.search(['Id', '=', instance.Id], {
    fields: Array.from(new Set<string>(['Id', ...upstreamInverseFields])),
  });

  const result = await repository.delete(['Id', '=', instance.Id]);

  if (result.length === 0) {
    throw new Error(
      _t(
        'Delete failed: Record with Id %s not found or could not be deleted',
        { scope: 'service/orm/model/model_instance' },
        instance.Id
      )
    );
  }

  try {
    await triggerModelUpstream({
      childCtor: instance.constructor as RuntimeModelCtor,
      operation: 'delete',
      changedFields: [],
      beforeEntity: oldRows?.[0],
    });
  } catch (error) {
    if (typeof console !== 'undefined') {
      console.warn('[delete] parent compute trigger failed:', error);
    }
  }

  invokeSymbol<void>(instance, MODEL_SYMBOLS.resetChanges);
}

export async function loadModelInstance<T extends BaseModel>(instance: T, fields?: FieldSelection<T>, options?: SoftDeleteOptions): Promise<T> {
  if (!instance.Id) {
    throw new Error('Cannot load an instance without Id');
  }

  const fieldsToLoad: FieldSelection<T> = fields && fields.length > 0 ? fields : (['*'] as FieldSelection<T>);
  const repository = resolveInstanceReadRepository(instance, options);
  const results = await repository.search(['Id', '=', instance.Id], {
    fields: fieldsToLoad,
  });

  if (!results.length) {
    throw new Error(_t('Record with Id %s no longer exists', { scope: 'service/orm/model/model_instance' }, instance.Id));
  }

  EntityConverter.entityToModel(instance, results[0]);
  mergeFields(instance, fieldsToLoad);
  return instance;
}

export async function reloadModelInstance<T extends BaseModel>(instance: T, options?: SoftDeleteOptions): Promise<T> {
  if (!instance.Id) {
    throw new Error('Cannot reload an instance without Id');
  }

  const instanceState = stateOf(instance);
  const fieldsToReload: FieldSelection<T> =
    Array.isArray(instanceState.fields) && instanceState.fields.length > 0 ? (instanceState.fields as FieldSelection<T>) : (['*'] as FieldSelection<T>);
  await loadModelInstance(instance, fieldsToReload, options);

  resetModelChanges(instance);
  return instance;
}

export function toTransportObject(instance: BaseModel): UnknownRecord {
  return toTransportObjectImpl(instance);
}
