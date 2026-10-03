// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Minimal class-variance-authority compatible helper for vendored L2.
 * Avoids adding `class-variance-authority` as a runtime peer; API matches
 * the `cva` / `VariantProps` surface used by shadcn-vue registry index files.
 */

import type { ClassValue } from './utils';

type VariantConfig = Record<string, Record<string, string>>;

type CvaConfig<V extends VariantConfig> = {
  variants?: V;
  defaultVariants?: { [K in keyof V]?: keyof V[K] & string };
};

type PropsOf<V extends VariantConfig> = {
  [K in keyof V]?: keyof V[K] & string;
} & {
  class?: ClassValue;
};

export type VariantProps<T extends (props?: any) => string> = NonNullable<Parameters<T>[0]>;

function flatten(value: ClassValue | undefined, out: string[]): void {
  if (value === false || value === null || value === undefined || value === '') {
    return;
  }
  if (typeof value === 'number') {
    out.push(String(value));
    return;
  }
  if (typeof value === 'string') {
    out.push(value);
    return;
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      flatten(item, out);
    }
    return;
  }
  if (typeof value === 'object') {
    for (const [key, on] of Object.entries(value)) {
      if (on) {
        out.push(key);
      }
    }
  }
}

/**
 * Builds a variant class resolver: base classes + selected variant keys.
 */
export function cva<V extends VariantConfig>(base: string, config: CvaConfig<V> = {}) {
  const variants = (config.variants || {}) as V;
  const defaults = (config.defaultVariants || {}) as { [K in keyof V]?: keyof V[K] & string };

  return (props?: PropsOf<V>): string => {
    const classes: string[] = [];
    flatten(base, classes);
    for (const key of Object.keys(variants) as (keyof V & string)[]) {
      const selected = (props?.[key] ?? defaults[key]) as string | undefined;
      if (!selected) {
        continue;
      }
      const chunk = variants[key]?.[selected];
      if (chunk) {
        flatten(chunk, classes);
      }
    }
    flatten(props?.class, classes);
    return classes.join(' ');
  };
}
