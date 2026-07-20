// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getContextLang } from '../runtime/context/scope';
import { resolveI18nScope, withI18nScope, type ResolveI18nScopeOptions } from './scope';
import { resolveRequestLang } from './request_lang';

export type TranslateOptions = ResolveI18nScopeOptions & {
  kind?: string;
};
export type CreateTranslateOptions = TranslateOptions;
type TermOptions = TranslateOptions;

export type TermIdentity = {
  module: string;
  scope: string;
  src: string;
  kind: string;
};

export type TermReference = {
  key: string;
  module: string;
  scope: string;
  src: string;
  kind: 'literal';
};

export const TERM_REFERENCE_NAMESPACE = '__terms';

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
export function createTermReferenceKey(
  module: string,
  scope: string,
  src: string,
  kind = 'literal'
): string {
  const values = [module, scope, src, kind].map(value => String(value));
  const identity = values
    .map(value => `${new TextEncoder().encode(value).length}:${value}`)
    .join('');
  return `${TERM_REFERENCE_NAMESPACE}.${utf8Hex(identity)}`;
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

/**
 * Immediate text translation helper.
 */
export type TranslateFn = {
  (src: string, opts: TermOptions, ...args: unknown[]): string;
  (src: string, arg1: string | number | boolean | null | undefined, ...args: unknown[]): string;
  (src: string): string;
};

/**
 * Pin a serializable term reference (no lookup, no interpolation).
 */
export type LazyTranslateFn = {
  (src: string, opts?: TermOptions): TermReference;
};

export function isTermReference(value: unknown): value is TermReference {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return false;
  }
  const reference = value as Partial<TermReference>;
  return (
    typeof reference.module === 'string' &&
    typeof reference.scope === 'string' &&
    typeof reference.src === 'string' &&
    typeof reference.key === 'string' &&
    reference.key === createTermReferenceKey(reference.module, reference.scope, reference.src, reference.kind) &&
    reference.kind === 'literal'
  );
}

export function createTermReference(module: string, src: string, opts?: TranslateOptions): TermReference {
  const identity = {
    ...createTermIdentity(module, src, opts),
    kind: 'literal' as const,
  };
  return {
    ...identity,
    key: createTermReferenceKey(
      identity.module,
      identity.scope,
      identity.src,
      identity.kind
    ),
  };
}

export type CreateTranslateResult = {
  _t: TranslateFn;
  _lt: LazyTranslateFn;
};

/**
 * Bind terminology helpers `_t` (text) and `_lt` (TermReference) to an owner module.
 *
 * `_t` translates immediately. `_lt` performs no lookup and returns deterministic,
 * serializable metadata for static wire (menu / route / field titles).
 */
export function createTranslate(
  module: string,
  defaults?: CreateTranslateOptions
): CreateTranslateResult {
  const mod = String(module || '').trim();
  const defaultScope = resolveI18nScope(defaults);
  const defaultKind = defaults?.kind;
  const defaultIdentities = new Map<string, TermIdentity>();
  const defaultReferences = new Map<string, TermReference>();

  const resolveOptions = (opts?: TranslateOptions): TranslateOptions => {
    const scope = resolveI18nScope(opts) || defaultScope;
    return {
      ...opts,
      scope,
      kind: opts?.kind ?? defaultKind,
    };
  };

  const _t = (src: string, ...args: unknown[]): string => {
    let opts: TranslateOptions | undefined;
    let interp: unknown[] = args;
    if (args.length > 0 && isTranslateOptions(args[0])) {
      opts = args[0] as TranslateOptions;
      interp = args.slice(1);
    }
    let identity: TermIdentity;
    if (!opts && defaultScope) {
      identity = defaultIdentities.get(src) ??
        createTermIdentity(mod, src, resolveOptions());
      defaultIdentities.set(src, identity);
    } else {
      identity = createTermIdentity(mod, src, resolveOptions(opts));
    }
    const hit = lookup(identity);
    const base = hit || identity.src;
    return interpolate(base, interp);
  };

  const _lt = (src: string, opts?: TermOptions, ...rest: unknown[]): TermReference => {
    if (rest.length > 0) {
      throw new Error('_lt does not accept interpolation arguments');
    }
    if (!opts && defaultScope) {
      const cached = defaultReferences.get(src);
      if (cached) {
        return cached;
      }
    }
    const reference = createTermReference(mod, src, resolveOptions(opts));
    if (!opts && defaultScope) {
      defaultReferences.set(src, reference);
    }
    return reference;
  };

  return {
    _t: _t as TranslateFn,
    _lt: _lt as LazyTranslateFn,
  };
}

export { withI18nScope, resolveI18nScope, formatScope } from './scope';
export { resolveRequestLang } from './request_lang';
