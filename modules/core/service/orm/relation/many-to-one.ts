// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { RepositoryFactory } from '../repository/repository_factory';
import { RelationProcessor } from './processor';
import { ManyToOneOperation, PrepareResult, RelationProcessingResult, BatchProcessingResult } from './types';
import type { BaseQueryCondition } from '../repository/types';
import type { RuntimeModelCtor } from '../model/types';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

type ManyToOneBatchGroup = {
  targetModel: RuntimeModelCtor;
  fieldName: string;
  items: Map<string, unknown>;
};

function resolveManyToOneTargetModel(value: unknown): RuntimeModelCtor | undefined {
  const relation = asObjectRecord(value);
  const targetModelFn = relation?.targetModel;
  if (typeof targetModelFn !== 'function') return undefined;
  const targetModel = targetModelFn();
  return typeof targetModel === 'function' ? (targetModel as RuntimeModelCtor) : undefined;
}

/**
 * ManyToOne relation processor.
 * Handles create and update flows for many-to-one relations.
 */
export class ManyToOneProcessor<T extends BaseModel = BaseModel> extends RelationProcessor<T> {
  /**
   * Preprocess ManyToOne relation data for create operations.
   * Ensures referenced target entities already exist.
   */
  public async prepareForCreate(value: ObjectRecord): Promise<PrepareResult<T>> {
    const processedValue = { ...value };

    // Extract ToMany relations. M2O values stay in processedValue directly.
    this.extractRelations(value);

    // Normalize ManyToOne foreign-key assignments.
    for (const [fieldName, field] of this.metadata.fields || []) {
      if (field.type !== 'ManyToOne') continue;

      const fieldValue = value[fieldName];

      // null detaches the relation.
      if (fieldValue === null) {
        processedValue[fieldName] = null;
        continue;
      }

      // Normalize provided values to the target Id.
      if (fieldValue !== undefined) {
        const targetModel = resolveManyToOneTargetModel(field.relation);
        if (!targetModel) {
          // compute/select-driven target resolution is not supported here yet.
          throw new Error(`ManyToOne field is missing relation.targetModel: ${this.modelClass.name}.${fieldName}`);
        }
        const targetId = await this.getOrCreateId(fieldValue, targetModel);
        processedValue[fieldName] = targetId;
      }
    }

    return {
      processedValue,
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
      },
    };
  }

  /**
   * Preprocess ManyToOne relation data for update operations.
   */
  public async prepareForUpdate(value: ObjectRecord, changedFields?: string[]): Promise<PrepareResult<T>> {
    const processedValue = { ...value };
    const fieldsToProcess = changedFields || Object.keys(value);

    for (const [fieldName, field] of this.metadata.fields || []) {
      if (!fieldsToProcess.includes(fieldName)) continue;
      if (field.type !== 'ManyToOne') continue;
      if (!(fieldName in value)) continue;

      const fieldValue = value[fieldName];

      // null detaches the relation.
      if (fieldValue === null) {
        processedValue[fieldName] = null;
        continue;
      }

      // Normalize provided values to the target Id.
      if (fieldValue !== undefined) {
        const targetModel = resolveManyToOneTargetModel(field.relation);
        if (!targetModel) {
          throw new Error(`ManyToOne field is missing relation.targetModel: ${this.modelClass.name}.${fieldName}`);
        }
        const targetId = await this.getOrCreateId(fieldValue, targetModel);
        processedValue[fieldName] = targetId;
      }
    }

    return {
      processedValue,
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
      },
    };
  }

  /**
   * Process a ManyToOne relation update.
   */
  public async processRelationUpdate(parentId: string, operation: ManyToOneOperation): Promise<RelationProcessingResult> {
    if (operation.type !== 'ManyToOne') {
      throw new Error(`Expected a ManyToOne operation, but received ${operation.type}`);
    }

    try {
      const { targetModel, fieldName, value: fieldValue } = operation;

      const repository = RepositoryFactory.getRepository(this.modelClass);

      // null detaches the relation.
      if (fieldValue === null) {
        const result = await repository.update({ [fieldName]: null }, ['Id', '=', parentId]);
        return {
          affectedCount: result.length,
          entityIds: [parentId],
          errors: [],
          targetModel,
          relationType: 'ManyToOne',
        };
      }

      // Point the relation at the requested target.
      const targetId = await this.getOrCreateId(fieldValue, targetModel);
      const result = await repository.update({ [fieldName]: targetId }, ['Id', '=', parentId]);

      return {
        affectedCount: result.length,
        entityIds: [parentId],
        errors: [],
        targetModel,
        relationType: 'ManyToOne',
      };
    } catch (error) {
      return {
        affectedCount: 0,
        entityIds: [],
        errors: [error instanceof Error ? error : new Error(String(error))],
        targetModel: operation.targetModel,
        relationType: 'ManyToOne',
      };
    }
  }

  /**
   * Batch-process ManyToOne relation updates.
   */
  public async batchProcessRelationUpdate(parentIds: string[], operations: ManyToOneOperation[]): Promise<BatchProcessingResult> {
    if (parentIds.length !== operations.length) {
      throw new Error('Parent entity Id array length must match relation operation array length');
    }

    for (const op of operations) {
      if (op.type !== 'ManyToOne') {
        throw new Error(`Expected a ManyToOne operation, but received ${op.type}`);
      }
    }

    // Group by field name and target model.
    const groupedOps = new Map<string, ManyToOneBatchGroup>();

    for (let i = 0; i < parentIds.length; i++) {
      const parentId = parentIds[i];
      const { fieldName, targetModel, value } = operations[i];

      const groupKey = `${fieldName}_${targetModel.name}`;
      if (!groupedOps.has(groupKey)) {
        groupedOps.set(groupKey, { targetModel, fieldName, items: new Map<string, unknown>() });
      }
      groupedOps.get(groupKey)!.items.set(parentId, value);
    }

    const allSuccessIds: string[] = [];
    const allErrors: Error[] = [];

    try {
      const repository = RepositoryFactory.getRepository(this.modelClass);

      for (const [, group] of groupedOps.entries()) {
        const { targetModel, fieldName, items } = group;

        // Group parent entities by target value: one bucket for null, one per target Id.
        const updatesByValue = new Map<string | null, string[]>();

        for (const [parentId, fieldValue] of items.entries()) {
          try {
            if (fieldValue === null) {
              if (!updatesByValue.has(null)) updatesByValue.set(null, []);
              updatesByValue.get(null)!.push(parentId);
              continue;
            }

            const targetId = await this.getOrCreateId(fieldValue, targetModel);
            if (!updatesByValue.has(targetId)) updatesByValue.set(targetId, []);
            updatesByValue.get(targetId)!.push(parentId);
          } catch (e) {
            allErrors.push(e instanceof Error ? e : new Error(String(e)));
          }
        }

        // Execute batched updates.
        for (const [value, ids] of updatesByValue.entries()) {
          if (!ids.length) continue;
          try {
            const condition: BaseQueryCondition = ids.length === 1 ? ['Id', '=', ids[0]] : ['Id', 'in', ids];
            await repository.update({ [fieldName]: value }, condition as never);
            allSuccessIds.push(...ids);
          } catch (e) {
            allErrors.push(e instanceof Error ? e : new Error(String(e)));
          }
        }
      }

      return {
        success: allSuccessIds.map(id => ({
          entityId: id,
          targetModel: this.modelClass.name,
        })),
        errors: allErrors.map(error => ({
          error,
          targetModel: this.modelClass.name,
        })),
        summary: {
          totalOperations: parentIds.length,
          successfulOperations: allSuccessIds.length,
          failedOperations: allErrors.length,
          relationType: 'ManyToOne',
        },
      };
    } catch (error) {
      return {
        success: [],
        errors: [
          {
            error: error instanceof Error ? error : new Error(String(error)),
            targetModel: this.modelClass.name,
          },
        ],
        summary: {
          totalOperations: parentIds.length,
          successfulOperations: 0,
          failedOperations: parentIds.length,
          relationType: 'ManyToOne',
        },
      };
    }
  }
}
