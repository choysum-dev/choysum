// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getContextLang } from '../runtime/context/scope';
import { resolveI18nScope, withI18nScope, type ResolveI18nScopeOptions } from './scope';
import { resolveRequestLang } from './request_lang';

export type TranslateOptions = ResolveI18nScopeOptions & { kind?: string };

export type TermIdentity = {
  module: string;
  scope: string;
  src: string;
  kind: string;
};

export type TextDescriptor = {
  key: string;
  module: string;
  scope: string;
  src: string;
  kind: 'literal';
};

export const TEXT_DESCRIPTOR_NAMESPACE = '__terms';

/**
 * Build the canonical, runtime-independent identity shared by terminology
 * producers and consumers.
 */
export function createTermIdentity(
  module: string,
  src: string,
  opts?: TranslateOptions
): TermIdentity {
  return {
    module: String(module || '').trim(),
    scope: resolveI18nScope(opts).trim(),
    src: String(src),
    kind: String(opts?.kind || '').trim() || 'literal',
  };
}

function utf8Hex(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let encoded = '';
  for (const byte of bytes) {
    encoded += byte.toString(16).padStart(2, '0');
  }
  return encoded;
}

/**
 * Build the collision-free vue-i18n key for a terminology identity.
 *
 * Each identity component is UTF-8 byte-length-prefixed before the complete
 * identity is UTF-8 hex encoded. The resulting segment is JSON-safe and never
 * contains dots, so vue-i18n only parses the reserved namespace separator.
 */
export function createTextDescriptorKey(
  module: string,
  scope: string,
  src: string,
  kind = 'literal'
): string {
  const values = [module, scope, src, kind].map(value => String(value));
  const identity = values
    .map(value => `${new TextEncoder().encode(value).length}:${value}`)
    .join('');
  return `${TEXT_DESCRIPTOR_NAMESPACE}.${utf8Hex(identity)}`;
}

type ChoysumI18n = {
  t: (module: string, lang: string, scope: string, src: string, kind?: string) => string;
};

function getBridge(): ChoysumI18n | undefined {
  const root = globalThis as { $choysum?: { i18n?: ChoysumI18n } };
  return root.$choysum?.i18n;
}

function isTranslateOptions(value: unknown): value is TranslateOptions {
  return !!value && typeof value === 'object' && !Array.isArray(value) && ('scope' in (value as object) || 'kind' in (value as object) || 'path' in (value as object) || 'location' in (value as object));
}

function interpolate(template: string, args: unknown[]): string {
  if (!args.length) {
    return template;
  }
  let i = 0;
  return template.replace(/%s|%d|%%/g, (match) => {
    if (match === '%%') {
      return '%';
    }
    if (i >= args.length) {
      return match;
    }
    const v = args[i++];
    return v == null ? '' : String(v);
  });
}

function lookup(identity: TermIdentity): string {
  const fromServiceCtx = getContextLang()?.trim();
  const lang = fromServiceCtx || resolveRequestLang(undefined, { final: 'en_US' });
  const bridge = getBridge();
  if (!bridge || typeof bridge.t !== 'function') {
    return '';
  }
  try {
    return bridge.t(
      identity.module,
      lang,
      identity.scope,
      identity.src,
      identity.kind
    ) || '';
  } catch {
    return '';
  }
}

export type TranslateFn = {
  (src: string, ...args: unknown[]): string;
};

export type LazyTranslate = {
  toString(): string;
  valueOf(): string;
  [Symbol.toPrimitive](): string;
};

export function isTextDescriptor(value: unknown): value is TextDescriptor {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return false;
  }
  const descriptor = value as Partial<TextDescriptor>;
  return (
    typeof descriptor.module === 'string' &&
    typeof descriptor.scope === 'string' &&
    typeof descriptor.src === 'string' &&
    typeof descriptor.key === 'string' &&
    descriptor.key === createTextDescriptorKey(descriptor.module, descriptor.scope, descriptor.src, descriptor.kind) &&
    descriptor.kind === 'literal'
  );
}

export function createTextDescriptor(module: string, src: string, opts?: TranslateOptions): TextDescriptor {
  const identity = {
    ...createTermIdentity(module, src, opts),
    kind: 'literal' as const,
  };
  return {
    ...identity,
    key: createTextDescriptorKey(
      identity.module,
      identity.scope,
      identity.src,
      identity.kind
    ),
  };
}

/**
 * Bind `_t` / `_lt` to a module name (term owner module).
 */
export function createTranslate(module: string): {
  _t: TranslateFn;
  _lt: (src: string, opts?: TranslateOptions) => LazyTranslate;
  _td: (src: string, opts?: TranslateOptions) => TextDescriptor;
} {
  const mod = String(module || '').trim();

  const _t: TranslateFn = (src: string, ...args: unknown[]): string => {
    let opts: TranslateOptions | undefined;
    let interp: unknown[] = args;
    if (args.length > 0 && isTranslateOptions(args[0])) {
      opts = args[0] as TranslateOptions;
      interp = args.slice(1);
    }
    const identity = createTermIdentity(mod, src, opts);
    const hit = lookup(identity);
    const base = hit || identity.src;
    return interpolate(base, interp);
  };

  const _lt = (src: string, opts?: TranslateOptions): LazyTranslate => {
    const identity = createTermIdentity(mod, src, opts);
    const resolve = () => {
      const hit = lookup(identity);
      return hit || identity.src;
    };
    return {
      toString: resolve,
      valueOf: resolve,
      [Symbol.toPrimitive]: resolve,
    };
  };

  const _td = (src: string, opts?: TranslateOptions): TextDescriptor => createTextDescriptor(mod, src, opts);

  return { _t, _lt, _td };
}

export { withI18nScope, resolveI18nScope, formatScope } from './scope';
export { resolveRequestLang } from './request_lang';
