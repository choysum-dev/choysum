// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createWriteProxy } from './onchangeDraftProxy';
import { getProxyKind } from './brand';

test('createWriteProxy records nested set and delete operations with full paths', () => {
  const patches: Array<{ path: string; value: any }> = [];
  const state = {
    profile: {
      name: 'Alice',
      age: 18,
    },
  };

  const proxy = createWriteProxy(state, (path, value) => {
    patches.push({ path, value });
  });

  proxy.profile.name = 'Bob';
  delete (proxy.profile as any).age;

  expect(patches).toEqual([
    { path: 'profile.name', value: 'Bob' },
    { path: 'profile.age', value: undefined },
  ]);
});

test('createWriteProxy wraps array mutating methods and reports parent path', () => {
  const patches: Array<{ path: string; value: any }> = [];
  const state = {
    items: [1, 2],
  };

  const proxy = createWriteProxy(state, (path, value) => {
    patches.push({ path, value: Array.isArray(value) ? [...value] : value });
  });

  proxy.items.push(3);
  proxy.items.splice(1, 1, 9);
  proxy.items.pop();

  expect(patches.map(p => p.path)).toEqual(['items', 'items', 'items']);
  expect(patches[0].value).toEqual([1, 2, 3]);
  expect(patches[1].value).toEqual([1, 9, 3]);
  expect(patches[2].value).toEqual([1, 9]);
});

test('createWriteProxy ignores array length internal writes', () => {
  const patches: Array<{ path: string; value: any }> = [];
  const state = {
    items: [1, 2, 3],
  };

  const proxy = createWriteProxy(state, (path, value) => {
    patches.push({ path, value });
  });

  proxy.items.length = 1;

  expect(proxy.items).toEqual([1]);
  expect(patches).toEqual([]);
});

test('createWriteProxy unwraps proxy value on assignment and supports basePath', () => {
  const patches: Array<{ path: string; value: any }> = [];
  const state = {
    left: { v: 1 },
    right: { v: 2 },
  };

  const proxy = createWriteProxy(
    state,
    (path, value) => {
      patches.push({ path, value });
    },
    'root'
  );

  proxy.left = proxy.right as any;

  expect(patches).toHaveLength(1);
  expect(patches[0].path).toBe('root.left');
  expect(patches[0].value).toBe(state.right);
});

test('createWriteProxy supports symbol keys and records normalized symbol path', () => {
  const patches: Array<{ path: string; value: any }> = [];
  const key = Symbol('flag');
  const state: any = {
    bucket: {},
  };

  const proxy: any = createWriteProxy(state, (path, value) => {
    patches.push({ path, value });
  });

  proxy.bucket[key] = 1;
  delete proxy.bucket[key];

  expect(patches).toEqual([
    { path: 'bucket.Symbol(flag)', value: 1 },
    { path: 'bucket.Symbol(flag)', value: undefined },
  ]);
});

test('createWriteProxy passes through non-object root values', () => {
  const patches: Array<{ path: string; value: any }> = [];
  const proxy = createWriteProxy(123 as any, (path, value) => {
    patches.push({ path, value });
  });

  expect(proxy).toBe(123);
  expect(patches).toEqual([]);
});

test('createWriteProxy brands nested object and array proxies as onchange-write', () => {
  const state = {
    profile: { name: 'Alice' },
    items: [{ id: 1 }],
  };
  const proxy = createWriteProxy(state, () => {});

  expect(getProxyKind(proxy)).toBe('onchange-write');
  expect(getProxyKind(proxy.profile)).toBe('onchange-write');
  expect(getProxyKind(proxy.items)).toBe('onchange-write');
  expect(getProxyKind(proxy.items[0])).toBe('onchange-write');
});
