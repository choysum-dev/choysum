// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RelationFactory } from '../relation';
import type { ExtractedRelations } from '../relation/types';
import { AuditUidUtils } from '../utils/audit_uid';
import { TimestampUtils } from '../utils/timestamp';
import type { Insertable, FieldSelection } from '../repository/types';
import type BaseModel from './model';
import { getModelRepository } from './model_internal_facade';
import { browseManyModels, browseModel, searchModels } from './model_read_facade';
import type { RuntimeModelCtor } from './types';
import {
  getModelRuntimeMetadata,
  recomputeModelMetadata,
  triggerModelUpstream,
  triggerModelUpstreamCreateBatch,
} from './model_runtime_service_facade';
import { getRuntimeErrorMessage, runWithValidationBypass } from './model_write_helpers';
import type { UnknownRecord } from '../../../utils/types';
import { asObjectRecord } from '../../../utils/object';
import { createServiceByModel } from '../../rpc';

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

type AttachmentBindingBindResp = {
  attachmentBindingId?: string;
};

type AttachmentBindingServiceLike = {
  Bind(req: AttachmentBindingBindReq): Promise<AttachmentBindingBindResp>;
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
    throw new Error('[Create] Unable to resolve ownerModel for attachment binding.');
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
      throw new Error(`[Create] Attachment field '${fieldName}' cannot be an empty string.`);
    }
    return {
      kind: 'set',
      attachmentObjectId,
      mutationId: $choysum.xid.New(),
    };
  }

  if (Array.isArray(raw)) {
    throw new Error(`[Create] Attachment field '${fieldName}' does not support array payload.`);
  }

  const record = asObjectRecord(raw);
  if (!record) {
    throw new Error(`[Create] Attachment field '${fieldName}' must be attachmentObjectId|string|null|omitted.`);
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
    throw new Error(`[Create] Attachment field '${fieldName}' kind='set' requires attachmentObjectId.`);
  }

  throw new Error(`[Create] Attachment field '${fieldName}' has invalid payload.`);
}

function collectAttachmentWriteActions(fields: Map<string, { type?: string }>, input: UnknownRecord): Map<string, AttachmentWriteAction> {
  const actions = new Map<string, AttachmentWriteAction>();
  for (const [fieldName, fieldMeta] of fields.entries()) {
    if (!isAttachmentFieldType(fieldMeta?.type)) continue;
    if (!Object.prototype.hasOwnProperty.call(input, fieldName)) continue;
    const action = normalizeAttachmentWriteAction(input[fieldName], fieldName);
    actions.set(fieldName, action);
  }
  return actions;
}

function rewriteCreateInputForAttachments(input: UnknownRecord, actions: Map<string, AttachmentWriteAction>): UnknownRecord {
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
  if (!service || typeof service.Bind !== 'function') {
    throw new Error('[Create] document.AttachmentBinding service is unavailable.');
  }
  return service;
}

function requireAttachmentBindingId(resp: unknown, fieldName: string): string {
  const record = asObjectRecord(resp);
  const attachmentBindingId = normalizeText(record?.attachmentBindingId);
  if (!attachmentBindingId) {
    throw new Error(`[Create] Attachment field '${fieldName}' bind response missing attachmentBindingId.`);
  }
  return attachmentBindingId;
}

/**
 * Test seam for attachment-action collection during create.
 */
export const __collectCreateAttachmentWriteActionsForTest = collectAttachmentWriteActions;

/**
 * Test seam for attachment-input rewriting during create.
 */
export const __rewriteCreateInputForAttachmentsForTest = rewriteCreateInputForAttachments;

/**
 * Test seam for checking whether the attachment create pipeline is enabled.
 */
export const __isCreateAttachmentWritePipelineEnabledForTest = isAttachmentWritePipelineEnabled;

/**
 * CreateOperations owns model create flows, including defaults, relations, attachments, and compute propagation.
 */
