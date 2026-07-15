// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../orm/model/model';
import type { ModelMetadata } from '../../orm/metadata/model';
import { createOnchangeDraft, createPreviewProxy, createWriteProxy, getProxyKind, markProxyKind } from '../proxy';
import { currentBridgeFrame, withBridgeFrame } from './bridge';

class BridgeIsolationModel extends BaseModel {}

function makeBridgeIsolationBase(): BridgeIsolationModel {
  return Object.create(BridgeIsolationModel.prototype) as BridgeIsolationModel;
}

function makePreviewMeta(): ModelMetadata {
  return {
    name: 'BridgeIsolationModel',
    modelName: 'BridgeIsolationModel',
    fullModelName: 'test.BridgeIsolationModel',
    type: BridgeIsolationModel,
    tableName: () => 'bridge_isolation',
    fields: new Map([['Name', { type: 'varchar' }]]),
    services: new Map(),
  } as unknown as ModelMetadata;
}

test('bridge contexts isolate payloads across concurrent calls', async () => {
  const first = {};
  const second = {};

  const p1 = withBridgeFrame(first, 'search', { token: 'first' }, async executionInstance => {
    await Promise.resolve();
    const frame = currentBridgeFrame<{ token: string }>(executionInstance, 'search');
    return frame.token;
  });

  const p2 = withBridgeFrame(second, 'search', { token: 'second' }, async executionInstance => {
    await Promise.resolve();
    const frame = currentBridgeFrame<{ token: string }>(executionInstance, 'search');
    return frame.token;
  });

  const [left, right] = await Promise.all([p1, p2]);
  expect(left).toBe('first');
  expect(right).toBe('second');
});

test('bridge context supports thenables without finally', async () => {
  const instance = {};

  const result = withBridgeFrame(instance, 'search', { token: 'thenable' }, () => ({
    then(resolve: (value: string) => void) {
      resolve('ok');
    },
  }));

  const settled = await Promise.resolve(result);
  expect(settled).toBe('ok');
  expect(() => currentBridgeFrame(instance, 'search')).toThrow('BRIDGE_CONTEXT_UNAVAILABLE');
});

test('bridge context expires after execution returns', () => {
  const instance = {};

  const observed = withBridgeFrame(instance, 'sql', { value: 1 }, executionInstance => currentBridgeFrame<{ value: number }>(executionInstance, 'sql').value);
  expect(observed).toBe(1);

  expect(() => currentBridgeFrame(instance, 'sql')).toThrow('BRIDGE_CONTEXT_UNAVAILABLE');
});

test('bridge context enforces kind checks', () => {
  const instance = {};

  withBridgeFrame(instance, 'sql', { value: 1 }, executionInstance => {
    expect(() => currentBridgeFrame(executionInstance, 'search')).toThrow('BRIDGE_CONTEXT_KIND_MISMATCH');
  });
});

test('bridge context resolves outer matching kind when inner frame uses another kind on same instance', () => {
  const instance = {};

  withBridgeFrame(instance, 'search', { token: 'outer' }, outerExecutionInstance => {
    withBridgeFrame(instance, 'inverse', { token: 'inner' }, innerExecutionInstance => {
      const searchFrame = currentBridgeFrame<{ token: string }>(outerExecutionInstance, 'search');
      const inverseFrame = currentBridgeFrame<{ token: string }>(innerExecutionInstance, 'inverse');
      expect(searchFrame.token).toBe('outer');
      expect(inverseFrame.token).toBe('inner');
    });
  });
});

test('bridge contexts isolate payloads across concurrent calls on the same instance', async () => {
  const instance = {};

  const p1 = withBridgeFrame(instance, 'search', { token: 'first' }, async executionInstance => {
    await Promise.resolve();
    const frame = currentBridgeFrame<{ token: string }>(executionInstance, 'search');
    return frame.token;
  });

  const p2 = withBridgeFrame(instance, 'search', { token: 'second' }, async executionInstance => {
    await Promise.resolve();
    const frame = currentBridgeFrame<{ token: string }>(executionInstance, 'search');
    return frame.token;
  });

  const [left, right] = await Promise.all([p1, p2]);
  expect(left).toBe('first');
  expect(right).toBe('second');
});

