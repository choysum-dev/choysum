// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage, ModelMetadata, FieldMetadata, ManyToOneMetadata } from '../metadata';
import type { ModelCtor } from '../metadata/field';
import { Repository } from '../repository/repository';
import { BaseQueryCondition } from '../repository/types';
import { RelationItem, IdRelationItem, ModelRelationItem } from '../repository/types';
import { createRelationModel } from './relation_model_service_facade';
import {
  ExtractedRelations,
  type ManyToManyRelationConfig,
  ManyToOneOperation,
  OneToManyOperation,
  ManyToManyOperation,
  type OneToManyRelationConfig,
  PrepareResult,
  RelationProcessingResult,
  BatchProcessingResult,
  resolveManyToManyRelationConfig as resolveManyToManyRelationConfigShared,
  resolveOneToManyRelationConfig as resolveOneToManyRelationConfigShared,
  type RelationFieldType,
} from './types';
import type { ObjectRecord } from '../../../utils/types';

type RelationOperationBag = Partial<Record<'replace' | 'create' | 'update' | 'delete', unknown[]>>;

/**
 * Abstract base class for relation processors.
 * Defines the shared methods and properties used by all relation processors.
 */
export abstract class RelationProcessor<T extends BaseModel = BaseModel> {
  /**
   * Model metadata.
   * @protected
   */
  protected readonly metadata = MetadataStorage.instance.getModelMetadata(this.modelClass);

  /**
   * Constructor.
   * @param modelClass The model class that owns the relation.
   */
  constructor(protected readonly modelClass: ModelCtor<T> & typeof BaseModel) {}

  /**
   * Extract all supported relation payloads from an object.
   *
   * @param value Model value object.
   * @returns Extracted relation operations.
   */
  public extractRelations(value: ObjectRecord): ExtractedRelations {
    const result: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(), // NEW
    };

    if (!this.metadata.fields) return result;

    this.metadata.fields.forEach((field, fieldName) => {
      if (!field.relation) return;
      if (value[fieldName] === undefined) return;

      const fieldValue = value[fieldName];
      switch (field.type) {
        case 'ManyToOne':
          // ManyToOne values stay in processedValue directly and are not recorded here.
          break;
        case 'OneToMany': {
          if (fieldValue != null) {
            const rel = this.resolveOneToManyRelationConfig(field.relation);
            if (!rel) {
              throw new Error(`Missing OneToMany configuration: ${this.modelClass.name}.${fieldName}`);
            }
            result.oneToManyRelations.push({
              type: 'OneToMany',
              fieldName,
              targetModel: rel.targetModel(),
              inverseField: rel.inverseField,
              operations: fieldValue as OneToManyOperation['operations'],
            });
            result.touchedCollections!.add(fieldName);
          }
          break;
        }
        case 'ManyToMany': {
          if (fieldValue != null) {
            const rel = this.resolveManyToManyRelationConfig(field.relation);
            if (rel) {
              result.manyToManyRelations.push({
                type: 'ManyToMany',
                fieldName,
                joinModel: rel.joinModel(),
                targetModel: rel.targetModel(),
                joinField: rel.joinField,
                inverseJoinField: rel.inverseJoinField,
                operations: fieldValue as ManyToManyOperation['operations'],
              });
              result.touchedCollections!.add(fieldName);
            }
          }
          break;
        }
      }
    });

