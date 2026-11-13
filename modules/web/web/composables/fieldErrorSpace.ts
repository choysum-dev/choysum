// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * How field errors occupy vertical space under the control.
 * Host views (Form/List/Settings) may provide a default via FIELD_ERROR_SPACE_KEY;
 * field props override the inject; unset falls back to collapse.
 */

import type { InjectionKey } from 'vue';

export type ChoyInlineErrorSpace = 'collapse' | 'reserve' | 'overlay';

export const FIELD_ERROR_SPACE_KEY: InjectionKey<ChoyInlineErrorSpace> = Symbol(
  'choy-field-error-space',
);

/** Resolve effective space: explicit prop wins, then inject, then collapse. */
export function resolveFieldErrorSpace(
  prop: ChoyInlineErrorSpace | undefined | null,
  injected: ChoyInlineErrorSpace | null | undefined,
): ChoyInlineErrorSpace {
  if (prop === 'reserve' || prop === 'overlay' || prop === 'collapse') return prop;
  if (injected === 'reserve' || injected === 'overlay' || injected === 'collapse') return injected;
  return 'collapse';
}
