// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RelationFactory } from '../relation';
import type { QueryCondition, Updateable, FieldSelection, UpdateOptions } from '../repository/types';
import type BaseModel from './model';
import { normalizePrefetchedRows } from './model_update_prefetch';
import { getModelRepository } from './model_internal_facade';
import { searchModels } from './model_read_facade';
import { resolveRepositoryWithSoftDeleteOptions } from './model_soft_delete_scope';
import type { RuntimeModelCtor } from './types';
import {
  collectModelUpstreamInverseFields,
  getModelRuntimeMetadata,
  recomputeModelMetadata,
  triggerModelDownstream,
  triggerModelUpstream,
} from './model_runtime_service_facade';
import { getRuntimeErrorMessage, runWithValidationBypass } from './model_write_helpers';
import type { UnknownRecord } from '../../../utils/types';
import { asObjectRecord } from '../../../utils/object';
import { createServiceByModel } from '../../rpc';
import { applyInverseWriteback } from '../../runtime/compute/inverse_writeback';
import { _t } from '@/core/service/i18n_binder';

type AttachmentDownloadDisposition = 'inline' | 'attachment';
type AttachmentWriteAction =
  | {
      kind: 'set';
      attachmentObjectId: string;
      mutationId: string;
      displayFileName?: string;
      downloadDisposition?: AttachmentDownloadDisposition;
    }
  | { kind: 'clear'; mutationId: string }
  | { kind: 'noop' };

type AttachmentBindingBindReq = {
  attachmentObjectId: string;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  mutationId: string;
  displayFileName?: string;
  downloadDisposition?: AttachmentDownloadDisposition;
};

type AttachmentBindingUnbindReq = {
  attachmentBindingId: string;
  mutationId: string;
  reason?: string;
};

type AttachmentBindingBindResp = {
  attachmentBindingId?: string;
};

type AttachmentBindingServiceLike = {
  Bind(req: AttachmentBindingBindReq): Promise<AttachmentBindingBindResp>;
  Unbind(req: AttachmentBindingUnbindReq): Promise<unknown>;
  Search(condition: unknown, options?: unknown): Promise<unknown>;
};

function normalizeText(value: unknown): string | undefined {
  const text = String(value ?? '').trim();
  return text === '' ? undefined : text;
}

function normalizeDownloadDisposition(value: unknown): AttachmentDownloadDisposition | undefined {
  const normalized = normalizeText(value)?.toLowerCase();
  if (normalized === 'inline' || normalized === 'attachment') {
    return normalized;
  }
  return undefined;
}

function isAttachmentFieldType(type: unknown): boolean {
  const normalized = String(type ?? '').toLowerCase();
  return normalized === 'binary' || normalized === 'image';
}

function resolveOwnerModelName(meta: { fullModelName?: string; modelName?: string; name?: string }): string {
  const model = normalizeText(meta.fullModelName) || normalizeText(meta.modelName) || normalizeText(meta.name);
  if (!model) {
    throw new Error('[Update] Unable to resolve ownerModel for attachment binding.');
  }
  return model;
}

function isAttachmentWritePipelineEnabled(ownerModel: string): boolean {
  if (ownerModel === 'document.AttachmentObject') return false;
  if (ownerModel === 'document.UploadSession') return false;
  if (ownerModel === 'document.AttachmentContent') return false;
  if (ownerModel === 'document.AttachmentUploadSession') return false;
  if (ownerModel === 'document.StoredContent') return false;
  return true;
}

function normalizeAttachmentWriteAction(raw: unknown, fieldName: string): AttachmentWriteAction {
  if (raw === undefined) return { kind: 'noop' };
  if (raw === null) {
    return {
      kind: 'clear',
      mutationId: $choysum.xid.New(),
    };
  }

  if (typeof raw === 'string') {
    const attachmentObjectId = normalizeText(raw);
    if (!attachmentObjectId) {
      throw new Error(`[Update] Attachment field '${fieldName}' cannot be an empty string.`);
    }
    return {
      kind: 'set',
      attachmentObjectId,
      mutationId: $choysum.xid.New(),
    };
  }

  if (Array.isArray(raw)) {
    throw new Error(`[Update] Attachment field '${fieldName}' does not support array payload.`);
  }

  const record = asObjectRecord(raw);
  if (!record) {
    throw new Error(`[Update] Attachment field '${fieldName}' must be attachmentObjectId|string|null|omitted.`);
  }

  const kind = normalizeText(record.kind)?.toLowerCase();
  if (kind === 'noop') return { kind: 'noop' };
  if (kind === 'clear') {
    return {
      kind: 'clear',
      mutationId: normalizeText(record.mutationId) || $choysum.xid.New(),
    };
  }

  const attachmentObjectId = normalizeText(record.attachmentObjectId);
  if (attachmentObjectId) {
    return {
      kind: 'set',
      attachmentObjectId,
      mutationId: normalizeText(record.mutationId) || $choysum.xid.New(),
      displayFileName:
        normalizeText(record.displayFileName) ||
        normalizeText(record.displayName) ||
        normalizeText(record.fileName) ||
        normalizeText(record.originalFileName) ||
        normalizeText(record.proposedFileName),
      downloadDisposition: normalizeDownloadDisposition(record.downloadDisposition),
    };
  }

  if (kind === 'set') {
    throw new Error(`[Update] Attachment field '${fieldName}' kind='set' requires attachmentObjectId.`);
  }

  throw new Error(`[Update] Attachment field '${fieldName}' has invalid payload.`);
}

