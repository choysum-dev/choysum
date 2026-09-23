// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoyTabRegistration } from './choyTabsContext';

/**
 * Picks a tab value: prefer an enabled defaultValue, else first enabled tab.
 * Returns '' when nothing is selectable (e.g. all disabled).
 */
export function pickChoyTabSelection(
  list: ChoyTabRegistration[],
  defaultValue?: string,
): string {
  if (defaultValue && list.some((item) => item.value === defaultValue && !item.disabled)) {
    return defaultValue;
  }
  return list.find((item) => !item.disabled)?.value ?? '';
}

/**
 * Decides the next model value when the registration list changes.
 * Returns undefined when the caller should leave the current value unchanged
 * (still valid, or waiting for a late-registering defaultValue).
 */
export function nextChoyTabSelection(
  list: ChoyTabRegistration[],
  current: string | undefined,
  defaultValue?: string,
): string | undefined {
  if (!list.length) {
    return undefined;
  }
  const stillValid =
    !!current && list.some((item) => item.value === current && !item.disabled);
  if (stillValid) {
    return undefined;
  }
  // Wait for the configured default to register across mount ticks.
  if (defaultValue && !list.some((item) => item.value === defaultValue)) {
    return undefined;
  }
  return pickChoyTabSelection(list, defaultValue);
}
