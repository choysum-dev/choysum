// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { currentBridgeFrame, withBridgeFrame } from './bridge';

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
