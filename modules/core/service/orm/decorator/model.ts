// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata';
import { validateModelMonetaryCurrencyFields } from '../metadata/monetary_currency';
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
   * Company row isolation (P2-1): ownership field name.
   * Non-empty string enables Repository company filtering on that column.
   * Omitted values inherit the parent model's companyField; clearing/renaming a
   * parent value is rejected.
   */
  companyField?: string;
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

function resolveParentCompanyField(target: Function): string | undefined {
  let current: unknown = Object.getPrototypeOf(target);
  while (current && current !== Object.prototype && typeof current === 'function') {
    try {
      const parentMeta = MetadataStorage.instance.getModelMetadata(current as InstantiableModelCtor<BaseModel>);
      const field = String(parentMeta?.companyField ?? '').trim();
      if (field) return field;
    } catch {
      // Parent may not be registered yet; keep walking.
    }
    current = Object.getPrototypeOf(current);
  }
  return undefined;
}

/**
 * Resolve companyField with monotonic inheritance (design D4).
 */
export function resolveModelCompanyField(target: Function, optionsCompanyField: string | undefined): string | undefined {
  const parentField = resolveParentCompanyField(target);

  if (optionsCompanyField !== undefined) {
    const field = String(optionsCompanyField).trim();
    if (!field) {
      throw new Error(`@Model companyField cannot be empty on ${target.name || 'model'}`);
    }
    if (parentField && parentField !== field) {
      throw new Error(
        `@Model companyField cannot rename inherited value '${parentField}' to '${field}' on ${target.name || 'model'}`
      );
    }
    return field;
  }

  return parentField;
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

    const companyField = resolveModelCompanyField(target, options?.companyField);

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
      companyField,
      autoMigrate: options?.autoMigrate,
      readonly: options?.readonly,
      parentField: options?.parentField,
    });

    // Monetary currencyField targets must resolve after all @Field decorators ran.
    validateModelMonetaryCurrencyFields(MetadataStorage.instance.getModelMetadata(target));

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
