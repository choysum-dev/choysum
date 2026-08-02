// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { applyFieldColumnDefaults, DefaultOperations } from './model_default';
import { MetadataStorage } from '../metadata/storage';

function createModelWithDefaults(defs: Record<string, any>) {
  class DefaultOpsModel extends BaseModel {}
  const fields = new Map<string, any>();
  for (const [name, def] of Object.entries(defs)) {
    fields.set(name, { name, type: 'varchar', column: { default: def } });
  }
  MetadataStorage.instance.setModelMetadata(DefaultOpsModel as any, { fields } as any);
  return DefaultOpsModel;
}

test('DefaultOperations applies literal and function defaults with dependency order', async () => {
  const ModelCtor = createModelWithDefaults({
    Code: 'A-001',
    Name: (self: any) => `N-${self.Code}`,
    Label: (self: any) => `${self.Name}-ok`,
  });

  const out = await DefaultOperations.DefaultGet(ModelCtor as any, {} as any);
  expect(out).toEqual({
    Code: 'A-001',
    Name: 'N-A-001',
    Label: 'N-A-001-ok',
  });
});

test('DefaultOperations keeps provided values and only fills missing defaults', async () => {
  const ModelCtor = createModelWithDefaults({
    Code: 'A-001',
    Name: (self: any) => `N-${self.Code}`,
  });

  const out = await DefaultOperations.DefaultGet(
    ModelCtor as any,
    {
      Code: 'X-9',
    } as any
  );

  expect(out).toEqual({
    Code: 'X-9',
    Name: 'N-X-9',
  });
});

test('DefaultOperations skips field default when dependency is missing', async () => {
  const ModelCtor = createModelWithDefaults({
    Name: (self: any) => `N-${self.UnknownField}`,
    Code: 'B-002',
  });

  const out = await DefaultOperations.DefaultGet(ModelCtor as any, {} as any);
  expect((out as any).Name).toBeUndefined();
  expect((out as any).Code).toBe('B-002');
});

test('DefaultOperations detects circular dependencies between default functions', async () => {
  const ModelCtor = createModelWithDefaults({
    A: (self: any) => `A-${self.B}`,
    B: (self: any) => `B-${self.A}`,
  });

  let err: any;
  try {
    await DefaultOperations.DefaultGet(ModelCtor as any, {} as any);
  } catch (e) {
    err = e;
  }

  expect(Boolean(err)).toBe(true);
  expect(String(err?.message || '')).toContain('Circular dependency detected');
  expect(String(err?.message || '')).toContain('depends on in-progress field');
});

test('DefaultOperations propagates non-missing dependency errors from default function', async () => {
  const ModelCtor = createModelWithDefaults({
    Name: () => {
      throw new Error('custom-default-error');
    },
  });

  let err: any;
  try {
    await DefaultOperations.DefaultGet(ModelCtor as any, {} as any);
  } catch (e) {
    err = e;
  }

  expect(Boolean(err)).toBe(true);
  expect(String(err?.message || '')).toContain('custom-default-error');
});

test('applyFieldColumnDefaults no-ops when dependency pass already processed a later field', async () => {
  // Name is visited first and pulls Code; the later forEach entry for Code hits the processedFields guard.
  const ModelCtor = createModelWithDefaults({
    Name: (self: any) => `N-${self.Code}`,
    Code: 'C-1',
  });

  const out = await applyFieldColumnDefaults(ModelCtor as any, {} as any);
  expect(out).toEqual({
    Name: 'N-C-1',
    Code: 'C-1',
  });
});

test('applyFieldColumnDefaults skips field when default becomes undefined before process', async () => {
  class DefaultDisappearsModel extends BaseModel {}
  let reads = 0;
  const column: Record<string, unknown> = {};
  Object.defineProperty(column, 'default', {
    enumerable: true,
    configurable: true,
    get() {
      reads += 1;
      // Pending collection sees a default; processField sees undefined and returns early.
      return reads === 1 ? 'seed' : undefined;
    },
  });
  const fields = new Map<string, any>([['Name', { name: 'Name', type: 'varchar', column }]]);
  MetadataStorage.instance.setModelMetadata(DefaultDisappearsModel as any, { fields } as any);

  const out = await applyFieldColumnDefaults(DefaultDisappearsModel as any, {} as any);
  expect((out as any).Name).toBeUndefined();
  expect(reads).toBeGreaterThanOrEqual(2);
});
