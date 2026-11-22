// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Navigation session for Form record pager (List/Kanban → Form).
 * Snapshotted at open-detail time; not the live List ModelStore result.
 */
export type RecordSetEntry = {
  /** Stable key, e.g. `${model}|${listRouteName}`. */
  key: string;
  /** Model technical name (e.g. `base.Country`). */
  model: string;
  /** Merged Search condition at seed time. */
  condition: unknown;
  /** FE orderBy rows (`direction`) at seed time. */
  orderBy?: unknown;
  /** Collection size (`result.total`) at seed time. */
  total: number;
  /** 0-based global index of the current record. */
  index: number;
  /** Sliding window of Ids (at least the page at seed). */
  ids: string[];
  /** Global index of `ids[0]`. */
  windowStart: number;
};

/** Fetches a window of Ids for cross-page neighbor resolution. */
export type SearchRecordSetIds = (args: {
  condition: unknown;
  orderBy?: unknown;
  offset: number;
  limit: number;
}) => Promise<string[]>;
