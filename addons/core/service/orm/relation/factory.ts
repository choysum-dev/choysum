// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata';
import type { RuntimeModelCtor } from '../model/types';
import { RelationProcessor } from './processor';
import { ManyToOneProcessor } from './many-to-one';
import { OneToManyProcessor } from './one-to-many';
import { ManyToManyProcessor } from './many-to-many';
import {
  ExtractedRelations,
  type ManyToManyRelationConfig,
  OneToManyOperation,
  ManyToManyOperation,
  type OneToManyRelationConfig,
  RelationChangesCollection,
  resolveManyToManyRelationConfig,
  resolveOneToManyRelationConfig,
  type BatchProcessingResult,
  type RelationFieldType,
  type RelationProcessingResult,
} from './types';
import { asObjectRecord } from '../../../utils/object';
import type { ObjectRecord } from '../../../utils/types';

function extractRelationItemId(value: unknown): string | undefined {
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'bigint') {
    return String(value);
  }
  const record = asObjectRecord(value);
  if (!record) return undefined;
  const rawId = record.Id ?? record.id;
  if (typeof rawId === 'string' || typeof rawId === 'number' || typeof rawId === 'bigint') {
    return String(rawId);
  }
  return undefined;
}

function toRelationPayload(items: unknown[]): Array<{ Id: string }> {
  const payload: Array<{ Id: string }> = [];
  for (const item of items) {
    const id = extractRelationItemId(item);
    if (id) payload.push({ Id: id });
  }
  return payload;
}

function createBatchFailureResult(
  modelClass: RuntimeModelCtor,
  relationType: 'OneToMany' | 'ManyToMany',
  totalOperations: number,
  error: unknown
): BatchProcessingResult {
  return {
    success: [],
    errors: [
      {
        error: error instanceof Error ? error : new Error(String(error)),
        targetModel: modelClass.name,
      },
    ],
    summary: {
      totalOperations,
      successfulOperations: 0,
      failedOperations: totalOperations,
      relationType,
    },
  };
}

/**
 * Relation processor factory.
 * Creates and manages processor instances for each supported relation type.
 */
export class RelationFactory {
  /**
   * Processor cache to avoid recreating identical processor instances.
   * Cache key format: `${modelClassName}_${relationType}`.
   */
  private static readonly processorCache = new Map<string, RelationProcessor<BaseModel>>();

  /**
   * Create a processor instance for the given model and relation type.
   */
  static createProcessor<T extends BaseModel>(modelClass: RuntimeModelCtor<T>, relationType: RelationFieldType): RelationProcessor<T> {
    const cacheKey = `${modelClass.name}_${relationType}`;
    const cached = this.processorCache.get(cacheKey);
    if (cached) {
      return cached as RelationProcessor<T>;
    }

    let processor: RelationProcessor<T>;
    switch (relationType) {
      case 'ManyToOne':
        processor = new ManyToOneProcessor<T>(modelClass);
        break;
      case 'OneToMany':
        processor = new OneToManyProcessor<T>(modelClass);
        break;
      case 'ManyToMany':
        processor = new ManyToManyProcessor<T>(modelClass);
        break;
      default:
        throw new Error(`Unsupported relation type: ${relationType}`);
    }

    this.processorCache.set(cacheKey, processor as RelationProcessor<BaseModel>);
    return processor;
  }

