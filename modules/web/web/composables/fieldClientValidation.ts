// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Client-side RuleItem evaluation for FieldBase after el-form-item removal.
 * FormView collects registered validators and blocks submit on the first failure.
 */

import type { RuleItem } from 'async-validator';
import type { InjectionKey } from 'vue';

export type FieldClientValidateFn = () => Promise<string>;

export const FIELD_CLIENT_VALIDATORS_KEY: InjectionKey<Map<string, FieldClientValidateFn>> =
  Symbol('field-client-validators');

/** Run async-validator-style rules; returns the first error message (or ''). */
export async function firstRuleError(rules: RuleItem[] | undefined | null, value: unknown): Promise<string> {
  const list = Array.isArray(rules) ? rules : [];
  for (const rule of list) {
    if (!rule || typeof rule !== 'object') continue;
    const required = (rule as { required?: boolean }).required === true;
    if (required && (value === undefined || value === null || value === '')) {
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