function collectAttachmentWriteActions(fields: Map<string, { type?: string }>, input: UnknownRecord): Map<string, AttachmentWriteAction> {
  const actions = new Map<string, AttachmentWriteAction>();
  for (const [fieldName, fieldMeta] of fields.entries()) {
    if (!isAttachmentFieldType(fieldMeta?.type)) continue;
    if (!Object.prototype.hasOwnProperty.call(input, fieldName)) continue;
    actions.set(fieldName, normalizeAttachmentWriteAction(input[fieldName], fieldName));
  }
  return actions;
}

function rewriteUpdateInputForAttachments(input: UnknownRecord, actions: Map<string, AttachmentWriteAction>): UnknownRecord {
  const rewritten: UnknownRecord = { ...input };
  for (const [fieldName, action] of actions.entries()) {
    if (action.kind === 'set' || action.kind === 'clear') {
      rewritten[fieldName] = null;
      continue;
    }
    delete rewritten[fieldName];
  }
  return rewritten;
}

function resolveAttachmentBindingService(): AttachmentBindingServiceLike {
  const service = createServiceByModel('document.AttachmentBinding') as unknown as AttachmentBindingServiceLike;
  if (!service || typeof service.Bind !== 'function' || typeof service.Unbind !== 'function' || typeof service.Search !== 'function') {
    throw new Error('[Update] document.AttachmentBinding service is unavailable.');
  }
  return service;
}

function requireAttachmentBindingId(resp: unknown, fieldName: string): string {
  const record = asObjectRecord(resp);
  const attachmentBindingId = normalizeText(record?.attachmentBindingId);
  if (!attachmentBindingId) {
    throw new Error(`[Update] Attachment field '${fieldName}' bind response missing attachmentBindingId.`);
  }
  return attachmentBindingId;
}

function isNotFoundLikeError(err: unknown): boolean {
  const record = asObjectRecord(err);
  const codeText = String(record?.grpcCode ?? record?.code ?? '')
    .trim()
    .toLowerCase();
  if (codeText === '5' || codeText === 'not_found' || codeText === 'notfound') return true;
  const message = String(record?.message ?? (err instanceof Error ? err.message : '')).toLowerCase();
  return message.includes('not found');
}

async function findActiveBindingId(
  service: AttachmentBindingServiceLike,
  ownerModel: string,
  ownerRecordId: string,
  fieldName: string
): Promise<string | undefined> {
  const rows = await service.Search(
    {
      And: [
        ['OwnerModel', '=', ownerModel],
        ['OwnerRecordId', '=', ownerRecordId],
        ['FieldName', '=', fieldName],
        ['Status', '=', 'active'],
      ],
    } as any,
    {
      limit: 1,
      fields: ['Id'],
    } as any
  );

  const first = Array.isArray(rows) ? rows[0] : undefined;
  const record = asObjectRecord(first);
  return normalizeText(record?.Id);
}

async function clearAttachmentBinding(
  service: AttachmentBindingServiceLike,
  ownerModel: string,
  ownerRecordId: string,
  fieldName: string,
  mutationId: string,
  candidateBindingId: unknown
): Promise<void> {
  const directBindingId = normalizeText(candidateBindingId);
  if (directBindingId) {
    try {
      await service.Unbind({
        attachmentBindingId: directBindingId,
        mutationId,
        reason: 'clear',
      });
      return;
    } catch (err) {
      if (!isNotFoundLikeError(err)) {
        throw err;
      }
    }
  }

  const activeBindingId = await findActiveBindingId(service, ownerModel, ownerRecordId, fieldName);
  if (!activeBindingId || activeBindingId === directBindingId) return;

  await service.Unbind({
    attachmentBindingId: activeBindingId,
    mutationId,
    reason: 'clear',
  });
}