  /**
   * Get all relation processors needed by the specified model.
   */
  static getProcessorsForModel<T extends BaseModel>(modelClass: RuntimeModelCtor<T>): Map<RelationFieldType, RelationProcessor<T>> {
    const result = new Map<RelationFieldType, RelationProcessor<T>>();
    const modelMetadata = MetadataStorage.instance.getModelMetadata(modelClass);

    if (!modelMetadata.fields) return result;

    modelMetadata.fields.forEach(field => {
      // Recognize relation fields based on FieldMetadata.type only.
      const relationType = field.type;
      if (relationType !== 'ManyToOne' && relationType !== 'OneToMany' && relationType !== 'ManyToMany') {
        return;
      }

      if (!result.has(relationType)) {
        try {
          const processor = this.createProcessor<T>(modelClass, relationType);
          result.set(relationType, processor);
        } catch (error) {
          console.warn(`Failed to create relation processor: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
    });

    return result;
  }

  /**
   * Prepare relation handling for create operations.
   */
  static async prepareForCreate<T extends BaseModel>(
    modelClass: RuntimeModelCtor<T>,
    value: ObjectRecord
  ): Promise<{ processedValue: ObjectRecord; relations: ExtractedRelations }> {
    const processors = this.getProcessorsForModel(modelClass);
    let processedValue = { ...value };
    const allRelations: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(), // NEW
    };

    for (const processor of processors.values()) {
      const result = await processor.prepareForCreate(processedValue);
      processedValue = result.processedValue;
      allRelations.oneToManyRelations.push(...(result.relations.oneToManyRelations || []));
      allRelations.manyToManyRelations.push(...(result.relations.manyToManyRelations || []));
      if (result.relations.touchedCollections) {
        result.relations.touchedCollections.forEach(c => allRelations.touchedCollections!.add(c));
      }
    }

    if (allRelations.touchedCollections && allRelations.touchedCollections.size === 0) {
      // delete allRelations.touchedCollections;
    }

    return { processedValue, relations: allRelations };
  }

  /**
   * Prepare relation handling for update operations.
   */
  static async prepareForUpdate<T extends BaseModel>(
    modelClass: RuntimeModelCtor<T>,
    value: ObjectRecord,
    changedFields?: string[]
  ): Promise<{ processedValue: ObjectRecord; relations: ExtractedRelations }> {
    const processors = this.getProcessorsForModel(modelClass);
    let processedValue = { ...value };
    const allRelations: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(), // NEW
    };

    for (const processor of processors.values()) {
      const result = await processor.prepareForUpdate(processedValue, changedFields);
      processedValue = result.processedValue;
      allRelations.oneToManyRelations.push(...(result.relations.oneToManyRelations || []));
      allRelations.manyToManyRelations.push(...(result.relations.manyToManyRelations || []));
      if (result.relations.touchedCollections) {
        result.relations.touchedCollections.forEach(c => allRelations.touchedCollections!.add(c));
      }
    }

    if (allRelations.touchedCollections && allRelations.touchedCollections.size === 0) {
      // delete allRelations.touchedCollections;
    }

    return { processedValue, relations: allRelations };
  }

  /**
   * Process ToMany relation updates.
   */
  static async processToManyRelations<T extends BaseModel>(
    modelClass: RuntimeModelCtor<T>,
    parentId: string,
    relationData: ExtractedRelations
  ): Promise<RelationProcessingResult[]> {
    const results: RelationProcessingResult[] = [];

    if (relationData.oneToManyRelations?.length) {
      const processor = this.createProcessor(modelClass, 'OneToMany');
      for (const operation of relationData.oneToManyRelations) {
        results.push(await processor.processRelationUpdate(parentId, operation));
      }
    }

    if (relationData.manyToManyRelations?.length) {
      const processor = this.createProcessor(modelClass, 'ManyToMany');
      for (const operation of relationData.manyToManyRelations) {
        results.push(await processor.processRelationUpdate(parentId, operation));
      }
    }

    return results;
  }

  /**
   * Batch-process ToMany relations for multiple entities.
   */
  static async batchProcessToManyRelations<T extends BaseModel>(
    modelClass: RuntimeModelCtor<T>,
    parentIds: string[],
    relationsList: ExtractedRelations[]
  ): Promise<BatchProcessingResult[]> {
    if (parentIds.length !== relationsList.length) {
      throw new Error('Parent entity Id array length must match relation data array length');
    }
    if (!parentIds.length) return [];

    const results: BatchProcessingResult[] = [];

    // Collect O2M operations.
    const oneToManyOperations: OneToManyOperation[] = [];
    const oneToManyParentIds: string[] = [];

    // Collect M2M operations.
    const manyToManyOperations: ManyToManyOperation[] = [];
    const manyToManyParentIds: string[] = [];

    for (let i = 0; i < parentIds.length; i++) {
      const parentId = parentIds[i];
      const relationData = relationsList[i];

      if (relationData.oneToManyRelations?.length) {
        for (const op of relationData.oneToManyRelations) {
          oneToManyOperations.push(op);
          oneToManyParentIds.push(parentId);
        }
      }

      if (relationData.manyToManyRelations?.length) {
        for (const op of relationData.manyToManyRelations) {
          manyToManyOperations.push(op);
          manyToManyParentIds.push(parentId);
        }
      }
    }

    if (oneToManyOperations.length) {
      const processor = this.createProcessor(modelClass, 'OneToMany');
      try {
        const result = await processor.batchProcessRelationUpdate(oneToManyParentIds, oneToManyOperations);
        results.push(result);
      } catch (error) {
        results.push(createBatchFailureResult(modelClass, 'OneToMany', oneToManyOperations.length, error));
      }
    }

    if (manyToManyOperations.length) {
      const processor = this.createProcessor(modelClass, 'ManyToMany');
      try {
        const result = await processor.batchProcessRelationUpdate(manyToManyParentIds, manyToManyOperations);
        results.push(result);
      } catch (error) {
        results.push(createBatchFailureResult(modelClass, 'ManyToMany', manyToManyOperations.length, error));
      }
    }

    return results;
  }

  /**
   * Clear the processor cache.
   */
  static clearCache(): void {
    this.processorCache.clear();
  }

  /**
   * Check whether a specific processor instance is already cached.
   */
  static hasProcessorCached<T extends BaseModel>(modelClass: RuntimeModelCtor<T>, relationType: RelationFieldType): boolean {
    const cacheKey = `${modelClass.name}_${relationType}`;
    return this.processorCache.has(cacheKey);
  }

  /**
   * Convert recorded array mutations into relation operations using the current naming model.
   */
  static prepareRelationChanges<T extends BaseModel>(
    modelClass: RuntimeModelCtor<T>,
    modelInstance: T,
    relationChanges: RelationChangesCollection,
    relations: ExtractedRelations
  ): void {
    const meta = MetadataStorage.instance.getModelMetadata(modelClass);
    const instanceRecord = asObjectRecord(modelInstance);
    if (!instanceRecord) return;

    Object.entries(relationChanges).forEach(([fieldName]) => {
      const fieldMeta = meta.fields.get(fieldName);
      if (!fieldMeta?.relation) return;

      const currentRelationValue = instanceRecord[fieldName];
      if (!Array.isArray(currentRelationValue)) return;

      switch (fieldMeta.type) {
        case 'OneToMany': {
          const rel = resolveOneToManyRelationConfig(fieldMeta.relation);
          if (!rel) return;
          const existing = relations.oneToManyRelations.find(r => r.fieldName === fieldName);
          const payload = toRelationPayload(currentRelationValue);
          if (existing) {
            existing.operations = payload;
          } else {
            relations.oneToManyRelations.push({
              fieldName,
              type: 'OneToMany',
              targetModel: rel.targetModel(),
              inverseField: rel.inverseField,
              operations: payload,
            });
          }
          (relations.touchedCollections ||= new Set()).add(fieldName);
          break;
        }
        case 'ManyToMany': {
          const rel = resolveManyToManyRelationConfig(fieldMeta.relation);
          if (!rel) return;
          const existing = relations.manyToManyRelations.find(r => r.fieldName === fieldName);
          const payload = toRelationPayload(currentRelationValue);
          if (existing) {
            existing.operations = payload;
          } else {
            relations.manyToManyRelations.push({
              fieldName,
              type: 'ManyToMany',
              joinModel: rel.joinModel(),
              targetModel: rel.targetModel(),
              joinField: rel.joinField,
              inverseJoinField: rel.inverseJoinField,
              operations: payload,
            });
          }
          (relations.touchedCollections ||= new Set()).add(fieldName);
          break;
        }

        default:
          // ManyToOne is not handled here.
          break;
      }
    });
  }
}
