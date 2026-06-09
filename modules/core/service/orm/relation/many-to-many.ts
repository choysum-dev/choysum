// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata';
import type { ModelCtor } from '../metadata/field';
import { Repository } from '../repository/repository';
import { RepositoryFactory } from '../repository/repository_factory';
import { RelationProcessor } from './processor';
import { ManyToManyOperation, PrepareResult, RelationProcessingResult, ExtractedRelations, BatchProcessingResult } from './types';
import { BaseQueryCondition, RelationOperations } from '../repository/types';
import { createRelationModel, updateRelationModelById } from './relation_model_service_facade';
import type { ObjectRecord } from '../../../utils/types';

/**
 * ManyToMany relation processor.
 * Handles create and update flows for many-to-many relations.
 */
export class ManyToManyProcessor<T extends BaseModel = BaseModel> extends RelationProcessor<T> {
  /**
   * Preprocess ManyToMany relation data for create operations.
   * ManyToMany relations are applied after the parent entity is created, so this step only extracts relation info.
   */
  public async prepareForCreate(value: ObjectRecord): Promise<PrepareResult<T>> {
    const processedValue = { ...value } as ObjectRecord;
    const relations = this.extractRelations(value);

    // Remove M2M fields that are not persisted directly on the write path.
    this.metadata.fields?.forEach((field, fieldName) => {
      if (field.type === 'ManyToMany') {
        delete processedValue[fieldName];
      }
    });

    return {
      processedValue,
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: relations.manyToManyRelations,
        touchedCollections: relations.touchedCollections,
      },
    };
  }

  /**
   * Preprocess ManyToMany relation data for update operations.
   */
  public async prepareForUpdate(value: ObjectRecord, changedFields?: string[]): Promise<PrepareResult<T>> {
    const processedValue = { ...value } as ObjectRecord;
    const fieldsToProcess = changedFields || Object.keys(value);

    const relations: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(),
    };

    this.metadata.fields?.forEach((field, fieldName) => {
      if (!fieldsToProcess.includes(fieldName)) return;
      if (field.type !== 'ManyToMany') return;
      if (!(fieldName in value)) return;

      const fieldValue = value[fieldName];
      delete processedValue[fieldName];

      const rel = this.resolveManyToManyRelationConfig(field.relation);
      if (fieldValue !== null && fieldValue !== undefined && rel) {
        relations.manyToManyRelations.push({
          fieldName,
          type: 'ManyToMany',
          joinModel: rel.joinModel(),
          targetModel: rel.targetModel(),
          joinField: rel.joinField,
          inverseJoinField: rel.inverseJoinField,
          operations: fieldValue as ManyToManyOperation['operations'],
        });
        relations.touchedCollections!.add(fieldName);
      }
    });

    return { processedValue, relations };
  }

  /**
   * Process a ManyToMany relation update.
   */
  public async processRelationUpdate(parentId: string, operation: ManyToManyOperation): Promise<RelationProcessingResult> {
    if (operation.type !== 'ManyToMany') {
      throw new Error(`Expected a ManyToMany operation, but received ${operation.type}`);
    }

    try {
      const { joinModel, targetModel, joinField, inverseJoinField, operations } = operation;

      // Join-table repository.
      const meta = MetadataStorage.instance.getModelMetadata(joinModel);
      const joinRepo = new Repository(meta);

      // Target-entity repository for relation existence checks and similar reads.
      const targetRepo = RepositoryFactory.getRepository(targetModel);

      const affectedIds: string[] = [];
      const errors: Error[] = [];

      // Array mode: replace.
      if (Array.isArray(operations)) {
        const diffResult = await this.removeExistingRelationsWithDiff(joinRepo, joinField, inverseJoinField, parentId, operations);

        // Only create newly introduced relations.
        const existingTargetIdSet = new Set(diffResult.existingTargetIds);
        const newItemsToCreate = operations.filter(item => {
          const id = this.extractId(item);
          return !id || !existingTargetIdSet.has(id);
        });

        if (newItemsToCreate.length > 0) {
          await this.createRelations(joinRepo, targetRepo, targetModel, parentId, joinField, inverseJoinField, newItemsToCreate, affectedIds, errors);
        }

        // Existing retained IDs also count as successful processing.
        affectedIds.push(...diffResult.existingTargetIds);

        return {
          affectedCount: affectedIds.length,
          entityIds: affectedIds,
          errors,
          targetModel,
          relationType: 'ManyToMany',
        };
      }

      // Object mode: replace/delete/create/update.
      const relationOps = operations as RelationOperations<BaseModel>;

      // replace
      if (relationOps.replace) {
        const diffResult = await this.removeExistingRelationsWithDiff(joinRepo, joinField, inverseJoinField, parentId, relationOps.replace);

        const existingTargetIdSet = new Set(diffResult.existingTargetIds);
        const newItemsToCreate = relationOps.replace.filter(item => {
          const id = this.extractId(item);
          return !id || !existingTargetIdSet.has(id);
        });

        if (newItemsToCreate.length > 0) {
          await this.createRelations(joinRepo, targetRepo, targetModel, parentId, joinField, inverseJoinField, newItemsToCreate, affectedIds, errors);
        }

        affectedIds.push(...diffResult.existingTargetIds);
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

            await joinRepo.delete({
              And: [
                [joinField, '=', parentId],
                [inverseJoinField, '=', id],
              ],
            });

            affectedIds.push(id);
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process delete item: ${String(e)}`));
          }
        }
      }

      // create
      if (relationOps.create) {
        await this.createRelations(joinRepo, targetRepo, targetModel, parentId, joinField, inverseJoinField, relationOps.create, affectedIds, errors);
      }

      // Update only the target entity, using UpdateById so compute and validation still run.
      if (relationOps.update) {
        for (const item of relationOps.update) {
          try {
            const id = this.extractId(item);
            if (!id) {
              errors.push(new Error('Could not extract Id from update item'));
              continue;
            }

            // Verify that the relation already exists.
            const existingRelation = await joinRepo.search({
              And: [
                [joinField, '=', parentId],
                [inverseJoinField, '=', id],
              ],
            });
            if (existingRelation.length === 0) {
              errors.push(new Error(`Attempted to update a non-related entity: Id=${id}`));
              continue;
            }

            const itemObj = this.ensureObject(item);
            const { Id, ...dataToUpdate } = itemObj;
            if (await updateRelationModelById(targetModel, id, dataToUpdate)) affectedIds.push(id);
          } catch (e) {
            errors.push(e instanceof Error ? e : new Error(`Failed to process update item: ${String(e)}`));
          }
        }
      }

      return {
        affectedCount: affectedIds.length,
        entityIds: affectedIds,
        errors,
        targetModel,
        relationType: 'ManyToMany',
      };
    } catch (error) {
      return {
        affectedCount: 0,
        entityIds: [],
        errors: [error instanceof Error ? error : new Error(String(error))],
        targetModel: operation.targetModel,
        relationType: 'ManyToMany',
      };
    }
  }

  /**
   * Batch-process ManyToMany relation updates.
   */
  public async batchProcessRelationUpdate(parentIds: string[], operations: ManyToManyOperation[]): Promise<BatchProcessingResult> {
    if (parentIds.length !== operations.length) {
      throw new Error('Parent entity Id array length must match relation operation array length');
    }
    for (const op of operations) {
      if (op.type !== 'ManyToMany') throw new Error(`Expected a ManyToMany operation, but received ${op.type}`);
    }

    const groupedOps = this.groupOperationsByTarget(parentIds, operations);
    const allSuccessIds: string[] = [];
    const allErrors: Error[] = [];

    try {
      for (const [, group] of groupedOps.entries()) {
        const { config, parentMap } = group;
        const { joinModel, targetModel, joinField, inverseJoinField } = config;

        const meta = MetadataStorage.instance.getModelMetadata(joinModel);
        const joinRepo = new Repository(meta);
        const targetRepo = RepositoryFactory.getRepository(targetModel);

        const classifiedOps = this.classifyOperations(parentMap);

        // replace
        const replaceOps = classifiedOps.replace;
        if (replaceOps.size > 0) {
          const parentIdsToReplace = Array.from(replaceOps.keys());

          const diffResults = await this.batchRemoveAssociationsWithDiff(joinRepo, joinField, inverseJoinField, parentIdsToReplace, replaceOps);

          for (const [parentId, items] of replaceOps.entries()) {
            const existingIds = diffResults.get(parentId)?.existingIds || new Set<string>();

            const itemsToCreate = items.filter(item => {
              const id = this.extractId(item);
              return !id || !existingIds.has(id);
            });

            if (itemsToCreate.length > 0) {
              const processedResults = await this.processReplaceItems(joinRepo, targetRepo, targetModel, parentId, joinField, inverseJoinField, itemsToCreate);
              allSuccessIds.push(...processedResults.successIds);
              allErrors.push(...processedResults.errors);
            }

            existingIds.forEach(id => {
              if (!allSuccessIds.includes(id)) allSuccessIds.push(id);
            });
          }
        }

        // delete
        const deleteOps = classifiedOps.delete;
        if (deleteOps.size > 0) {
          for (const [parentId, items] of deleteOps.entries()) {
            const ids = items.map(it => this.extractId(it)).filter(Boolean) as string[];
            const idExtractErrors = items.filter(it => !this.extractId(it)).map(() => new Error('Could not extract Id from delete item'));

            if (ids.length > 0) {
              try {
                const orConditions: BaseQueryCondition =
                  ids.length === 1
                    ? {
                        And: [[joinField, '=', parentId] as BaseQueryCondition, [inverseJoinField, '=', ids[0]] as BaseQueryCondition],
                      }
                    : {
                        And: [[joinField, '=', parentId] as BaseQueryCondition, [inverseJoinField, 'in', ids] as BaseQueryCondition],
                      };
                await joinRepo.delete(orConditions);
                allSuccessIds.push(...ids);
              } catch (e) {
                allErrors.push(e instanceof Error ? e : new Error(String(e)));
              }
            }

            allErrors.push(...idExtractErrors);
          }
        }

        // create
        const createOps = classifiedOps.create;
        if (createOps.size > 0) {
          for (const [parentId, items] of createOps.entries()) {
            // Filter out relations that already exist.
            const idsWithId = items.map(it => this.extractId(it)).filter(Boolean) as string[];
            let existingSet = new Set<string>();
            if (idsWithId.length > 0) {
              const condition: BaseQueryCondition =
                idsWithId.length === 1
                  ? {
                      And: [[joinField, '=', parentId] as BaseQueryCondition, [inverseJoinField, '=', idsWithId[0]] as BaseQueryCondition],
                    }
                  : {
                      And: [[joinField, '=', parentId] as BaseQueryCondition, [inverseJoinField, 'in', idsWithId] as BaseQueryCondition],
                    };
              const existing = await joinRepo.search(condition);
              existingSet = new Set(existing.map(r => this.toStringId(r[inverseJoinField])).filter((id): id is string => id !== null));
            }
            const toCreate = items.filter(it => {
              const id = this.extractId(it);
              return !id || !existingSet.has(id);
            });

            if (toCreate.length > 0) {
              const processedResults = await this.processReplaceItems(joinRepo, targetRepo, targetModel, parentId, joinField, inverseJoinField, toCreate);
              allSuccessIds.push(...processedResults.successIds);
              allErrors.push(...processedResults.errors);
            }
          }
        }

        // update
        const updateOps = classifiedOps.update;
        if (updateOps.size > 0) {
          for (const [parentId, items] of updateOps.entries()) {
            for (const item of items) {
              try {
                const id = this.extractId(item);
                if (!id) {
                  allErrors.push(new Error('Could not extract Id from update item'));
                  continue;
                }

                const existingRelation = await joinRepo.search({
                  And: [
                    [joinField, '=', parentId],
                    [inverseJoinField, '=', id],
                  ],
                });
                if (existingRelation.length === 0) {
                  allErrors.push(new Error(`Attempted to update a non-related entity: Id=${id}`));
                  continue;
                }

                const itemObj = this.ensureObject(item);
                const { Id, ...dataToUpdate } = itemObj;
                if (await updateRelationModelById(targetModel, id, dataToUpdate)) allSuccessIds.push(id);
              } catch (e) {
                allErrors.push(e instanceof Error ? e : new Error(String(e)));
              }
            }
          }
        }
      }

      return this.createBatchResult(
        allSuccessIds,
        allErrors,
        'ManyToMany',
        operations[0]?.targetModel?.name || 'Unknown target',
        operations[0]?.joinModel?.name
      );
    } catch (error) {
      return {
        success: [],
        errors: [
          {
            error: error instanceof Error ? error : new Error(String(error)),
            targetModel: operations[0]?.targetModel?.name || 'Unknown target',
            joinModel: operations[0]?.joinModel?.name,
          },
        ],
        summary: {
          totalOperations: parentIds.length,
          successfulOperations: 0,
          failedOperations: parentIds.length,
          relationType: 'ManyToMany',
        },
      };
    }
  }

  /**
   * Helper to create many-to-many relations by creating target entities and inserting join-table rows.
   * Target entity creation uses the target model Create path so defaults, compute, and onchange still run.
   */
  private async createRelations(
    joinRepo: Repository,
    targetRepo: unknown,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    parentId: string,
    joinField: string,
    inverseJoinField: string,
    items: unknown[],
    affectedIds: string[],
    errors: Error[]
  ): Promise<void> {
    const joinRecords: ObjectRecord[] = [];

    for (const item of items) {
      try {
        let targetId: string;

        if (typeof item === 'string') {
          targetId = item;
        } else if (this.extractId(item)) {
          targetId = this.extractId(item)!;
        } else {
          const itemObj = this.ensureObject(item);
          targetId = await createRelationModel(targetModel, itemObj);
          if (!targetId) throw new Error('Failed to create target entity');
        }

        joinRecords.push({
          [joinField]: parentId,
          [inverseJoinField]: targetId,
        });

        affectedIds.push(targetId);
      } catch (e) {
        errors.push(e instanceof Error ? e : new Error(`Failed to process relation item: ${String(e)}`));
      }
    }

    if (joinRecords.length > 0) {
      try {
        await joinRepo.create(joinRecords);
      } catch (e) {
        errors.push(e instanceof Error ? e : new Error(`Failed to create join records in batch: ${String(e)}`));
      }
    }
  }

  /**
   * Helper for diff updates on a single parent entity.
   */
  private async removeExistingRelationsWithDiff(
    joinRepo: Repository,
    joinField: string,
    inverseJoinField: string,
    parentId: string,
    newItems: unknown[]
  ): Promise<{ existingRecords: ObjectRecord[]; existingTargetIds: string[]; removedRecords: ObjectRecord[] }> {
    const existingRecords = await joinRepo.search([joinField, '=', parentId]);

    if (!existingRecords.length) {
      return { existingRecords, existingTargetIds: [], removedRecords: [] };
    }

    if (!newItems || newItems.length === 0) {
      await joinRepo.delete([joinField, '=', parentId]);
      const existingTargetIds = existingRecords.map(r => this.toStringId(r[inverseJoinField])).filter((id): id is string => id !== null);
      return {
        existingRecords,
        existingTargetIds,
        removedRecords: [...existingRecords],
      };
    }

    const existingTargetIds = existingRecords.map(r => this.toStringId(r[inverseJoinField])).filter((id): id is string => id !== null);
    const newTargetIds = new Set(newItems.map(item => this.extractId(item)).filter(Boolean));

    const recordsToRemove = existingRecords.filter(record => {
      const targetId = this.toStringId(record[inverseJoinField]);
      return !targetId || !newTargetIds.has(targetId);
    });

    if (recordsToRemove.length > 0) {
      if (recordsToRemove.length < existingRecords.length / 2) {
        const recordIds = recordsToRemove.map(r => this.toStringId(r.Id)).filter((id): id is string => id !== null);
        await joinRepo.delete(['Id', 'in', recordIds]);
      } else {
        const targetIdsToRemove = recordsToRemove.map(r => this.toStringId(r[inverseJoinField])).filter((id): id is string => id !== null);
        await joinRepo.delete({
          And: [
            [joinField, '=', parentId],
            [inverseJoinField, 'in', targetIdsToRemove],
          ],
        });
      }
    }

    return {
      existingRecords,
      existingTargetIds,
      removedRecords: recordsToRemove,
    };
  }

  /**
   * Helper for diff updates across multiple parent entities.
   */
  private async batchRemoveAssociationsWithDiff(
    joinRepo: Repository,
    joinField: string,
    inverseJoinField: string,
    parentIds: string[],
    replacementMap: Map<string, unknown[]>
  ): Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>> {
    if (parentIds.length === 0) return new Map();

    const condition: BaseQueryCondition = parentIds.length === 1 ? [joinField, '=', parentIds[0]] : [joinField, 'in', parentIds];
    const existingRecords = await joinRepo.search(condition);

    const resultMap = new Map<string, { existingIds: Set<string>; removedIds: string[] }>();
    for (const parentId of parentIds) resultMap.set(parentId, { existingIds: new Set(), removedIds: [] });

    for (const record of existingRecords) {
      const parentId = this.toStringId(record[joinField]);
      const targetId = this.toStringId(record[inverseJoinField]);
      if (parentId && targetId && resultMap.has(parentId)) {
        resultMap.get(parentId)!.existingIds.add(targetId);
      }
    }

    const allPairsToRemove: Array<[string, string]> = [];

    for (const [parentId, result] of resultMap.entries()) {
      const newItems = replacementMap.get(parentId) || [];
      const newItemIds = new Set(newItems.map(item => this.extractId(item)).filter(Boolean));

      const targetIdsToRemove: string[] = [];
      result.existingIds.forEach(targetId => {
        if (!newItemIds.has(targetId)) {
          targetIdsToRemove.push(targetId);
          allPairsToRemove.push([parentId, targetId]);
        }
      });

      result.removedIds = targetIdsToRemove;
    }

    if (allPairsToRemove.length > 0) {
      if (allPairsToRemove.length <= 100) {
        const batchSize = 20;
        for (let i = 0; i < allPairsToRemove.length; i += batchSize) {
          const batch = allPairsToRemove.slice(i, i + batchSize);
          const orConditions: BaseQueryCondition[] = batch.map(([pId, tId]) => ({
            And: [[joinField, '=', pId] as BaseQueryCondition, [inverseJoinField, '=', tId] as BaseQueryCondition],
          }));

          if (orConditions.length === 1) {
            await joinRepo.delete(orConditions[0]);
          } else {
            await joinRepo.delete({ Or: orConditions });
          }
        }
      } else {
        const parentIdsToProcess = Array.from(new Set(allPairsToRemove.map(p => p[0])));

        for (const parentId of parentIdsToProcess) {
          const targetIds = allPairsToRemove.filter(p => p[0] === parentId).map(p => p[1]);
          if (targetIds.length > 0) {
            const deleteCondition: BaseQueryCondition = {
              And: [[joinField, '=', parentId] as BaseQueryCondition, [inverseJoinField, 'in', targetIds] as BaseQueryCondition],
            };
            await joinRepo.delete(deleteCondition);
          }
        }
      }
    }

    return resultMap;
  }

  /**
   * Helper to batch-process replace and create items.
   * Target entity creation uses the target model Create path.
   */
  private async processReplaceItems(
    joinRepo: Repository,
    targetRepo: unknown,
    targetModel: ModelCtor<BaseModel> & typeof BaseModel,
    parentId: string,
    joinField: string,
    inverseJoinField: string,
    items: unknown[]
  ): Promise<{ successIds: string[]; errors: Error[] }> {
    const successIds: string[] = [];
    const errors: Error[] = [];
    const joinRecords: ObjectRecord[] = [];

    for (const item of items) {
      try {
        let targetId: string;

        if (typeof item === 'string') {
          targetId = item;
        } else if (this.extractId(item)) {
          targetId = this.extractId(item)!;
        } else {
          const itemObj = this.ensureObject(item);
          targetId = await createRelationModel(targetModel, itemObj);
          if (!targetId) throw new Error('Failed to create target entity');
        }

        joinRecords.push({
          [joinField]: parentId,
          [inverseJoinField]: targetId,
        });

        successIds.push(targetId);
      } catch (e) {
        errors.push(e instanceof Error ? e : new Error(String(e)));
      }
    }

    if (joinRecords.length > 0) {
      try {
        await joinRepo.create(joinRecords);
      } catch (e) {
        errors.push(e instanceof Error ? e : new Error(`Failed to create join records in batch: ${String(e)}`));
      }
    }

    return { successIds, errors };
  }
}