    if (result.touchedCollections && result.touchedCollections.size === 0) {
      // Optional: reduce serialization noise.
      // delete result.touchedCollections;
    }
    return result;
  }

  /**
   * Preprocess relation data for create operations.
   * @param value Model value object.
   * @returns Processed values plus extracted relation information.
   */
  abstract prepareForCreate(value: ObjectRecord): Promise<PrepareResult<T>>;

  /**
   * Preprocess relation data for update operations.
   * Similar to prepareForCreate, but specialized for updates.
   *
   * @param value Value object to update.
   * @param changedFields List of changed fields, if known.
   * @returns Processed values plus extracted relation information.
   */
  abstract prepareForUpdate(value: ObjectRecord, changedFields?: string[]): Promise<PrepareResult<T>>;

  /**
   * Process a relation update.
   * @param parentId Parent entity Id.
   * @param operation Relation operation payload.
   * @returns Processing result.
   */
  abstract processRelationUpdate(parentId: string, operation: ManyToOneOperation | OneToManyOperation | ManyToManyOperation): Promise<RelationProcessingResult>;

  /**
   * Batch-process relation updates.
   * Handles the same relation type for multiple parent entities more efficiently than individual calls.
   *
   * @param parentIds Parent entity Id array.
   * @param operations Relation operations paired with parentIds.
   * @returns Batch processing result.
   */
  abstract batchProcessRelationUpdate(
    parentIds: string[],
    operations: (ManyToOneOperation | OneToManyOperation | ManyToManyOperation)[]
  ): Promise<BatchProcessingResult>;

  /**
   * Group relation operations by target type and parent entity.
   * Used to optimize batch processing.
   *
   * @param parentIds Parent entity Id array.
   * @param operations Relation operation array.
   * @returns Operation map grouped by target.
   * @protected
   */
  protected groupOperationsByTarget<OpType extends OneToManyOperation | ManyToManyOperation>(
    parentIds: string[],
    operations: OpType[]
  ): Map<
    string,
    {
      config: OpType;
      parentMap: Map<string, unknown>;
    }
  > {
    if (parentIds.length !== operations.length) {
      throw new Error('Parent entity Id array length must match relation operation array length');
    }

    const groupMap = new Map<
      string,
      {
        config: OpType;
        parentMap: Map<string, unknown>;
      }
    >();

    for (let i = 0; i < operations.length; i++) {
      const operation = operations[i];
      const parentId = parentIds[i];

      // Build the grouping key.
      let groupKey: string;

      if ('targetModel' in operation && 'inverseField' in operation) {
        // OneToMany relation.
        groupKey = `${operation.fieldName}_${operation.targetModel.name}`;
      } else if ('joinModel' in operation && 'inverseJoinField' in operation) {
        // ManyToMany relation.
        groupKey = `${operation.fieldName}_${operation.joinModel.name}_${operation.targetModel.name}`;
      } else {
        // Unsupported operation type.
        continue;
      }

      // Get or create the group.
      if (!groupMap.has(groupKey)) {
        groupMap.set(groupKey, {
          config: { ...operation } as OpType,
          parentMap: new Map<string, unknown>(),
        });
      }

      // Add the parent entity Id and operation to the group.
      const group = groupMap.get(groupKey)!;
      group.parentMap.set(parentId, operation.operations);
    }

    return groupMap;
  }

  /**
   * Classify relation operations by operation type.
   * @param parentMap Mapping from parent entity Id to operation payload.
   * @returns Operations grouped by type.
   * @protected
   */
  protected classifyOperations(parentMap: Map<string, unknown>): {
    replace: Map<string, unknown[]>;
    create: Map<string, unknown[]>;
    update: Map<string, unknown[]>;
    delete: Map<string, unknown[]>;
  } {
    const result = {
      replace: new Map<string, unknown[]>(),
      create: new Map<string, unknown[]>(),
      update: new Map<string, unknown[]>(),
      delete: new Map<string, unknown[]>(),
    };

    // Classify each operation.
    for (const [parentId, operations] of parentMap.entries()) {
      // Handle array form as a direct replace.
      if (Array.isArray(operations)) {
        if (!result.replace.has(parentId)) {
          result.replace.set(parentId, []);
        }
        result.replace.get(parentId)!.push(...operations);
        continue;
      }

      // Handle object form.
      const bag = this.asOperationBag(operations);
      if (bag) {
        if (Array.isArray(bag.replace)) {
          if (!result.replace.has(parentId)) {
            result.replace.set(parentId, []);
          }
          result.replace.get(parentId)!.push(...bag.replace);
        }

        if (Array.isArray(bag.create)) {
          if (!result.create.has(parentId)) {
            result.create.set(parentId, []);
          }
          result.create.get(parentId)!.push(...bag.create);
        }

        if (Array.isArray(bag.update)) {
          if (!result.update.has(parentId)) {
            result.update.set(parentId, []);
          }
          result.update.get(parentId)!.push(...bag.update);
        }

        if (Array.isArray(bag.delete)) {
          if (!result.delete.has(parentId)) {
            result.delete.set(parentId, []);
          }
          result.delete.get(parentId)!.push(...bag.delete);
        }
      }
    }

    return result;
  }

  /**
   * Prepare entities for batch create operations.
   * @param parentMap Mapping from parent Id to entities that need to be created.
   * @param foreignKeyField Foreign-key field name.
   * @returns Prepared entity list.
   * @protected
   */
  protected prepareBatchCreateEntities(parentMap: Map<string, unknown[]>, foreignKeyField: string): ObjectRecord[] {
    const entities: ObjectRecord[] = [];

    for (const [parentId, items] of parentMap.entries()) {
      for (const item of items) {
        const entity = this.ensureObject(item);
        entity[foreignKeyField] = parentId;
        entities.push(entity);
      }
    }

    return entities;
  }

  /**
   * Delete records in batch.
   * Uses a single SQL operation when possible.
   *
   * @param repository Target-table repository.
   * @param idsToDelete Ids to delete.
   * @protected
   */
  protected async batchDeleteRecords(repository: Repository, idsToDelete: string[]): Promise<void> {
    if (!idsToDelete.length) return;

    // Pick the most suitable condition based on the number of Ids.
    const condition: BaseQueryCondition = idsToDelete.length === 1 ? ['Id', '=', idsToDelete[0]] : ['Id', 'in', idsToDelete];

    // Delete the records in batch.
    await repository.delete(condition);
  }

  /**
   * Collect Ids from operation items.
   * @param items Item array.
   * @returns Collected Ids and errors.
   * @protected
   */
  protected collectIds(items: unknown[]): { ids: string[]; errors: Error[] } {
    const ids: string[] = [];
    const errors: Error[] = [];

    for (const item of items) {
      try {
        const id = this.extractId(item);
        if (id) {
          ids.push(id);
        } else {
          errors.push(new Error(`Failed to extract Id from item: ${JSON.stringify(item)}`));
        }
      } catch (error) {
        errors.push(error instanceof Error ? error : new Error(String(error)));
      }
    }

    return { ids, errors };
  }

  /**
   * Prepare batch update operations.
   * Repository.update cannot update different records in one call, so callers must group them.
   *
   * @param items Items to update.
   * @returns Update operations grouped by Id.
   * @protected
   */
  protected prepareBatchUpdateOperations(items: unknown[]): {
    updateOps: Map<string, ObjectRecord>;
    errors: Error[];
  } {
    const updateOps = new Map<string, ObjectRecord>();
    const errors: Error[] = [];

    for (const item of items) {
      try {
        const id = this.extractId(item);
        if (!id) {
          errors.push(new Error(`Update operation is missing Id: ${JSON.stringify(item)}`));
          continue;
        }

        const entity = this.ensureObject(item);
        updateOps.set(id, entity);
      } catch (error) {
        errors.push(error instanceof Error ? error : new Error(String(error)));
      }
    }

    return { updateOps, errors };
  }

  /**
   * Build a batch-processing result.
   * @param successIds Successfully processed Ids.
   * @param errors Processing errors.
   * @param relationType Relation type.
   * @param targetModelName Target model name.
   * @returns Formatted batch-processing result.
   * @protected
   */
  protected createBatchResult(
    successIds: string[],
    errors: Error[],
    relationType: RelationFieldType,
    targetModelName: string,
    joinModelName?: string
  ): BatchProcessingResult {
    return {
      success: successIds.map(id => ({
        entityId: id,
        targetModel: targetModelName,
        joinModel: joinModelName,
      })),
      errors: errors.map(error => ({
        error,
        targetModel: targetModelName,
        joinModel: joinModelName,
      })),
      summary: {
        totalOperations: successIds.length + errors.length,
        successfulOperations: successIds.length,
        failedOperations: errors.length,
        relationType,
      },
    };
  }

  /**
   * Ensure a value is represented as an object.
   * @param value Input value.
   * @returns Normalized object.
   * @protected
   */
  protected ensureObject(value: unknown): ObjectRecord {
    // Handle string Ids.
    if (typeof value === 'string') {
      return { Id: value };
    }

    // Handle BaseModel instances.
    if (value instanceof BaseModel) {
      return value.toEntity();
    }

    // Handle plain objects.
    if (typeof value === 'object' && value !== null) {
      return value as ObjectRecord;
    }

    // Return an empty object for unsupported input types.
    return {};
  }

  /**
   * Extract an Id from supported input shapes.
   * @param value Value that may contain an Id.
   * @returns Extracted Id or null.
   * @protected
   */
  protected extractId(value: unknown): string | null {
    // Handle empty values.
    if (value === null || value === undefined) {
      return null;
    }

    // Handle string Ids.
    if (typeof value === 'string') {
      return value;
    }

    // Handle objects with an Id property.
    if (typeof value === 'object' && value !== null) {
      if ('Id' in value && typeof (value as { Id: unknown }).Id === 'string') {
        return (value as { Id: string }).Id;
      }
    }

    // Handle BaseModel instances.
    if (value instanceof BaseModel) {
      return value.Id;
    }

    return null;
  }

  protected toStringId(value: unknown): string | null {
    return typeof value === 'string' && value.length > 0 ? value : null;
  }

  /**
   * Determine whether a value looks like a model object.
   * @param value Value to inspect.
   * @returns Whether the value is model-like.
   * @protected
   */
  protected isModelLike(value: unknown): value is RelationItem<T> {
    // BaseModel instances are model-like.
    if (value instanceof BaseModel) {
      return true;
    }

    // Objects with an Id property are model-like.
    if (typeof value === 'object' && value !== null && 'Id' in value && typeof (value as { Id: unknown }).Id === 'string') {
      return true;
    }

    // Plain non-array objects are model-like too.
    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
      return true;
    }

    return false;
  }

  /**
   * Determine whether a value is an Id-style relation item.
   * @param value Value to inspect.
   * @returns Whether the value is an Id relation item.
   * @protected
   */
  protected isIdRelationItem(value: unknown): value is IdRelationItem {
    if (typeof value === 'string') {
      return true;
    }

    if (typeof value === 'object' && value !== null && 'Id' in value && typeof (value as { Id: unknown }).Id === 'string' && Object.keys(value).length === 1) {
      return true;
    }

    return false;
  }

  /**
   * Determine whether a value is a model-style relation item.
   * @param value Value to inspect.
   * @returns Whether the value is a model relation item.
   * @protected
   */
  protected isModelRelationItem(value: unknown): value is ModelRelationItem<T> {
    if (value instanceof BaseModel) {
      return true;
    }

    if (typeof value === 'object' && value !== null && !Array.isArray(value) && (!('Id' in value) || Object.keys(value).length > 1)) {
      return true;
    }

    return false;
  }

  /**
   * Get an Id or create a new entity and return its Id.
   * When a target must be created implicitly, the target model Create path is used so defaults, compute, and onchange still run.
   *
   * @param value Relation item value.
   * @param targetClass Target model class.
   * @returns Entity Id.
   * @protected
   */
  protected async getOrCreateId<R extends BaseModel>(value: unknown, targetClass: ModelCtor<R> & typeof BaseModel): Promise<string> {
    // Extract the Id from the relation item.
    const id = this.extractId(value);
    if (id) {
      return id;
    }

    // Create a new entity when the item does not already carry an Id.
    if (this.isModelRelationItem(value)) {
      try {
        return await createRelationModel(targetClass, value as ObjectRecord);
      } catch (error) {
        throw new Error(`Failed to create relation entity: ${error instanceof Error ? error.message : String(error)}`);
      }
    }

    throw new Error(`Invalid relation item: Cannot extract Id or create entity`);
  }

  /**
   * Calculate relation diffs.
   * Determines which items to add, update, keep, or remove.
   *
   * @param existingIds Existing related Ids.
   * @param newItems New related items.
   * @returns Diff calculation result.
   * @protected
   */
  protected calculateRelationDiff<T>(
    existingIds: string[],
    newItems: T[]
  ): {
    toKeep: Set<string>;
    toRemove: string[];
    toAdd: T[];
    toUpdate: Map<string, T>;
  } {
    // Track processed Ids to avoid duplicates.
    const processedIds = new Set<string>();

    // Use a set for fast membership checks against existing Ids.
    const existingIdSet = new Set(existingIds);

    // Result accumulators.
    const toKeep = new Set<string>();
    const toAdd: T[] = [];
    const toUpdate = new Map<string, T>();

    // Process incoming items.
    for (const item of newItems) {
      const id = this.extractId(item);

      // Missing Id means the item needs to be created.
      if (!id) {
        toAdd.push(item);
        continue;
      }

      // Avoid processing the same Id more than once.
      if (processedIds.has(id)) continue;
      processedIds.add(id);

      // Check whether the Id already exists in the current relation.
      if (existingIdSet.has(id)) {
        // Existing relation: keep it and track a possible update payload.
        toKeep.add(id);
        toUpdate.set(id, item);
      } else {
        // New relation: add it.
        toAdd.push(item);
      }
    }

    // Remove any existing Ids that are no longer kept.
    const toRemove = existingIds.filter(id => !toKeep.has(id));

    return { toKeep, toRemove, toAdd, toUpdate };
  }

  /**
   * Resolve the onDelete policy from the target model foreign-key field metadata.
   * This is only available on ManyToOne fields.
   */
  protected getOnDeletePolicy(
    targetClass: ModelCtor<BaseModel> & typeof BaseModel,
    foreignKeyField: string
  ): 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION' {
    const targetMeta = MetadataStorage.instance.getModelMetadata(targetClass);
    const fkFieldMeta = targetMeta.fields?.get(foreignKeyField);
    if (fkFieldMeta?.type === 'ManyToOne') {
      const rel = fkFieldMeta.relation as ManyToOneMetadata<BaseModel>;
      return rel.onDelete ?? 'SET NULL';
    }
    return 'SET NULL';
  }

  /**
   * Remove associations in batch, optionally as part of a diff update.
   * Uses the relation onDelete policy to decide how associations are removed.
   *
   * @param repository Target-table repository.
   * @param foreignKeyField Foreign-key field name.
   * @param parentIds Parent entity Id list.
   * @param targetClass Target entity class, if available.
   * @param newItemsMap New relation-item map, used for diff updates.
   * @returns Existing relation info grouped by parent Id.
   * @protected
   */
  protected async batchRemoveAssociations(
    repository: Repository,
    foreignKeyField: string,
    parentIds: string[],
    targetClass?: ModelCtor<BaseModel> & typeof BaseModel,
    newItemsMap?: Map<string, unknown[]>
  ): Promise<Map<string, { existingIds: string[]; removedIds: string[] }>> {
    // Without parent Ids there is nothing to remove.
    if (!parentIds.length) return new Map();

    // Result map: parent Id -> existing Ids plus removed Ids.
    const resultMap = new Map<string, { existingIds: string[]; removedIds: string[] }>();

    // Initialize each parent entry.
    for (const parentId of parentIds) {
      resultMap.set(parentId, { existingIds: [], removedIds: [] });
    }

    // Build the search condition.
    const condition: BaseQueryCondition = parentIds.length === 1 ? [foreignKeyField, '=', parentIds[0]] : [foreignKeyField, 'in', parentIds];

    // Load existing relations.
    const existingRecords = await repository.search(condition);

    // Return early when there are no existing relations.
    if (existingRecords.length === 0) return resultMap;

    // Group existing record Ids by parent Id.
    for (const record of existingRecords) {
      const parentId = this.toStringId(record[foreignKeyField]);
      const recordId = this.toStringId(record.Id);
      if (parentId && recordId && resultMap.has(parentId)) {
        resultMap.get(parentId)!.existingIds.push(recordId);
      }
    }

    // Without a new-item map or target class, perform the traditional full-clear operation.
    if (!newItemsMap || !targetClass) {
      // Read the onDelete policy. Only ManyToOne relations expose it.
      const onDelete = targetClass ? this.getOnDeletePolicy(targetClass, foreignKeyField) : 'SET NULL';

      // Collect all existing Ids.
      const allExistingIds = existingRecords.map(record => this.toStringId(record.Id)).filter((id): id is string => id !== null);

      // Apply the configured removal strategy.
      switch (onDelete) {
        case 'CASCADE':
          if (allExistingIds.length > 0) {
            await repository.delete(['Id', 'in', allExistingIds]);
          }
          break;
        case 'SET NULL':
          await repository.update({ [foreignKeyField]: null }, condition);
          break;
        case 'RESTRICT':
        case 'NO ACTION':
          if (allExistingIds.length > 0) {
            throw new Error(`Cannot remove associations: ${allExistingIds.length} related record(s) exist and onDelete is set to ${onDelete}`);
          }
          break;
      }

      // Record removed Ids.
      for (const [, result] of resultMap.entries()) {
        result.removedIds = [...result.existingIds];
      }

      return resultMap;
    }

    // Diff update: only remove associations that are no longer needed.
    const onDelete = this.getOnDeletePolicy(targetClass, foreignKeyField);

    // Compute the Ids to remove for each parent entity.
    const allIdsToRemove: string[] = [];

    for (const [parentId, result] of resultMap.entries()) {
      const existingIds = result.existingIds;
      const newItems = newItemsMap.get(parentId) || [];
      const newItemIds = new Set(newItems.map(item => this.extractId(item)).filter(Boolean) as string[]);

      // Remove any existing Id that is not present in the new item set.
      const idsToRemove = existingIds.filter(id => !newItemIds.has(id));

      // Record the removal set in the result map.
      result.removedIds = idsToRemove;
      allIdsToRemove.push(...idsToRemove);
    }

    // Apply the removal strategy when there are Ids to remove.
    if (allIdsToRemove.length > 0) {
      switch (onDelete) {
        case 'CASCADE':
          await repository.delete(['Id', 'in', allIdsToRemove]);
          break;
        case 'SET NULL':
          await repository.update({ [foreignKeyField]: null }, ['Id', 'in', allIdsToRemove]);
          break;
        case 'RESTRICT':
        case 'NO ACTION':
          throw new Error(`Cannot remove ${allIdsToRemove.length} association(s) because onDelete is set to ${onDelete}`);
      }
    }

    return resultMap;
  }

  // Small helper for subclasses that need to track touched collection fields.
  protected markCollectionTouched(relations: ExtractedRelations, fieldName: string) {
    (relations.touchedCollections ||= new Set()).add(fieldName);
  }

  protected resolveOneToManyRelationConfig(relation: unknown): OneToManyRelationConfig | null {
    return resolveOneToManyRelationConfigShared(relation) ?? null;
  }

  protected resolveManyToManyRelationConfig(relation: unknown): ManyToManyRelationConfig | null {
    return resolveManyToManyRelationConfigShared(relation) ?? null;
  }

  private asOperationBag(operations: unknown): RelationOperationBag | null {
    if (!operations || typeof operations !== 'object' || Array.isArray(operations)) return null;
    return operations as RelationOperationBag;
  }
}
