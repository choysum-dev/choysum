// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyChoyThemePreference,
  persistChoyThemePreference,
  readChoyThemePreference,
  resolveChoyThemePreference,
  type ChoyThemePreference,
} from './applyChoyThemePreference';

test('resolveChoyThemePreference maps standard density and auto theme', () => {
  expect(resolveChoyThemePreference({ theme: 'dark', density: 'standard' })).toEqual({
    theme: 'dark',
    density: 'comfortable',
    dark: true,
  });
  expect(
    resolveChoyThemePreference({ theme: 'auto', density: 'compact' }, { prefersDark: true }),
  ).toEqual({
    theme: 'auto',
    density: 'compact',
    dark: true,
  });
  expect(
    resolveChoyThemePreference({ theme: 'auto' }, { prefersDark: false }),
  ).toEqual({
    theme: 'auto',
    density: 'comfortable',
    dark: false,
  });
  expect(resolveChoyThemePreference(null)).toEqual({
    theme: 'light',
    density: 'comfortable',
    dark: false,
  });
});

test('read / persist round-trip via memory storage', () => {
  const mem = new Map<string, string>();
  const storage = {
    getItem: (k: string) => mem.get(k) ?? null,
    setItem: (k: string, v: string) => {
      mem.set(k, v);
    },
  };
  const prefs: ChoyThemePreference = { theme: 'dark', density: 'compact' };
  persistChoyThemePreference(prefs, storage);
  expect(readChoyThemePreference(storage)).toEqual(prefs);
});

test('readChoyThemePreference ignores corrupt JSON', () => {
  const storage = {
    getItem: () => '{',
    setItem: () => undefined,
  };
  expect(readChoyThemePreference(storage)).toEqual({});
});

test('readChoyThemePreference rejects JSON arrays', () => {
  expect(
    readChoyThemePreference({
      getItem: () => '[]',
    }),
  ).toEqual({});
});

test('applyChoyThemePreference toggles root class and density attr', () => {
  const classes = new Set<string>();
  const attrs = new Map<string, string>();
  const root = {
    classList: {
      toggle(name: string, force?: boolean) {
        if (force) classes.add(name);
        else classes.delete(name);
      },
    },
    setAttribute(name: string, value: string) {
      attrs.set(name, value);
    },
    removeAttribute(name: string) {
      attrs.delete(name);
    },
  };
  const mem = new Map<string, string>();
  const storage = {
    getItem: (k: string) => mem.get(k) ?? null,
    setItem: (k: string, v: string) => {
      mem.set(k, v);
    },
  };

  const dark = applyChoyThemePreference(
    { theme: 'dark', density: 'compact' },
    { root: root as never, storage, persist: true },
  );
  expect(dark.dark).toBe(true);
  expect(classes.has('dark')).toBe(true);
  expect(attrs.get('data-density')).toBe('compact');
  expect(JSON.parse(mem.get('choy.ui.theme')!)).toEqual({
    theme: 'dark',
    density: 'compact',
  });

  applyChoyThemePreference(
    { theme: 'light', density: 'comfortable' },
    { root: root as never, storage, persist: true },
  );
  expect(classes.has('dark')).toBe(false);
  expect(attrs.has('data-density')).toBe(false);
});

test('applyChoyThemePreference honors persist:false, prefersDark and storageKey', () => {
  const mem = new Map<string, string>();
  const storage = {
    getItem: (k: string) => mem.get(k) ?? null,
    setItem: (k: string, v: string) => {
      mem.set(k, v);
    },
  };
  const root = {
    classList: { toggle: () => undefined },
    setAttribute: () => undefined,
    removeAttribute: () => undefined,
  };
  const resolved = applyChoyThemePreference(
    { theme: 'auto', density: 'standard' },
    {
      root: root as never,
      storage,
      persist: false,
      prefersDark: true,
      storageKey: 'custom.key',
    },
  );
  expect(resolved).toEqual({ theme: 'auto', density: 'comfortable', dark: true });
  expect(mem.size).toBe(0);
});

test('applyChoyThemePreference persists original density including standard', () => {
  const mem = new Map<string, string>();
  const storage = {
    getItem: (k: string) => mem.get(k) ?? null,
    setItem: (k: string, v: string) => {
      mem.set(k, v);
    },
  };
  applyChoyThemePreference(
    { theme: 'light', density: 'standard' },
    {
      root: {
        classList: { toggle: () => undefined },
        setAttribute: () => undefined,
        removeAttribute: () => undefined,
      } as never,
      storage,
      persist: true,
    },
  );
  expect(JSON.parse(mem.get('choy.ui.theme')!)).toEqual({
    theme: 'light',
    density: 'standard',
  });
});

test('readChoyThemePreference returns {} when storage access throws', () => {
  expect(
    readChoyThemePreference({
      getItem: () => {
        throw new Error('denied');
      },
    }),
  ).toEqual({});
});

test('persistChoyThemePreference swallows storage write failures', () => {
  expect(() =>
    persistChoyThemePreference(
      { theme: 'dark', density: 'compact' },
      {
        setItem: () => {
          throw new Error('quota');
        },
      },
    ),
  ).not.toThrow();
});

test('applyChoyThemePreference keeps a stored theme when only density is applied', () => {
  const mem = new Map<string, string>([['choy.ui.theme', JSON.stringify({ theme: 'dark' })]]);
  const storage = {
    getItem: (k: string) => mem.get(k) ?? null,
    setItem: (k: string, v: string) => {
      mem.set(k, v);
    },
  };
  const classes = new Set<string>();
  applyChoyThemePreference(
    { density: 'compact' },
    {
      root: {
        classList: {
          toggle(name: string, force?: boolean) {
            if (force) classes.add(name);
            else classes.delete(name);
          },
        },
        setAttribute: () => undefined,
        removeAttribute: () => undefined,
      } as never,
      storage,
      persist: true,
    },
  );
  expect(classes.has('dark')).toBe(true);
  expect(JSON.parse(mem.get('choy.ui.theme')!)).toEqual({
    theme: 'dark',
    density: 'compact',
  });
});
