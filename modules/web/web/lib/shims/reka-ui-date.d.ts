// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Type shim for the `reka-ui/date` subpath (Calendar year helpers).
 * Runtime resolves via esm; type-fetch does not yet map this subpath into
 * modules/tsconfig.json paths.
 */
declare module 'reka-ui/date' {
  import type { DateValue } from 'reka-ui';

  export function createYear(props: { dateObj: DateValue }): DateValue[];
  export function createYearRange(props: {
    start?: DateValue;
    end?: DateValue;
    years?: number;
  }): DateValue[];
  export function toDate(dateValue: DateValue, timeZone?: string): Date;
}
