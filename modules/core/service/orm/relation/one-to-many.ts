// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { RepositoryFactory } from '../repository/repository_factory';
import { RelationProcessor } from './processor';
import { OneToManyOperation, PrepareResult, RelationProcessingResult, ExtractedRelations, BatchProcessingResult } from './types';
import { BaseQueryCondition, RelationOperations } from '../repository/types';
import { MetadataStorage } from '../metadata';
import { Repository } from '../repository/repository';
import type { ModelCtor } from '../metadata/field';
import { createRelationModel, updateRelationModelById } from './relation_model_service_facade';
import type { ObjectRecord } from '../../../utils/types';

/**
 * OneToMany relation processor.
 * Handles create and update flows for one-to-many relations.
 */
export class OneToManyProcessor<T extends BaseModel = BaseModel> extends RelationProcessor<T> {
  /**
   * Preprocess OneToMany relation data for create operations.
   * OneToMany relations are applied after the parent entity is created, so this step only extracts relation info.
   */
  public async prepareForCreate(value: ObjectRecord): Promise<PrepareResult<T>> {
    const processedValue = { ...value };

    // Extract ToMany relations.
    const relations = this.extractRelations(value);

    // Remove O2M fields from the write path.
    this.metadata.fields?.forEach((field, fieldName) => {
      if (field.type === 'OneToMany') {
        delete processedValue[fieldName];
      }
    });

    return {
      processedValue,
      relations: {
        oneToManyRelations: relations.oneToManyRelations,
        manyToManyRelations: [],
        touchedCollections: relations.touchedCollections, // Pass through.
      },
    };
  }

  /**
   * Preprocess OneToMany relation data for update operations.
   */
  public async prepareForUpdate(value: ObjectRecord, changedFields?: string[]): Promise<PrepareResult<T>> {
    const processedValue = { ...value };
    const fieldsToProcess = changedFields || Object.keys(value);

    const relations: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(), // NEW
    };

    this.metadata.fields?.forEach((field, fieldName) => {
      if (!fieldsToProcess.includes(fieldName)) return;
      if (field.type !== 'OneToMany') return;
      if (!(fieldName in value)) return;

      const fieldValue = value[fieldName];
      delete processedValue[fieldName];

      // Only participate in the write path when the relation config is complete.
      const rel = this.resolveOneToManyRelationConfig(field.relation);
      if (fieldValue !== null && fieldValue !== undefined && rel) {
        relations.oneToManyRelations.push({
          type: 'OneToMany',
          fieldName,
          targetModel: rel.targetModel(),
          inverseField: rel.inverseField,
          operations: fieldValue as OneToManyOperation['operations'],
        });
        relations.touchedCollections!.add(fieldName); // Mark the collection field as touched.
      }
    });

    if (relations.touchedCollections && relations.touchedCollections.size === 0) {
      // Optional: keep the empty set or delete relations.touchedCollections.
    }

