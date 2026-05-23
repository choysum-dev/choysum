// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { MetadataStorage } from '../metadata/storage';
import { Constraint } from './constraint';

class ConstraintDecoratorArgModel extends BaseModel {
  validate() {}
}

test('constraint decorator normalizes mixed args and ignores empty items', () => {
  const decorate = Constraint(
    null as any,
    [' Name ', '', 0 as any] as any,
    'Code' as any,
    {
      fields: [' Code ', ' Age ', '' as any] as any,
      preview: true,
      priority: 7,
    } as any
  );

  decorate(ConstraintDecoratorArgModel.prototype, 'validate', Object.getOwnPropertyDescriptor(ConstraintDecoratorArgModel.prototype, 'validate')!);

  const meta = MetadataStorage.instance.getModelMetadata(ConstraintDecoratorArgModel as any);
  const handler = (meta.constraintHandlers || []).find((item: any) => item.method === 'validate');

  expect(handler).toBeDefined();
  expect(handler?.fields).toEqual(['Name', 'Code', 'Age']);
  expect(handler?.preview).toBe(true);
  expect(handler?.priority).toBe(7);
  expect(handler?.alwaysOnCreate).toBe(true);
  expect(handler?.isStatic).toBe(false);
});

test('constraint decorator initializes handler list when metadata has no preloaded handlers', () => {
  const storage = MetadataStorage.instance as any;
  const originalGetModelMetadata = storage.getModelMetadata;
  const originalSetModelMetadata = storage.setModelMetadata;
  let writtenMeta: any;

  class ConstraintDecoratorNoPreloadedModel extends BaseModel {
    check() {}
  }

  try {
    storage.getModelMetadata = (() => ({
      name: 'ConstraintDecoratorNoPreloadedModel',
      modelName: '',
      fullModelName: '',
      className: 'ConstraintDecoratorNoPreloadedModel',
      tableName: () => '',
      type: ConstraintDecoratorNoPreloadedModel,
      fields: new Map(),
      services: new Map(),
    })) as any;

    storage.setModelMetadata = ((_ctor: any, meta: any) => {
      writtenMeta = meta;
    }) as any;

    const decorate = Constraint({ preview: false } as any);
    decorate(ConstraintDecoratorNoPreloadedModel.prototype, 'check', Object.getOwnPropertyDescriptor(ConstraintDecoratorNoPreloadedModel.prototype, 'check')!);

    expect(Array.isArray(writtenMeta.constraintHandlers)).toBe(true);
    expect(writtenMeta.constraintHandlers).toHaveLength(1);
    expect(writtenMeta.constraintHandlers[0]).toEqual({
      method: 'check',
      fields: [],
      preview: false,
      alwaysOnCreate: true,
      priority: 20,
      isStatic: false,
    });
  } finally {
    storage.getModelMetadata = originalGetModelMetadata;
    storage.setModelMetadata = originalSetModelMetadata;
  }
});
