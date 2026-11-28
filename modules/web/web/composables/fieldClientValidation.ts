// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Client-side RuleItem evaluation for ChoyFieldBase.
 * FormView collects registered validators and blocks submit on the first failure.
 */

import type { RuleItem } from 'async-validator';
import type { InjectionKey } from 'vue';

export type FieldClientValidateFn = () => Promise<string>;

export const FIELD_CLIENT_VALIDATORS_KEY: InjectionKey<Map<string, FieldClientValidateFn>> =
  Symbol('field-client-validators');

/**
 * Merge conceptual required (meta.notNull / :required) with explicit field rules.
 * When required and rules do not already include required:true, prepend a default
 * required rule so asterisk and submit/blur validation stay aligned.
 */
export function isEmptyFieldValue(value: unknown): boolean {
  return (
    value === undefined ||
    value === null ||
    value === '' ||
    (Array.isArray(value) && value.length === 0)
  );
}

export function rulesHaveExplicitRequired(rules?: RuleItem[] | null): boolean {
  const list = Array.isArray(rules) ? rules : [];
  return list.some((rule) => rule && typeof rule === 'object' && (rule as { required?: boolean }).required === true);
}

/**
 * Auto-required (meta.notNull / :required without an explicit rules required)
 * fails validation but should not paint inline/overlay copy — toast + invalid chrome only.
 */
export function isChromeOnlyRequiredFailure(opts: {
  required: boolean;
  rules?: RuleItem[] | null;
  value: unknown;
}): boolean {
  return opts.required && isEmptyFieldValue(opts.value) && !rulesHaveExplicitRequired(opts.rules);
}

export function resolveEffectiveClientRules(opts: {
  required: boolean;
  rules?: RuleItem[] | null;
  requiredMessage: string;
}): RuleItem[] {
  const list = Array.isArray(opts.rules) ? opts.rules.filter((r) => r && typeof r === 'object') : [];
  if (!opts.required || rulesHaveExplicitRequired(list)) {
    return list.slice();
  }
  const message = String(opts.requiredMessage || '').trim() || 'Required';
  return [{ required: true, message }, ...list];
}

/** Run async-validator-style rules; returns the first error message (or ''). */
export async function firstRuleError(rules: RuleItem[] | undefined | null, value: unknown): Promise<string> {
  const list = Array.isArray(rules) ? rules : [];
  for (const rule of list) {
    if (!rule || typeof rule !== 'object') continue;
    const required = (rule as { required?: boolean }).required === true;
    const isEmpty = isEmptyFieldValue(value);
    if (required && isEmpty) {
      const msg = String((rule as { message?: string }).message || '').trim();
      return msg || 'Required';
    }
    const validator = (rule as { validator?: unknown }).validator;
    if (typeof validator !== 'function') continue;
    const message = await new Promise<string | undefined>(resolve => {
      let settled = false;
      const done = (error?: Error | string | null) => {
        if (settled) return;
        settled = true;
        if (!error) {
          resolve(undefined);
          return;
        }
        if (typeof error === 'string') {
          resolve(error);
          return;
        }
        resolve(error.message || String(error));
      };
      try {
        const ret = (validator as (r: unknown, v: unknown, cb: (e?: Error | string) => void) => unknown)(
          rule,
          value,
          done
        );
        if (ret && typeof (ret as Promise<unknown>).then === 'function') {
          (ret as Promise<unknown>)
            .then(() => done())
            .catch((e: unknown) => {
              done(e instanceof Error ? e : new Error(String(e)));
            });
        }
      } catch (e) {
        done(e instanceof Error ? e : new Error(String(e)));
      }
    });
    if (message) return message;
  }
  return '';
}