function getScaleFieldName(spec: unknown): string | undefined {
  const record = asObjectRecord(spec);
  const scaleField = record?.scaleField;
  return typeof scaleField === 'string' && scaleField ? scaleField : undefined;
}

export const __collectUpdateAttachmentWriteActionsForTest = collectAttachmentWriteActions;

/**
 * Test seam for attachment-input rewriting during update.
 */
export const __rewriteUpdateInputForAttachmentsForTest = rewriteUpdateInputForAttachments;

/**
 * Test seam for attachment binding cleanup during update.
 */
export const __clearAttachmentBindingForTest = clearAttachmentBinding;

/**
 * Test seam for checking whether the attachment update pipeline is enabled.
 */
export const __isUpdateAttachmentWritePipelineEnabledForTest = isAttachmentWritePipelineEnabled;

/**
 * UpdateOperations owns model update flows, including attachments, relation writes, and compute propagation.
 */
export class UpdateOperations {
  private static resolveRepository<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>, options?: UpdateOptions) {
    return resolveRepositoryWithSoftDeleteOptions(ModelCtor, options);
  }

  /**
   * Updates all records matching a condition.
   */
  static async Update<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    condition: QueryCondition<T>,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>[]> {
    const meta = getModelRuntimeMetadata(ModelCtor);
    const ownerModel = resolveOwnerModelName(meta);
    const repository = UpdateOperations.resolveRepository(ModelCtor, options);

    // 0) Rewrite behavior-field assignments through inverse handlers before raw write planning.
    values = (await applyInverseWriteback(meta, values as UnknownRecord)) as Partial<Updateable<T>>;

    // 1) Strip compute fields so callers cannot write compute fields directly.
    if (meta.computeGraph?.computeFields?.size) {
      const cleaned: UnknownRecord = { ...(values as UnknownRecord) };
      const virtualComputeFields = meta.computeGraph?.virtualComputeFields || new Set<string>();
      let removed = 0;
      meta.computeGraph.computeFields.forEach((f: string) => {
        if (f in cleaned) {
          const handler = meta.computeHandlers?.get(f);
          const isVirtual = virtualComputeFields.has(f) || handler?.store === false;
          if (isVirtual) return;
          delete cleaned[f];
          removed++;
        }
      });
      if (removed) values = cleaned as Partial<Updateable<T>>;
    }

    // 1.5) Normalize binary/image field writes into set/clear/noop actions.
    const attachmentActions = isAttachmentWritePipelineEnabled(ownerModel)
      ? collectAttachmentWriteActions(meta.fields as Map<string, { type?: string }>, values as UnknownRecord)
      : new Map<string, AttachmentWriteAction>();
    if (attachmentActions.size) {
      values = rewriteUpdateInputForAttachments(values as UnknownRecord, attachmentActions) as Partial<Updateable<T>>;
    }

    // 2) Preprocess relations.
    const { processedValue, relations } = await RelationFactory.prepareForUpdate(ModelCtor, values);
    const baseChangedInitial = Object.keys(processedValue);
    const touchedCollections: Set<string> = relations.touchedCollections || new Set<string>();

    // 2.1) Restriction: multi-row updates do not support collection-field mutations.
    if (touchedCollections.size) {
      const count = await repository.count(condition);
      if (count > 1) {
        throw new Error(
          'static Update: collection field updates (OneToMany / ManyToMany) are not supported for multi-row updates. Split the condition or update rows one by one.'
        );
      }
    }

    // 3) Compute the affected compute-field set.
    const g = meta.computeGraph;
    const affectedCompute = new Set<string>();
    if (g) {
      const triggerSeed = new Set<string>([...touchedCollections]);
      const queue: string[] = [];
      const seen = new Set<string>();
      triggerSeed.forEach(src => {
        const arr = g.fastReverseDeps.get(src);
        if (arr)
          arr.forEach(cf => {
            if (!seen.has(cf)) {
              seen.add(cf);
              queue.push(cf);
            }
          });
      });
      for (let i = 0; i < queue.length; i++) {
        const cf = queue[i];
        affectedCompute.add(cf);
        const next = g.fastReverseDeps.get(cf);
        if (next)
          next.forEach(n => {
            if (!seen.has(n)) {
              seen.add(n);
              queue.push(n);
            }
          });
      }
    }

    // 4) Build the query-field set and read the rows under lock.
    const queryFields = new Set<string>(['Id', 'UpdatedAt']);
    const upstreamInverseFields = new Set<string>(collectModelUpstreamInverseFields(ModelCtor));

    // Helper: add a field to queryFields and include its scaleField when the field is decimal.
    const addFieldWithScale = (fieldName: string | undefined, set: Set<string>) => {
      if (!fieldName) return;
      set.add(fieldName);
      const fm = meta.fields.get(fieldName);
      if (fm?.type === 'decimal') {
        const scaleField = getScaleFieldName(fm.column || {});
        if (scaleField) {
          set.add(scaleField);
        }
      }
    };

    baseChangedInitial.forEach(f => addFieldWithScale(f, queryFields));
    upstreamInverseFields.forEach(f => queryFields.add(f));
    if (g && affectedCompute.size) {
      affectedCompute.forEach(cf => {
        // The compute field itself.
        addFieldWithScale(cf, queryFields);
        // Scalar dependencies.
        g.computeScalarDeps?.get(cf)?.forEach(dep => addFieldWithScale(dep, queryFields));
      });
    }

    const locked = await repository.search(condition, {
      fields: Array.from(queryFields),
      forUpdate: true,
    });
    if (!locked.length) return [];
    const attachmentBindingService = attachmentActions.size ? resolveAttachmentBindingService() : undefined;

    // 5) Update rows one by one.
    const updatedIds: string[] = [];

    for (const row of locked) {
      const entityId = typeof row.Id === 'string' ? row.Id : '';
      if (!entityId) {
        throw new Error('[Update] Read an invalid record Id.');
      }
      const beforeEntityForUpstream = { ...row } as UnknownRecord;

      // 5.1) Update scalar fields.
      const entityObj: UnknownRecord = { ...row }; // Start from the locked row so scaleField companions are available.
      const scalarUpdate: UnknownRecord = {};
      for (const k of baseChangedInitial) {
        if (attachmentActions.has(k)) continue;
        if (Object.prototype.hasOwnProperty.call(processedValue, k)) {
          scalarUpdate[k] = (processedValue as UnknownRecord)[k];
          entityObj[k] = (processedValue as UnknownRecord)[k];
        }
      }

      // Helper: include scaleField companions for decimal updates.
      const addScaleForUpdates = (fieldSet: Set<string>, source: UnknownRecord, target: UnknownRecord) => {
        fieldSet.forEach(fieldName => {
          const fm = meta.fields.get(fieldName);
          if (fm?.type !== 'decimal') return;
          const scaleField = getScaleFieldName(fm.column || {});
          if (scaleField) {
            if (source[scaleField] !== undefined && !(scaleField in target)) {
              target[scaleField] = source[scaleField];
            }
          }
        });
      };

      // Include scale-field companions for scalar updates.
      addScaleForUpdates(new Set(baseChangedInitial), entityObj, scalarUpdate);

      scalarUpdate.UpdatedAt = new Date();

      const didScalarUpdate = Object.keys(scalarUpdate).length > 1;
      if (didScalarUpdate) {
        await repository.update(scalarUpdate, ['Id', '=', entityId]);
      }

      // 5.2) Collection relation handling.
      if (touchedCollections.size && (relations.oneToManyRelations.length || relations.manyToManyRelations.length)) {
        const relResults = await RelationFactory.batchProcessToManyRelations(ModelCtor, [entityId], [relations]);
        const relErrors: string[] = [];
        for (const r of relResults || []) {
          const errs = Array.isArray(r?.errors) ? r.errors : [];
          for (const e of errs) {
            relErrors.push(getRuntimeErrorMessage(e));
          }
        }
        if (relErrors.length) {
          throw new Error(`[Update] relation handling failed for ${relErrors.length} item(s). Example: ${relErrors[0]}`);
        }
      }

      // 5.3) Recompute compute fields triggered by collection changes.
      // Scalar persistence recomputation is still handled by Repository.
      const g2 = meta.computeGraph;
      if (g2 && touchedCollections.size) {
        // entityObj already contains row plus the base update, including scale fields.
        // If compute fields depend on collection paths, prefetch collection data.
        if (touchedCollections.size) {
          const collChainsMap = new Map<string, Set<string>>();
          affectedCompute.forEach(cf => {
            const deps = g2.computeCollectionPathDeps?.get(cf) || [];
            for (const d of deps) {
              if (!collChainsMap.has(d.collection)) collChainsMap.set(d.collection, new Set<string>());
              collChainsMap.get(d.collection)!.add(d.chain && d.chain.length ? d.chain.join('.') : 'Id');
            }
          });

          for (const [coll, chains] of collChainsMap.entries()) {
            const fieldMeta = meta.fields.get(coll);
            const rel = asObjectRecord(fieldMeta?.relation);
            const targetModel = rel?.targetModel;
            const childCtor = typeof targetModel === 'function' ? targetModel() : undefined;
            const inverseFieldValue = rel?.inverseField ?? rel?.fkField ?? rel?.foreignKey ?? rel?.refField;
            const inverseField = typeof inverseFieldValue === 'string' ? inverseFieldValue : undefined;
            if (!childCtor || !inverseField) continue;

            const childRepo = getModelRepository(childCtor);
            const select = new Set<string>(['Id']);
            chains.forEach(p => p && select.add(p));
            const rows2 = await childRepo.search([inverseField, '=', entityId], {
              fields: Array.from(select),
            });
            entityObj[coll] = normalizePrefetchedRows(rows2 as UnknownRecord[]);
          }
        }

        const baseSeed = new Set<string>([...touchedCollections]);
        const oldSnap: UnknownRecord = {};
        affectedCompute.forEach(k => (oldSnap[k] = (row as UnknownRecord)[k]));

        await recomputeModelMetadata(meta, entityObj, baseSeed, 'persist');

        // Write back compute changes, including decimal scale fields.
        const followUp: UnknownRecord = {};
        for (const f of affectedCompute) {
          if (f in entityObj && entityObj[f] !== oldSnap[f]) {
            followUp[f] = entityObj[f];
          }
        }
        if (Object.keys(followUp).length) {
          addScaleForUpdates(affectedCompute, entityObj, followUp);
          await runWithValidationBypass(repository, async () => {
            await repository.update(followUp, ['Id', '=', entityId]);
          });
        }
      }

      // 5.4) Post-write attachment handling: set binds, clear unbinds, and the owner row does not persist the attachment field.
      if (attachmentActions.size) {
        const bindingService = attachmentBindingService!;
        let touchedAttachment = false;

        for (const [fieldName, action] of attachmentActions.entries()) {
          if (action.kind === 'set') {
            const bindResp = await bindingService.Bind({
              attachmentObjectId: action.attachmentObjectId,
              ownerModel,
              ownerRecordId: entityId,
              fieldName,
              mutationId: action.mutationId,
              displayFileName: action.displayFileName,
              downloadDisposition: action.downloadDisposition,
            });
            const attachmentBindingId = requireAttachmentBindingId(bindResp, fieldName);
            entityObj[fieldName] = attachmentBindingId;
            touchedAttachment = true;
            continue;
          }

          if (action.kind === 'clear') {
            await clearAttachmentBinding(bindingService, ownerModel, entityId, fieldName, action.mutationId, (row as UnknownRecord)[fieldName]);
            entityObj[fieldName] = null;
            touchedAttachment = true;
          }
        }

        if (touchedAttachment && !didScalarUpdate) {
          await runWithValidationBypass(repository, async () => {
            await repository.update({ UpdatedAt: new Date() }, ['Id', '=', entityId]);
          });
        }
      }

      // 5.5) Trigger cascade hooks without blocking the update result.
      try {
        if (baseChangedInitial.length) {
          await triggerModelUpstream({
            childCtor: ModelCtor,
            operation: 'update',
            changedFields: baseChangedInitial,
            beforeEntity: beforeEntityForUpstream,
            afterEntity: entityObj,
          });
        }
      } catch (e) {
        if (typeof console !== 'undefined') {
          console.warn('[Update] upstream recompute failed and was ignored:', e);
        }
      }

      try {
        if (baseChangedInitial.length) {
          await triggerModelDownstream(ModelCtor, baseChangedInitial, String(entityId));
        }
      } catch (e) {
        if (typeof console !== 'undefined') {
          console.warn('[Update] downstream cascade recompute failed and was ignored:', e);
        }
      }

      updatedIds.push(entityId);
    }

    if (returnFields && updatedIds.length > 0) {
      const searchOptions: {
        fields: FieldSelection<T>;
        withDeleted?: boolean;
        onlyDeleted?: boolean;
      } = { fields: returnFields };
      if (options?.withDeleted) searchOptions.withDeleted = true;
      if (options?.onlyDeleted) searchOptions.onlyDeleted = true;
      return (await searchModels(ModelCtor, ['Id', 'in', updatedIds], searchOptions)) as unknown as Partial<T>[];
    }

    return updatedIds.map((id: string) => ({ Id: id }) as unknown as Partial<T>);
  }

  /**
   * Updates a single record by Id.
   */
  static async UpdateById<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    id: string,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Partial<T>> {
    const results = await UpdateOperations.Update<T>(ModelCtor, ['Id', '=', id] as QueryCondition<T>, values, returnFields, options);
    if (results.length === 0) {
      throw new Error(_t('Update failed: Record %s not found.', { scope: 'service/orm/model/model_update' }, id));
    }
    return results[0];
  }
}
