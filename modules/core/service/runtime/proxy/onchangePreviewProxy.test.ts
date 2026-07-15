// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../orm/model/model';
import type { ModelMetadata } from '../../orm/metadata/model';
import { createPreviewProxy, type PreviewProxyCtx } from './onchangePreviewProxy';

class PreviewTestModel extends BaseModel {}

function makeMeta(fields: Map<string, any>): ModelMetadata {
  return {
    name: 'PreviewTestModel',
    modelName: 'preview.test',
    fullModelName: 'core.preview.test',
    type: PreviewTestModel,
    tableName: () => 'preview_test',
    fields,
    services: new Map(),
  } as unknown as ModelMetadata;
}

function makeCtx(fields: Map<string, any>, overrides?: Partial<PreviewProxyCtx>): PreviewProxyCtx {
  return {
    meta: makeMeta(fields),
    triggers: new Set<string>(),
    reads: new Set<string>(),
    loaded: new Set<string>(),
    ...overrides,
  };
}

test('preview proxy disables forbidden persistence methods and keeps safe methods', () => {
  const base = {
    update() {
      return 'should-not-run';
    },
    save() {
      return 'save-should-not-run';
    },
    Upsert() {
      return 'upsert-should-not-run';
    },
    hello() {
      return 'ok';
    },
  } as unknown as PreviewTestModel;

  const proxy = createPreviewProxy(base, makeCtx(new Map()));

  expect(() => (proxy as any).update()).toThrow('PREVIEW_METHOD_FORBIDDEN');
  expect(() => (proxy as any).update()).toThrow('method "update" is disabled in onchange preview');
  expect(() => (proxy as any).save()).toThrow('PREVIEW_METHOD_FORBIDDEN');
  expect(() => (proxy as any).Upsert()).toThrow('PREVIEW_METHOD_FORBIDDEN');
  expect((proxy as any).hello()).toBe('ok');
});

test('preview proxy warns and returns undefined when reading unloaded field without permission', () => {
  const base = {} as PreviewTestModel;
  const fields = new Map<string, any>([['Name', { type: 'varchar' }]]);
  const proxy = createPreviewProxy(base, makeCtx(fields));

  const originalWarn = console.warn;
  const warnings: string[] = [];
  console.warn = ((msg: unknown) => {
    warnings.push(String(msg));
  }) as unknown as typeof console.warn;

  try {
    expect((proxy as any).Name).toBe(undefined);
  } finally {
    console.warn = originalWarn;
  }

  expect(warnings).toHaveLength(1);
  expect(warnings[0]).toContain('reading unloaded field "Name"');
  expect(warnings[0]).toContain('PreviewTestModel');
});

test('preview proxy allows unloaded read when strategy permits or value already exists', () => {
  const fields = new Map<string, any>([['Name', { type: 'varchar' }]]);

  const strategyProxy = createPreviewProxy(
    {} as PreviewTestModel,
    makeCtx(fields, {
      pathAccessAllowed: rootField => rootField === 'Name',
    })
  );

  const presetProxy = createPreviewProxy({ Name: 'preset' } as unknown as PreviewTestModel, makeCtx(fields));

  const originalWarn = console.warn;
  let warnCount = 0;
  console.warn = (() => {
    warnCount += 1;
  }) as unknown as typeof console.warn;

  try {
    expect((strategyProxy as any).Name).toBe(undefined);
    expect((presetProxy as any).Name).toBe('preset');
  } finally {
    console.warn = originalWarn;
  }

  expect(warnCount).toBe(0);
});

test('preview proxy returns readonly empty array placeholder for unloaded or null collections', () => {
  const fields = new Map<string, any>([['Tags', { type: 'OneToMany' }]]);

  const unloaded = createPreviewProxy({ Tags: [{ Id: '1' }] } as unknown as PreviewTestModel, makeCtx(fields));

  const loadedNull = createPreviewProxy({ Tags: null } as unknown as PreviewTestModel, makeCtx(fields, { loaded: new Set(['Tags']) }));

  const loadedArray = createPreviewProxy({ Tags: [{ Id: '2' }] } as unknown as PreviewTestModel, makeCtx(fields, { loaded: new Set(['Tags']) }));

  expect((unloaded as any).Tags).toEqual([]);
  expect((loadedNull as any).Tags).toEqual([]);
  expect((loadedArray as any).Tags).toEqual([{ Id: '2' }]);
});

test('preview proxy set trap passes through symbols non-fields and model fields', () => {
  const sym = Symbol('internal');
  const fields = new Map<string, any>([['Name', { type: 'varchar' }]]);
  const base = {} as any;
  const proxy = createPreviewProxy(base as PreviewTestModel, makeCtx(fields));

  (proxy as any)[sym] = 1;
  (proxy as any).TempFlag = true;
  (proxy as any).Name = 'Alice';

  expect(base[sym]).toBe(1);
  expect(base.TempFlag).toBe(true);
  expect(base.Name).toBe('Alice');
});
