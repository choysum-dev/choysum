// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { currentBridgeFrame, withBridgeFrame } from './bridge';

test('bridge contexts isolate payloads across concurrent calls', async () => {
  const first = {};
  const second = {};

  const p1 = withBridgeFrame(first, 'search', { token: 'first' }, async () => {
    await Promise.resolve();
    const frame = currentBridgeFrame<{ token: string }>(first, 'search');
    return frame.token;
  });

  const p2 = withBridgeFrame(second, 'search', { token: 'second' }, async () => {
    await Promise.resolve();
    const frame = currentBridgeFrame<{ token: string }>(second, 'search');
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
  expect(() => currentBridgeFrame(instance, 'search')).toThrow('BRIDGE_CONTEXT_EXPIRED');
});

test('bridge context expires after execution returns', () => {
  const instance = {};

  const observed = withBridgeFrame(instance, 'sql', { value: 1 }, () => currentBridgeFrame<{ value: number }>(instance, 'sql').value);
  expect(observed).toBe(1);

  expect(() => currentBridgeFrame(instance, 'sql')).toThrow('BRIDGE_CONTEXT_EXPIRED');
});

test('bridge context enforces kind checks', () => {
  const instance = {};

  withBridgeFrame(instance, 'sql', { value: 1 }, () => {
    expect(() => currentBridgeFrame(instance, 'search')).toThrow('BRIDGE_CONTEXT_KIND_MISMATCH');
  });
});

test('bridge context resolves outer matching kind when inner frame uses another kind on same instance', () => {
  const instance = {};

  withBridgeFrame(instance, 'search', { token: 'outer' }, () => {
    withBridgeFrame(instance, 'inverse', { token: 'inner' }, () => {
      const searchFrame = currentBridgeFrame<{ token: string }>(instance, 'search');
      const inverseFrame = currentBridgeFrame<{ token: string }>(instance, 'inverse');
      expect(searchFrame.token).toBe('outer');
      expect(inverseFrame.token).toBe('inner');
    });
  });
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

  const first = withBridgeFrame(instance, 'sql', { token: 'first' }, () => currentBridgeFrame<{ token: string }>(instance, 'sql').token);
  expect(first).toBe('first');
  // Frame is expired after first call completes
  expect(() => currentBridgeFrame(instance, 'sql')).toThrow('BRIDGE_CONTEXT_EXPIRED');

  const second = withBridgeFrame(instance, 'sql', { token: 'second' }, () => currentBridgeFrame<{ token: string }>(instance, 'sql').token);
  expect(second).toBe('second');
});
