// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Repository } from './repository';
import { MetadataStorage } from '../metadata/storage';
import type { default as BaseModel } from '../model/model';
import type { ModelCtor } from '../metadata/field';

/**
 * Repository factory responsible for creating, caching, and retrieving repository instances associated with models.
 */
export class RepositoryFactory {
  // Map model classes to repository instances through a WeakMap.
  private static readonly repositoryMap = new WeakMap<Function, Repository>();

  /**
   * Get the repository instance associated with the provided model class.
   * Create and cache a new instance when one does not already exist.
   *
   * @param modelClass Model class.
   * @returns Repository instance.
   */
  static getRepository<T extends BaseModel>(modelClass: ModelCtor<T> & typeof BaseModel): Repository {
    if (!this.repositoryMap.has(modelClass)) {
      const meta = MetadataStorage.instance.getModelMetadata(modelClass);
      const repository = new Repository(meta);
      this.repositoryMap.set(modelClass, repository);
    }
    return this.repositoryMap.get(modelClass)!;
  }

  /**
   * Set a custom repository instance for a model class.
   * This is mainly used by tests or advanced integration scenarios.
   *
   * @param modelClass Model class.
   * @param repository Custom repository instance.
   */
  static setRepository<T extends BaseModel>(modelClass: ModelCtor<T> & typeof BaseModel, repository: Repository): void {
    this.repositoryMap.set(modelClass, repository);
  }
}