    return { processedValue, relations };
  }

  /**
   * Process a OneToMany relation update.
   * Supports creating, updating, deleting child entities, and replacing the entire collection.
   */
  public async processRelationUpdate(parentId: string, operation: OneToManyOperation): Promise<RelationProcessingResult> {
    if (operation.type !== 'OneToMany') {
      throw new Error(`Expected a OneToMany operation, but received ${operation.type}`);
    }

    try {
      const { targetModel, inverseField, operations } = operation;
      const targetRepo = RepositoryFactory.getRepository(targetModel);
      const affectedIds: string[] = [];
      const errors: Error[] = [];

      // Array mode: replace.
      if (Array.isArray(operations)) {
        const diffResult = await this.removeExistingRelationsWithDiff(targetRepo, targetModel, inverseField, parentId, operations);
        const existingIdSet = new Set(diffResult.existingIds);

        for (const item of operations) {
          try {
            // Strip compute fields so clients cannot override them.
            const itemObj = this.stripChildComputeFields(targetModel, item);
            const id = this.extractId(item);
            const processedItem = { ...itemObj, [inverseField]: parentId };

            if (id && existingIdSet.has(id)) {
              const { Id, ...dataToUpdate } = processedItem;
              const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
              if (await updateRelationModelById(targetModel, id, cleanedUpdate)) affectedIds.push(id);
            } else if (id) {
              const { Id, ...dataToUpdate } = processedItem;
              const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
              if (await updateRelationModelById(targetModel, id, cleanedUpdate)) affectedIds.push(id);
            } else {
              const createdId = await createRelationModel(targetModel, processedItem);
              if (createdId) affectedIds.push(createdId);
            }
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process collection item: ${String(e)}`));
          }
        }

        return {
          affectedCount: affectedIds.length,
          entityIds: affectedIds,
          errors,
          targetModel,
          relationType: 'OneToMany',
        };
      }

      // Object mode: replace/delete/create/update.
      const relationOps = operations as RelationOperations<BaseModel>;

      // replace
      if (relationOps.replace) {
        const diffResult = await this.removeExistingRelationsWithDiff(targetRepo, targetModel, inverseField, parentId, relationOps.replace);
        const existingIdSet = new Set(diffResult.existingIds);

        for (const item of relationOps.replace) {
          try {
            const itemObj = this.stripChildComputeFields(targetModel, item);
            const id = this.extractId(item);
            const processedItem = { ...itemObj, [inverseField]: parentId };

            if (id && existingIdSet.has(id)) {
              const { Id, ...dataToUpdate } = processedItem;
              const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
              if (await updateRelationModelById(targetModel, id, cleanedUpdate)) affectedIds.push(id);
            } else if (id) {
              const { Id, ...dataToUpdate } = processedItem;
              const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
              if (await updateRelationModelById(targetModel, id, cleanedUpdate)) affectedIds.push(id);
            } else {
              const createdId = await createRelationModel(targetModel, processedItem);
              if (createdId) affectedIds.push(createdId);
            }
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process replace item: ${String(e)}`));
          }
        }
      }

      // delete
      if (relationOps.delete) {
        for (const item of relationOps.delete) {
          try {
            const id = this.extractId(item);
            if (!id) {
              errors.push(new Error('Could not extract Id from delete item'));
              continue;
            }

            const result = await this.applySingleItemDeleteStrategy(targetRepo, targetModel, inverseField, parentId, id);
            if (result.success) {
              affectedIds.push(id);
            } else if (result.error) {
              errors.push(result.error);
            }
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process delete item: ${String(e)}`));
          }
        }
      }

      // update
      if (relationOps.update) {
        for (const item of relationOps.update) {
          try {
            const id = this.extractId(item);
            if (!id) {
              errors.push(new Error('Could not extract Id from update item'));
              continue;
            }

            const existing = await targetRepo.search({
              And: [
                ['Id', '=', id],
                [inverseField, '=', parentId],
              ],
            });
            if (existing.length === 0) {
              errors.push(new Error(`Item does not exist or does not belong to the parent entity: Id=${id}`));
              continue;
            }

            const itemObj = this.stripChildComputeFields(targetModel, item);
            const { Id, ...dataToUpdate } = itemObj;
            const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
            if (await updateRelationModelById(targetModel, id, cleanedUpdate)) affectedIds.push(id);
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process update item: ${String(e)}`));
          }
        }
      }

      // create
      if (relationOps.create) {
        for (const item of relationOps.create) {
          try {
            if (this.extractId(item)) {
              errors.push(new Error(`Create item must not include Id: ${JSON.stringify(item)}`));
              continue;
            }
            const itemObj = this.stripChildComputeFields(targetModel, item);
            const processedItem = { ...itemObj, [inverseField]: parentId };
            const createdId = await createRelationModel(targetModel, processedItem);
            if (createdId) affectedIds.push(createdId);
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process create item: ${String(e)}`));
          }
        }
      }

      return {
        affectedCount: affectedIds.length,
        entityIds: affectedIds,
        errors,
        targetModel,
        relationType: 'OneToMany',
      };
    } catch (error) {
      return {
        affectedCount: 0,
        entityIds: [],
        errors: [error instanceof Error ? error : new Error(String(error))],
        targetModel: operation.targetModel,
        relationType: 'OneToMany',
      };
    }
  }

  /**
   * Batch-process OneToMany relation updates.
   */
  public async batchProcessRelationUpdate(parentIds: string[], operations: OneToManyOperation[]): Promise<BatchProcessingResult> {
    if (parentIds.length !== operations.length) {
      throw new Error('Parent entity Id array length must match relation operation array length');
    }
    for (const op of operations) {
      if (op.type !== 'OneToMany') throw new Error(`Expected a OneToMany operation, but received ${op.type}`);
    }

    const groupedOps = this.groupOperationsByTarget(parentIds, operations);
    const allSuccessIds: string[] = [];
    const allErrors: Error[] = [];

    try {
      for (const [, group] of groupedOps.entries()) {
        const { config, parentMap } = group;
        const { targetModel, inverseField } = config;

        const targetRepo = RepositoryFactory.getRepository(targetModel);
        const classifiedOps = this.classifyOperations(parentMap);

        // replace
        const replaceOps = classifiedOps.replace;
        if (replaceOps.size > 0) {
          const parentIdsToReplace = Array.from(replaceOps.keys());
          const diffResults = await this.batchRemoveAssociationsWithDiff(targetRepo, targetModel, inverseField, parentIdsToReplace, replaceOps);

          const entitiesToUpdate = new Map<string, ObjectRecord>();
          const toCreateList: Array<{ parentId: string; payload: ObjectRecord }> = [];

          for (const [parentId, items] of replaceOps.entries()) {
            const existingIds = diffResults.get(parentId)?.existingIds || new Set<string>();

            for (const item of items) {
              try {
                const itemObj = this.stripChildComputeFields(targetModel, item);
                const id = this.extractId(item);

                if (id && existingIds.has(id)) {
                  entitiesToUpdate.set(id, { ...itemObj, [inverseField]: parentId });
                } else if (id) {
                  entitiesToUpdate.set(id, { ...itemObj, [inverseField]: parentId });
                } else {
                  toCreateList.push({ parentId, payload: { ...itemObj, [inverseField]: parentId } });
                }
              } catch (e) {
                allErrors.push(e instanceof Error ? e : new Error(String(e)));
              }
            }
          }

          // Create one by one instead of using bulk repository.create.
          if (toCreateList.length > 0) {
            for (const it of toCreateList) {
              try {
                const createdId = await createRelationModel(targetModel, it.payload);
                if (createdId) allSuccessIds.push(createdId);
              } catch (e) {
                allErrors.push(e instanceof Error ? e : new Error(String(e)));
              }
            }
          }

          for (const [id, updateData] of entitiesToUpdate.entries()) {
            try {
              const { Id, ...dataToUpdate } = updateData;
              const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
              if (await updateRelationModelById(targetModel, id, cleanedUpdate)) allSuccessIds.push(id);
            } catch (e) {
              allErrors.push(e instanceof Error ? e : new Error(String(e)));
            }
          }
        }

        // delete
        const deleteOps = classifiedOps.delete;
        if (deleteOps.size > 0) {
          const deleteItems: { id: string; parentId: string }[] = [];
          for (const [parentId, items] of deleteOps.entries()) {
            for (const item of items) {
              const id = this.extractId(item);
              if (id) deleteItems.push({ id, parentId });
              else allErrors.push(new Error('Could not extract Id from delete item'));
            }
          }

          if (deleteItems.length > 0) {
            const strategyGroups = await this.groupItemsByDeleteStrategy(targetRepo, targetModel, inverseField, deleteItems);
            const deleteResult = await this.executeDeleteStrategies(targetRepo, inverseField, strategyGroups);
            allSuccessIds.push(...deleteResult.successIds);
            allErrors.push(...deleteResult.errors);
          }
        }

        // update
        const updateOps = classifiedOps.update;
        if (updateOps.size > 0) {
          const entitiesToUpdate = new Map<string, ObjectRecord>();

          for (const [parentId, items] of updateOps.entries()) {
            for (const item of items) {
              try {
                const id = this.extractId(item);
                if (!id) {
                  allErrors.push(new Error('Could not extract Id from update item'));
                  continue;
                }

                const existing = await targetRepo.search({
                  And: [
                    ['Id', '=', id],
                    [inverseField, '=', parentId],
                  ],
                });
                if (existing.length === 0) {
                  allErrors.push(new Error(`Item does not exist or does not belong to the parent entity: Id=${id}`));
                  continue;
                }

                const itemObj = this.stripChildComputeFields(targetModel, item);
                entitiesToUpdate.set(id, { ...itemObj });
              } catch (e) {
                allErrors.push(e instanceof Error ? e : new Error(String(e)));
              }
            }
          }

          for (const [id, updateData] of entitiesToUpdate.entries()) {
            try {
              const { Id, ...dataToUpdate } = updateData;
              const cleanedUpdate = this.stripChildComputeFields(targetModel, dataToUpdate);
              if (await updateRelationModelById(targetModel, id, cleanedUpdate)) allSuccessIds.push(id);
            } catch (e) {
              allErrors.push(e instanceof Error ? e : new Error(String(e)));
            }
          }
        }

        // create
        const createOps = classifiedOps.create;
        if (createOps.size > 0) {
          for (const [parentId, items] of createOps.entries()) {
            for (const item of items) {
              try {
                if (this.extractId(item)) {
                  allErrors.push(new Error(`Create item must not include Id: ${JSON.stringify(item)}`));
                  continue;
                }
                const itemObj = this.stripChildComputeFields(targetModel, item);
                const payload = { ...itemObj, [inverseField]: parentId };
                const createdId = await createRelationModel(targetModel, payload);
                if (createdId) allSuccessIds.push(createdId);
              } catch (e) {
                allErrors.push(e instanceof Error ? e : new Error(String(e)));
              }
            }
          }
        }
      }

      return this.createBatchResult(allSuccessIds, allErrors, 'OneToMany', operations[0]?.targetModel?.name || 'Unknown target');
    } catch (error) {
      return {
        success: [],
        errors: [
          {
            error: error instanceof Error ? error : new Error(String(error)),
            targetModel: operations[0]?.targetModel?.name || 'Unknown target',
          },
        ],
        summary: {
          totalOperations: parentIds.length,
          successfulOperations: 0,
          failedOperations: parentIds.length,
          relationType: 'OneToMany',
        },
      };
    }
  }

  /**
   * Diff update: remove only associations that are no longer needed instead of clearing and rebuilding everything.
   */
  private async removeExistingRelationsWithDiff(
    repository: Repository,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    inverseField: string,
    parentId: string,
    newItems: unknown[]
  ): Promise<{ existingIds: string[]; removedIds: string[] }> {
    const existingRecords = await repository.search([inverseField, '=', parentId]);
    const existingIds = existingRecords.map(record => String(record.Id ?? '')).filter(Boolean);

    if (!newItems || newItems.length === 0) {
      await this.applyDeleteStrategy(repository, targetModel, inverseField, parentId);
      return { existingIds, removedIds: [...existingIds] };
    }

    const newItemIds = new Set(newItems.map(item => this.extractId(item)).filter(Boolean));
    const idsToDisassociate = existingIds.filter((id: string) => !newItemIds.has(id));

    if (idsToDisassociate.length > 0) {
      const onDelete = this.getOnDeletePolicy(targetModel, inverseField);
      switch (onDelete) {
        case 'CASCADE':
          await repository.delete(['Id', 'in', idsToDisassociate]);
          break;
        case 'SET NULL':
          await repository.update({ [inverseField]: null }, ['Id', 'in', idsToDisassociate]);
          break;
        case 'RESTRICT':
        case 'NO ACTION':
          throw new Error(`Cannot remove ${idsToDisassociate.length} association(s) because onDelete is set to ${onDelete}`);
      }
    }

    return { existingIds, removedIds: idsToDisassociate };
  }

  /**
   * Batch diff update for associations.
   */
  private async batchRemoveAssociationsWithDiff(
    repository: Repository,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    inverseField: string,
    parentIds: string[],
    replacementMap: Map<string, unknown[]>
  ): Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>> {
    if (parentIds.length === 0) return new Map();

    const condition: BaseQueryCondition = parentIds.length === 1 ? [inverseField, '=', parentIds[0]] : [inverseField, 'in', parentIds];
    const existingRecords = await repository.search(condition);

    const resultMap = new Map<string, { existingIds: Set<string>; removedIds: string[] }>();
    for (const parentId of parentIds) {
      resultMap.set(parentId, { existingIds: new Set(), removedIds: [] });
    }

    for (const record of existingRecords) {
      const parentId = this.toStringId(record[inverseField]);
      const recordId = this.toStringId(record.Id);
      if (parentId && recordId && resultMap.has(parentId)) resultMap.get(parentId)!.existingIds.add(recordId);
    }

    const onDelete = this.getOnDeletePolicy(targetModel, inverseField);

    const allIdsToRemove: string[] = [];
    for (const [parentId, result] of resultMap.entries()) {
      const newItems = replacementMap.get(parentId) || [];
      const newItemIds = new Set(newItems.map(item => this.extractId(item)).filter(Boolean));

      const idsToRemove: string[] = [];
      result.existingIds.forEach(id => {
        if (!newItemIds.has(id)) {
          idsToRemove.push(id);
          allIdsToRemove.push(id);
        }
      });
      result.removedIds = idsToRemove;
    }

    if (allIdsToRemove.length > 0) {
      switch (onDelete) {
        case 'CASCADE':
          await repository.delete(['Id', 'in', allIdsToRemove]);
          break;
        case 'SET NULL':
          await repository.update({ [inverseField]: null }, ['Id', 'in', allIdsToRemove]);
          break;
        case 'RESTRICT':
        case 'NO ACTION':
          throw new Error(`Cannot remove ${allIdsToRemove.length} association(s) because onDelete is set to ${onDelete}`);
      }
    }

    return resultMap;
  }

  /**
   * Keep compatibility by delegating existing-relation removal to the diff-update implementation.
   */
  private async removeExistingRelations(repository: Repository, inverseField: string, parentId: string): Promise<void> {
    const targetModel = (repository as Repository & { getModelClass?: () => ModelCtor<BaseModel> & typeof BaseModel }).getModelClass?.();
    if (!targetModel) return;
    await this.removeExistingRelationsWithDiff(repository, targetModel, inverseField, parentId, []);
  }

  /**
   * Apply the delete strategy to all child rows for a specific parent entity.
   */
  private async applyDeleteStrategy(
    repository: Repository,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    inverseField: string,
    parentId: string
  ): Promise<void> {
    const onDelete = this.getOnDeletePolicy(targetModel, inverseField);
    const affectedRecords = await repository.search([inverseField, '=', parentId]);
    const affectedIds = affectedRecords.map(record => String(record.Id ?? '')).filter(Boolean);
    if (affectedIds.length === 0) return;

    switch (onDelete) {
      case 'CASCADE':
        await repository.delete(['Id', 'in', affectedIds]);
        break;
      case 'SET NULL':
        await repository.update({ [inverseField]: null }, ['Id', 'in', affectedIds]);
        break;
      case 'RESTRICT':
      case 'NO ACTION':
        throw new Error(`Cannot remove associations: ${affectedIds.length} related ${targetModel.name} record(s) exist and onDelete is set to ${onDelete}`);
    }
  }

  /**
   * Apply the delete strategy to a single child item.
   */
  private async applySingleItemDeleteStrategy(
    repository: Repository,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    inverseField: string,
    parentId: string,
    itemId: string
  ): Promise<{ success: boolean; error?: Error }> {
    try {
      const existing = await repository.search({
        And: [
          ['Id', '=', itemId],
          [inverseField, '=', parentId],
        ],
      });

      if (existing.length === 0) {
        return { success: false, error: new Error(`Item does not exist or does not belong to the parent entity: Id=${itemId}`) };
      }

      const onDelete = this.getOnDeletePolicy(targetModel, inverseField);
      switch (onDelete) {
        case 'CASCADE':
          await repository.delete(['Id', '=', itemId]);
          break;
        case 'SET NULL':
          await repository.update({ [inverseField]: null }, ['Id', '=', itemId]);
          break;
        case 'RESTRICT':
        case 'NO ACTION':
          throw new Error(`Cannot remove association: ${targetModel.name} (ID: ${itemId}) has onDelete set to ${onDelete}`);
      }

      return { success: true };
    } catch (error) {
      return { success: false, error: error instanceof Error ? error : new Error(String(error)) };
    }
  }

  /**
   * Group items by delete strategy.
   */
  private async groupItemsByDeleteStrategy(
    repository: Repository,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    inverseField: string,
    items: { id: string; parentId: string }[]
  ): Promise<{
    cascadeIds: string[];
    setNullIds: string[];
    restrictIds: string[];
    notFoundIds: string[];
  }> {
    const onDelete = this.getOnDeletePolicy(targetModel, inverseField);

    const cascadeIds: string[] = [];
    const setNullIds: string[] = [];
    const restrictIds: string[] = [];
    const notFoundIds: string[] = [];

    for (const item of items) {
      const existing = await repository.search({
        And: [
          ['Id', '=', item.id],
          [inverseField, '=', item.parentId],
        ],
      });

      if (existing.length === 0) {
        notFoundIds.push(item.id);
        continue;
      }

      switch (onDelete) {
        case 'CASCADE':
          cascadeIds.push(item.id);
          break;
        case 'SET NULL':
          setNullIds.push(item.id);
          break;
        case 'RESTRICT':
        case 'NO ACTION':
          restrictIds.push(item.id);
          break;
      }
    }

    return { cascadeIds, setNullIds, restrictIds, notFoundIds };
  }

  /**
   * Execute the grouped delete strategies.
   */
  private async executeDeleteStrategies(
    repository: Repository,
    inverseField: string,
    groups: {
      cascadeIds: string[];
      setNullIds: string[];
      restrictIds: string[];
      notFoundIds: string[];
    }
  ): Promise<{ successIds: string[]; errors: Error[] }> {
    const successIds: string[] = [];
    const errors: Error[] = [];

    if (groups.cascadeIds.length > 0) {
      try {
        await repository.delete(['Id', 'in', groups.cascadeIds]);
        successIds.push(...groups.cascadeIds);
      } catch (error) {
        errors.push(error instanceof Error ? error : new Error(`Cascade delete failed: ${String(error)}`));
      }
    }

    if (groups.setNullIds.length > 0) {
      try {
        await repository.update({ [inverseField]: null }, ['Id', 'in', groups.setNullIds]);
        successIds.push(...groups.setNullIds);
      } catch (error) {
        errors.push(error instanceof Error ? error : new Error(`Setting foreign key to NULL failed: ${String(error)}`));
      }
    }

    if (groups.restrictIds.length > 0) {
      errors.push(new Error(`Cannot remove ${groups.restrictIds.length} association(s) because onDelete is set to RESTRICT`));
    }

    if (groups.notFoundIds.length > 0) {
      errors.push(new Error(`${groups.notFoundIds.length} item(s) do not exist or do not belong to the specified parent entity`));
    }

    return { successIds, errors };
  }

  /**
   * Apply the delete strategy in batch across a parent-entity set.
   */
  private async applyBatchDeleteStrategy(
    repository: Repository,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    inverseField: string,
    parentIds: string[]
  ): Promise<void> {
    if (parentIds.length === 0) return;

    const onDelete = this.getOnDeletePolicy(targetModel, inverseField);
    const condition: BaseQueryCondition = parentIds.length === 1 ? [inverseField, '=', parentIds[0]] : [inverseField, 'in', parentIds];
    const affectedRecords = await repository.search(condition);
    if (affectedRecords.length === 0) return;

    const affectedIds = affectedRecords.map(record => String(record.Id ?? '')).filter(Boolean);

    switch (onDelete) {
      case 'CASCADE':
        await repository.delete(['Id', 'in', affectedIds]);
        break;
      case 'SET NULL':
        await repository.update({ [inverseField]: null }, condition);
        break;
      case 'RESTRICT':
      case 'NO ACTION':
        throw new Error(`Cannot remove associations: ${affectedRecords.length} related ${targetModel.name} record(s) exist and onDelete is set to ${onDelete}`);
    }
  }

  /**
   * Strip compute fields from child-model input so clients cannot override them.
   */
  private stripChildComputeFields(childCtor: ModelCtor<BaseModel> & typeof BaseModel, row: unknown): ObjectRecord {
    const obj = this.ensureObject(row);
    try {
      const childMeta = MetadataStorage.instance.getModelMetadata(childCtor);
      const computeFields = childMeta.computeGraph?.computeFields || new Set<string>();
      if (computeFields.size) {
        for (const k of computeFields) {
          if (k in obj) delete obj[k];
        }
      }
    } catch {
      // Ignore metadata-access failures.
    }
    return obj;
  }

}