test('bridge context enforces instance checks', () => {
  const first = {};
  const second = {};

  withBridgeFrame(first, 'inverse', { value: 1 }, () => {
    expect(() => currentBridgeFrame(second, 'inverse')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
  });
});

test('bridge context rejects unknown kind', () => {
  const instance = {};
  expect(() => {
    withBridgeFrame(instance, 'unknown' as any, {}, () => {});
  }).toThrow('BRIDGE_CONTEXT_UNAVAILABLE');
});

test('bridge context rejects non-object instance', () => {
  expect(() => {
    withBridgeFrame(null as any, 'sql', {}, () => {});
  }).toThrow('BRIDGE_CONTEXT_UNAVAILABLE');

  expect(() => {
    withBridgeFrame(42 as any, 'sql', {}, () => {});
  }).toThrow('BRIDGE_CONTEXT_UNAVAILABLE');
});

test('bridge context isolates the same instance across sequential calls', () => {
  const instance = {};

  const first = withBridgeFrame(
    instance,
    'sql',
    { token: 'first' },
    executionInstance => currentBridgeFrame<{ token: string }>(executionInstance, 'sql').token
  );
  expect(first).toBe('first');
  // Frame is isolated to execution instance and unavailable from original instance.
  expect(() => currentBridgeFrame(instance, 'sql')).toThrow('BRIDGE_CONTEXT_UNAVAILABLE');

  const second = withBridgeFrame(
    instance,
    'sql',
    { token: 'second' },
    executionInstance => currentBridgeFrame<{ token: string }>(executionInstance, 'sql').token
  );
  expect(second).toBe('second');
});

test('bridge context rejects onchange-write draft identity for currentBridgeFrame and $search', () => {
  const base = makeBridgeIsolationBase();
  const writeDraft = createWriteProxy(base, () => {});
  expect(getProxyKind(writeDraft)).toBe('onchange-write');

  withBridgeFrame(base, 'search', { token: 'active' }, executionInstance => {
    expect(currentBridgeFrame<{ token: string }>(executionInstance, 'search').token).toBe('active');
    expect(() => currentBridgeFrame(writeDraft as object, 'search')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');

    let searchError: unknown;
    try {
      void (writeDraft as BridgeIsolationModel).$search;
    } catch (err) {
      searchError = err;
    }
    expect(String((searchError as Error)?.message || searchError || '')).toContain('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
  });
});

test('bridge context rejects onchange-preview and composed onchange draft identity', () => {
  const base = makeBridgeIsolationBase();
  const meta = makePreviewMeta();
  const preview = createPreviewProxy(base, {
    meta,
    triggers: new Set<string>(),
    reads: new Set<string>(['Name']),
    loaded: new Set<string>(['Name']),
  });
  expect(getProxyKind(preview)).toBe('onchange-preview');

  const composed = createOnchangeDraft(base, {
    meta,
    triggers: new Set<string>(),
    reads: new Set<string>(['Name']),
    loaded: new Set<string>(['Name']),
    patchSink: () => {},
  });
  expect(getProxyKind(composed)).toBe('onchange-preview');

  withBridgeFrame(base, 'sql', { token: 'sql-active' }, executionInstance => {
    expect(currentBridgeFrame<{ token: string }>(executionInstance, 'sql').token).toBe('sql-active');

    expect(() => currentBridgeFrame(preview as object, 'sql')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
    expect(() => currentBridgeFrame(composed as object, 'sql')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');

    let previewSqlError: unknown;
    try {
      void (preview as BridgeIsolationModel).$sql;
    } catch (err) {
      previewSqlError = err;
    }
    expect(String((previewSqlError as Error)?.message || previewSqlError || '')).toContain('BRIDGE_CONTEXT_INSTANCE_MISMATCH');

    let composedInverseError: unknown;
    try {
      void (composed as BridgeIsolationModel).$inverse;
    } catch (err) {
      composedInverseError = err;
    }
    expect(String((composedInverseError as Error)?.message || composedInverseError || '')).toContain('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
  });
});

test('bridge context rejects constraint-draft identity for currentBridgeFrame and $search', () => {
  const base = makeBridgeIsolationBase();
  const constraintDraft = new Proxy(base, {}) as BridgeIsolationModel;
  markProxyKind(constraintDraft as object, 'constraint-draft');
  expect(getProxyKind(constraintDraft)).toBe('constraint-draft');

  withBridgeFrame(base, 'search', { token: 'constraint-peer' }, executionInstance => {
    expect(currentBridgeFrame<{ token: string }>(executionInstance, 'search').token).toBe('constraint-peer');
    expect(() => currentBridgeFrame(constraintDraft as object, 'search')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');

    let searchError: unknown;
    try {
      void constraintDraft.$search;
    } catch (err) {
      searchError = err;
    }
    expect(String((searchError as Error)?.message || searchError || '')).toContain('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
  });
});

test('bridge execution instance remains the only identity that resolves an active frame', () => {
  const base = makeBridgeIsolationBase();
  const draft = createWriteProxy(base, () => {});

  withBridgeFrame(base, 'inverse', { token: 'only-execution' }, executionInstance => {
    expect(getProxyKind(executionInstance)).toBe('bridge-execution');
    expect(currentBridgeFrame<{ token: string }>(executionInstance, 'inverse').token).toBe('only-execution');
    expect(() => currentBridgeFrame(base as object, 'inverse')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
    expect(() => currentBridgeFrame(draft as object, 'inverse')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
  });
});