export class CreateOperations {
  private static stripComputedFields<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>, value: Partial<Insertable<T>>): Partial<Insertable<T>> {
    const meta = getModelRuntimeMetadata(ModelCtor);
    if (!meta.computeGraph?.computeFields?.size) return value;

    const cleaned: UnknownRecord = { ...(value as UnknownRecord) };
    const virtualComputeFields = meta.computeGraph?.virtualComputeFields || new Set<string>();
    meta.computeGraph.computeFields.forEach((f: string) => {
      if (!(f in cleaned)) return;
      const handler = meta.computeHandlers?.get(f);
      const isVirtual = virtualComputeFields.has(f) || handler?.store === false;
      if (!isVirtual) delete cleaned[f];
    });
    return cleaned as Partial<Insertable<T>>;
  }

  /**
   * Creates a single model record and returns the hydrated result.
   */
  static async Create<T extends BaseModel>(ModelCtor: RuntimeModelCtor<T>, value: Partial<Insertable<T>>, returnFields?: FieldSelection<T>): Promise<T> {
    const meta = getModelRuntimeMetadata(ModelCtor);
    const ownerModel = resolveOwnerModelName(meta);

    // 1) Strip compute fields.
    value = this.stripComputedFields<T>(ModelCtor, value);

    // 2) DefaultGet — polymorphic hook (must not bypass ModelCtor.DefaultGet)
    value = (await ModelCtor.DefaultGet(value as Partial<Insertable<T & BaseModel>>)) as Partial<Insertable<T>>;

    // 2.1) Defensively strip again so DefaultGet or callers cannot reintroduce compute fields into the create payload.
    value = this.stripComputedFields<T>(ModelCtor, value as Partial<Insertable<T>>);

    // 2.2) Normalize binary/image field writes into set/clear/noop actions.
    const attachmentActions = isAttachmentWritePipelineEnabled(ownerModel)
      ? collectAttachmentWriteActions(meta.fields as Map<string, { type?: string }>, value as UnknownRecord)
      : new Map<string, AttachmentWriteAction>();
    if (attachmentActions.size) {
      value = rewriteCreateInputForAttachments(value as UnknownRecord, attachmentActions) as Partial<Insertable<T>>;
    }

    // 3) Preprocess relations.
    const { processedValue, relations } = await RelationFactory.prepareForCreate(ModelCtor, value);
    const processedRecord = processedValue as UnknownRecord;

    // 3.1) Generate Id before compute so fields such as ParentPath can resolve correctly.
    if (!processedRecord.Id) {
      processedRecord.Id = $choysum.xid.New();
    }

    // 4) Timestamps.
    const valueWithTimestamps = AuditUidUtils.addCreateUids(TimestampUtils.addTimestamps(processedValue));

    // 5) Insert.
    const repository = getModelRepository(ModelCtor);
    const ids = await runWithValidationBypass(repository, async () => {
      return await repository.create([valueWithTimestamps]);
    });
    const parentId = ids[0];

    // 6) Relation handling.
    if (relations.oneToManyRelations.length > 0 || relations.manyToManyRelations.length > 0) {
      const relResults = await RelationFactory.batchProcessToManyRelations(ModelCtor, [parentId], [relations]);

      const relErrors: string[] = [];
      for (const r of relResults || []) {
        const errs = Array.isArray(r?.errors) ? r.errors : [];
        for (const e of errs) {
          relErrors.push(getRuntimeErrorMessage(e));
        }
      }
      if (relErrors.length) {
        const first = relErrors[0];
        throw new Error(`[Create] relation handling failed for ${relErrors.length} item(s). Example: ${first}`);
      }

      // 7) Recompute parent fields triggered by touched collections during persist.
      if (relations.touchedCollections?.size) {
        try {
          const meta = getModelRuntimeMetadata(ModelCtor);
          const g = meta.computeGraph;
          if (g && relations.touchedCollections && relations.touchedCollections.size) {
            const baseSeed = new Set<string>(relations.touchedCollections);
            const affectedCompute = new Set<string>();

            // Reverse-dependency closure.
            {
              const queue: string[] = [];
              const seen = new Set<string>();
              baseSeed.forEach(src => {
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

            if (affectedCompute.size) {
              // Query the parent record and prefetch collections.
              const scalarNeeded = new Set<string>(['Id']);
              affectedCompute.forEach(f => scalarNeeded.add(f));
              affectedCompute.forEach(f => {
                const set = g.computeScalarDeps?.get(f);
                if (set) set.forEach(k => scalarNeeded.add(k));
              });

              const parentRepo = getModelRepository(ModelCtor);
              const rows = await parentRepo.search(['Id', '=', parentId], { fields: Array.from(scalarNeeded) });
              if (rows && rows.length) {
                const parentEntity = rows[0] as UnknownRecord;

                // Prefetch collection data.
                const collChainsMap = new Map<string, Set<string>>();
                affectedCompute.forEach(f => {
                  const deps = g.computeCollectionPathDeps?.get(f) || [];
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

                  if (!childCtor || !inverseField) {
                    console.warn(`[Create] collection field ${coll} is missing targetModel or inverseField; skipping prefetch`);
                    continue;
                  }

                  const childRepo = getModelRepository(childCtor);
                  const fields = new Set<string>(['Id']);
                  chains.forEach(p => p && fields.add(p));

                  const childs = await childRepo.search([inverseField, '=', parentId], { fields: Array.from(fields) });
                  parentEntity[coll] = childs || [];
                }

                // Snapshot previous values.
                const oldSnapshot: UnknownRecord = {};
                affectedCompute.forEach(k => (oldSnapshot[k] = parentEntity[k]));

                // Recompute and write back follow-up changes.
                await recomputeModelMetadata(meta, parentEntity, baseSeed, 'persist');

                const updates: UnknownRecord = {};
                let changed = false;
                affectedCompute.forEach(k => {
                  if (parentEntity[k] !== oldSnapshot[k]) {
                    updates[k] = parentEntity[k];
                    changed = true;
                  }
                });

                if (changed) {
                  updates.UpdatedAt = new Date();
                  await runWithValidationBypass(parentRepo, async () => {
                    await parentRepo.update(updates, ['Id', '=', parentId]);
                  });
                }
              }
            }
          }
        } catch (e) {
          console.error('[Create] post-relations compute (with collection prefetch) failed:', e);
          throw e;
        }
      }
    }

    // 7.5) Post-write attachment handling: create runs Bind and does not write the attachment field back to the owner row.
    if (attachmentActions.size) {
      const bindingService = resolveAttachmentBindingService();
      for (const [fieldName, action] of attachmentActions.entries()) {
        if (action.kind === 'set') {
          const bindResp = await bindingService.Bind({
            attachmentObjectId: action.attachmentObjectId,
            ownerModel,
            ownerRecordId: parentId,
            fieldName,
            mutationId: action.mutationId,
            displayFileName: action.displayFileName,
            downloadDisposition: action.downloadDisposition,
          });
          requireAttachmentBindingId(bindResp, fieldName);
          continue;
        }
      }
    }

    // 8) Child-to-parent upstream propagation through the shared model-layer entry point.
    try {
      await triggerModelUpstream({
        childCtor: ModelCtor,
        operation: 'create',
        changedFields: Object.keys(processedValue),
        afterEntity: {
          ...(valueWithTimestamps as UnknownRecord),
          Id: parentId,
        },
      });
    } catch (e) {
      if (typeof console !== 'undefined') {
        console.warn('[Create] upstream recompute failed and was ignored:', e);
      }
    }

    return (await browseModel(ModelCtor, parentId, returnFields)) as T;
  }

  /**
   * Creates multiple model records and returns the hydrated results.
   */
  static async CreateMany<T extends BaseModel>(
    ModelCtor: RuntimeModelCtor<T>,
    values: Partial<Insertable<T>>[],
    returnFields?: FieldSelection<T>
  ): Promise<T[]> {
    if (!values.length) return [];

    // 1) Strip computed fields.
    const strippedInput = values.map(v => this.stripComputedFields<T>(ModelCtor, v));

    // 2) DefaultGet — polymorphic hook (must not bypass ModelCtor.DefaultGet).
    // Sequential: overrides may do I/O; avoid unbounded concurrency and orphaned rejections.
    const preProcessed: Array<Partial<Insertable<T>>> = [];
    for (const v of strippedInput) {
      const next = await ModelCtor.DefaultGet(v as Partial<Insertable<T & BaseModel>>);
      preProcessed.push(this.stripComputedFields<T>(ModelCtor, next as Partial<Insertable<T>>));
    }

    const repository = getModelRepository(ModelCtor);

    const processedValues: Array<Partial<Insertable<T>>> = [];
    const allRelations: ExtractedRelations[] = [];

    // 3) Preprocess values and add timestamps.
    for (const value of preProcessed) {
      const { processedValue, relations } = await RelationFactory.prepareForCreate(ModelCtor, value);

      const valueWithTimestamps = AuditUidUtils.addCreateUids(TimestampUtils.addTimestamps(processedValue));
      processedValues.push(valueWithTimestamps);
      allRelations.push(relations);
    }

    // 4) Bulk insert.
    const parentIds = await runWithValidationBypass(repository, async () => {
      return await repository.create(processedValues);
    });

    // 5) Relation handling.
    if (parentIds.length > 0) {
      const relResults = await RelationFactory.batchProcessToManyRelations(ModelCtor, parentIds, allRelations);
      const relErrors: string[] = [];
      for (const r of relResults || []) {
        const errs = Array.isArray(r?.errors) ? r.errors : [];
        for (const e of errs) relErrors.push(getRuntimeErrorMessage(e));
      }
      if (relErrors.length) {
        throw new Error(`[CreateMany] relation handling failed for ${relErrors.length} item(s). Example: ${relErrors[0]}`);
      }
    }

    // 6) Child-to-parent upstream propagation in batch form.
    try {
      const createdRows = parentIds.map((id: string, i: number) => ({
        ...((processedValues[i] as UnknownRecord) || {}),
        Id: id,
      }));
      await triggerModelUpstreamCreateBatch(ModelCtor, createdRows);
    } catch (e) {
      if (typeof console !== 'undefined') {
        console.warn('[CreateMany] upstream recompute failed and was ignored:', e);
      }
    }

    if (returnFields) {
      return (await searchModels(ModelCtor, ['Id', 'in', parentIds], { fields: returnFields })) as T[];
    }
    return (await browseManyModels(ModelCtor, parentIds)) as T[];
  }
}
