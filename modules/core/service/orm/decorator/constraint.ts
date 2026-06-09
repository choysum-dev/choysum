// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import type BaseModel from '../model/model';
import type { ModelCtor } from '../metadata/field';
import type { ConstraintField, ConstraintMeta, ConstraintOptions } from '../metadata/constraint';

const CONSTRAINT_DEFAULT_PRIORITY = 20;
const CONSTRAINT_DEFAULT_ALWAYS_ON_CREATE = true;

type ConstraintDecoratorArg<TModel extends BaseModel> = ConstraintField<TModel> | ConstraintField<TModel>[] | ConstraintOptions<TModel>;

function normalizeConstraintArgs<TModel extends BaseModel>(args: Array<ConstraintDecoratorArg<TModel>>): ConstraintOptions<TModel> {
  let options: ConstraintOptions<TModel> = {};
  let rawFields = args;

  if (args.length > 0) {
    const last = args[args.length - 1];
    if (last && typeof last === 'object' && !Array.isArray(last)) {
      options = last as ConstraintOptions<TModel>;
      rawFields = args.slice(0, -1);
    }
  }

  const fields: ConstraintField<TModel>[] = [];
  for (const item of rawFields) {
    if (!item) continue;
    if (Array.isArray(item)) {
      for (const field of item) {
        if (typeof field === 'string' && field.trim()) fields.push(field.trim() as ConstraintField<TModel>);
      }
      continue;
    }
    if (typeof item === 'string' && item.trim()) {
      fields.push(item.trim() as ConstraintField<TModel>);
    }
  }

  if (Array.isArray(options.fields)) {
    for (const field of options.fields) {
      if (typeof field === 'string' && field.trim()) fields.push(field.trim() as ConstraintField<TModel>);
    }
  }

  return {
    ...options,
    fields: [...new Set(fields)],
  };
}

/**
 * Registers a model constraint handler and its execution metadata.
 *
 * @param args Constraint fields and optional execution options.
 * @returns A method decorator that records the constraint handler on model metadata.
 */
export function Constraint<TModel extends BaseModel = BaseModel>(...args: Array<ConstraintDecoratorArg<TModel>>): MethodDecorator;
export function Constraint<T extends BaseModel = BaseModel>(...args: Array<ConstraintDecoratorArg<T>>): MethodDecorator {
  const options = normalizeConstraintArgs(args);

  return (target, propertyKey) => {
    const isStatic = typeof target === 'function';
    const ctor = (isStatic ? target : target.constructor) as ModelCtor<T> & typeof BaseModel;
    const meta = MetadataStorage.instance.getModelMetadata(ctor);
    const list: ConstraintMeta[] = [...(meta.constraintHandlers || [])];

    list.push({
      method: String(propertyKey),
      fields: options.fields || [],
      preview: !!options.preview,
      alwaysOnCreate: typeof options.alwaysOnCreate === 'boolean' ? options.alwaysOnCreate : CONSTRAINT_DEFAULT_ALWAYS_ON_CREATE,
      priority: typeof options.priority === 'number' ? options.priority : CONSTRAINT_DEFAULT_PRIORITY,
      isStatic,
    });

    MetadataStorage.instance.setModelMetadata(ctor, { ...meta, constraintHandlers: list });
  };
}
