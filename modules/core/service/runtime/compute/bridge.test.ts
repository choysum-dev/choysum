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

test('bridge context enforces instance checks', () => {
  const first = {};
  const second = {};

  withBridgeFrame(first, 'inverse', { value: 1 }, () => {
    expect(() => currentBridgeFrame(second, 'inverse')).toThrow('BRIDGE_CONTEXT_INSTANCE_MISMATCH');
  });
});
