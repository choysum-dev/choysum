// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ONCHANGE_DEFAULT_PRIORITY } from '../../runtime/onchange/constants';
import { MetadataStorage } from '../metadata/storage';
import BaseModel from '../model/model';
import { Onchange } from './onchange';

class OnchangeDecoratorModelWithPriority extends BaseModel {
  handle() {}
}

class OnchangeDecoratorModelWithDefaultPriority extends BaseModel {
  handle() {}
}

class OnchangeDecoratorModelWithoutPreloadedHandlers extends BaseModel {
  handle() {}
}

test('onchange decorator normalizes triggers and reads with explicit numeric priority', () => {
  const decorate = Onchange(
    undefined as any,
    ['Name', '', 0 as any, 'Name', 'Qty'] as any,
    'State' as any,
    { priority: 7, reads: [' PartnerId.Name ', '', 'PartnerId.Name', 0 as any, 'Lines.Product'] as any } as any
  );

  decorate(
    OnchangeDecoratorModelWithPriority.prototype,
    'handle' as any,
    Object.getOwnPropertyDescriptor(OnchangeDecoratorModelWithPriority.prototype, 'handle')!
  );

  const meta = MetadataStorage.instance.getModelMetadata(OnchangeDecoratorModelWithPriority as any);
  const handler = (meta.onchangeHandlers || []).find(item => item.method === 'handle');

  expect(handler).toBeDefined();
  expect(handler?.triggers).toEqual(['Name', 'Qty', 'State']);
  expect(handler?.reads).toEqual(['PartnerId.Name', 'Lines.Product']);
  expect(handler?.priority).toBe(7);
});

test('onchange decorator uses default priority when options are omitted', () => {
  const decorate = Onchange('Amount' as any, ['Currency'] as any);
  decorate(
    OnchangeDecoratorModelWithDefaultPriority.prototype,
    'handle' as any,
    Object.getOwnPropertyDescriptor(OnchangeDecoratorModelWithDefaultPriority.prototype, 'handle')!
  );

  const meta = MetadataStorage.instance.getModelMetadata(OnchangeDecoratorModelWithDefaultPriority as any);
  const handler = (meta.onchangeHandlers || []).find(item => item.method === 'handle');

  expect(handler).toBeDefined();
  expect(handler?.triggers).toEqual(['Amount', 'Currency']);
  expect(handler?.priority).toBe(ONCHANGE_DEFAULT_PRIORITY);
  expect(handler?.reads).toBeUndefined();
});

test('onchange decorator tolerates metadata without onchangeHandlers list', () => {
  const storage = MetadataStorage.instance as any;
  const originalGetModelMetadata = storage.getModelMetadata;
  const originalSetModelMetadata = storage.setModelMetadata;
  let writtenMeta: any;

  try {
    storage.getModelMetadata = (() => ({
      name: 'OnchangeDecoratorModelWithoutPreloadedHandlers',
      modelName: '',
      fullModelName: '',
      className: 'OnchangeDecoratorModelWithoutPreloadedHandlers',
      tableName: () => '',
      type: OnchangeDecoratorModelWithoutPreloadedHandlers,
      fields: new Map(),
      services: new Map(),
    })) as any;

    storage.setModelMetadata = ((_ctor: any, meta: any) => {
      writtenMeta = meta;
    }) as any;

    const decorate = Onchange('State' as any);
    decorate(
      OnchangeDecoratorModelWithoutPreloadedHandlers.prototype,
      'handle' as any,
      Object.getOwnPropertyDescriptor(OnchangeDecoratorModelWithoutPreloadedHandlers.prototype, 'handle')!
    );

    expect(writtenMeta).toBeDefined();
    expect(Array.isArray(writtenMeta.onchangeHandlers)).toBe(true);
    expect(writtenMeta.onchangeHandlers.length).toBe(1);
    expect(writtenMeta.onchangeHandlers[0]).toEqual({
      method: 'handle',
      triggers: ['State'],
      priority: ONCHANGE_DEFAULT_PRIORITY,
      reads: undefined,
      signature: undefined,
    });
  } finally {
    storage.getModelMetadata = originalGetModelMetadata;
    storage.setModelMetadata = originalSetModelMetadata;
  }
});

class OnchangeDecoratorModelWithExplicitSignature extends BaseModel {
  handle() {}
}

class OnchangeDecoratorModelWithDefaultSignature extends BaseModel {
  handle() {}
}

test('onchange decorator writes explicit signature to metadata', () => {
  const decorate = Onchange('Name' as any, { signature: 'instanceNoArgs' } as any) as (target: Object, key: string | symbol) => void;

  decorate(
    OnchangeDecoratorModelWithExplicitSignature.prototype,
    'handle' as any,
  );

  const meta = MetadataStorage.instance.getModelMetadata(OnchangeDecoratorModelWithExplicitSignature as any);
  const handler = (meta.onchangeHandlers || []).find(item => item.method === 'handle');

  expect(handler).toBeDefined();
  expect(handler?.signature).toBe('instanceNoArgs');
  expect(handler?.priority).toBe(ONCHANGE_DEFAULT_PRIORITY);
});

test('onchange decorator omits signature when not specified (raw metadata)', () => {
  const decorate = Onchange('Value' as any) as (target: Object, key: string | symbol) => void;

  decorate(
    OnchangeDecoratorModelWithDefaultSignature.prototype,
    'handle' as any,
  );

  const meta = MetadataStorage.instance.getModelMetadata(OnchangeDecoratorModelWithDefaultSignature as any);
  const handler = (meta.onchangeHandlers || []).find(item => item.method === 'handle');

  expect(handler).toBeDefined();
  expect(handler?.signature).toBeUndefined();
});
