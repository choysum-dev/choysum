// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared dayjs + utc + customParseFormat for calendar/display formatting.
 * Peer dayjs typings omit plugin methods; cast once here instead of per call site.
 */

import dayjs, { type ConfigType, type Dayjs } from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import utc from 'dayjs/plugin/utc';

dayjs.extend(utc);
dayjs.extend(customParseFormat);

type UtcDayjsFactory = {
  (date?: ConfigType, format?: string, strict?: boolean): Dayjs;
  utc: (date?: ConfigType, format?: string, strict?: boolean) => Dayjs;
  extend: typeof dayjs.extend;
};

/** dayjs with `.utc(...)` for wall-clock / calendar display formatting. */
export const dayjsUtc = dayjs as unknown as UtcDayjsFactory;
