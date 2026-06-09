// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata';
import BaseModel from '../model/model';
import type { InstantiableModelCtor } from '../model/types';
import type { OrderBy } from '../repository/types';
import { installConventionalServiceRuntimeWrappers, registerLoadedModelForGeneratedServiceMetadata } from './service';
import { getRuntimeGlobalPoolValue, setRuntimeGlobalPoolValue } from '@/core/utils/env';
import { asObjectRecord, asRuntimeCarrier } from '@/core/utils/object';
import type { ObjectRecord } from '../../../utils/types';

type ModelDecoratorOrderBy = OrderBy<ObjectRecord>;
type RegisteredModelCtor<T extends BaseModel = BaseModel> = InstantiableModelCtor<T>;
type GlobalPoolLike = {
  set(name: string, model: RegisteredModelCtor<BaseModel>): void;
  get(name: string): RegisteredModelCtor<BaseModel> | undefined;
};

type CompanyScopedDefaultCarrier = {
  __choysum_companyScopedDefault?: unknown;
};

function asGlobalPoolLike(value: unknown): GlobalPoolLike | undefined {
  const record = asRuntimeCarrier(value) ?? asObjectRecord(value);
  if (!record) return undefined;
  if (typeof record.set !== 'function' || typeof record.get !== 'function') return undefined;
  return value as GlobalPoolLike;
}

class ApplicationModelPool {
  private static models: Map<string, RegisteredModelCtor<BaseModel>> = new Map();

  static set(name: string, model: RegisteredModelCtor<BaseModel>): void {
    this.models.set(name, model);
  }

  static get(name: string): RegisteredModelCtor<BaseModel> | undefined {
    return this.models.get(name);
  }
}

/**
 * Configures runtime metadata and behavior for a model class.
 */
export interface ModelOptions {
  tableName?: string;
  orderBy?: ModelDecoratorOrderBy | ModelDecoratorOrderBy[];
  softDelete?: boolean;
  // Only parentField is configurable; ParentPath remains fixed.
  parentField?: string;
  application?: string;
  autoMigrate?: boolean;
  readonly?: boolean;
  /**
   * Enables default company filtering (P2-1).
   * - Prefer explicit configuration to avoid relying on implicit inheritance defaults.
   */
  companyScoped?: boolean;
}

/** @deprecated Prefer ModelOptions. Kept for compatibility. */
export type modelOptions = ModelOptions;

function toSnakeCase(str: string): string {
  return (
    str
      // Insert underscores before uppercase letters, except at the start of the string.
      .replace(/([A-Z])/g, (match, p1, offset) => {
        return (offset > 0 ? '_' : '') + p1.toLowerCase();
      })
      // Lowercase the remaining characters.
      .toLowerCase()
  );
}

/**
 * Registers a model class in metadata storage and the runtime model pool.
 *
 * @param name Model name without the application prefix.
 * @param options Optional runtime metadata for the model.
 * @returns A class decorator that registers the model and installs runtime wrappers.
 */
export function Model(name: string, options?: ModelOptions) {
  return function <T extends RegisteredModelCtor<BaseModel>>(target: T & typeof BaseModel): T {
    const globalPool = asGlobalPoolLike(getRuntimeGlobalPoolValue());
    const modelPool = globalPool || ApplicationModelPool;
    if (!globalPool) {
      setRuntimeGlobalPoolValue(modelPool);
    }

    const appName = options?.application || 'application';
    const fullModelName = appName + '.' + name;

    const inheritCompanyScoped = Boolean((target as unknown as CompanyScopedDefaultCarrier).__choysum_companyScopedDefault);
    const companyScoped = typeof options?.companyScoped === 'boolean' ? options.companyScoped : inheritCompanyScoped;

    MetadataStorage.instance.setModelMetadata(target, {
      name: target.name,
      modelName: name,
      fullModelName: fullModelName,
      application: appName,
      className: target.name,
      tableName: () => options?.tableName || toSnakeCase(appName) + '_' + toSnakeCase(name),
      type: target,
      orderBy: options?.orderBy,
      softDelete: options?.softDelete ?? true,
      companyScoped,
      autoMigrate: options?.autoMigrate,
      readonly: options?.readonly,
      parentField: options?.parentField,
    });

    registerLoadedModelForGeneratedServiceMetadata(fullModelName, target);
    installConventionalServiceRuntimeWrappers(target);

    modelPool.set(fullModelName, target);
    return target;
  };
}

/**
 * Preserves model option typing when declaring strongly typed option objects.
 */
export function defineModelOptions<T extends BaseModel, O extends { tableName?: string; orderBy?: OrderBy<T> | OrderBy<T>[] }>(o: O): O {
  return o;
}
