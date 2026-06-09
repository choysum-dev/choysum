// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Compilable, SimplifyResult } from './common';

type RepositoryStubQueryLike = {
  kind: string;
  [key: string]: unknown;
};

export type RepositoryQueryLike<TRow = unknown> = Compilable<TRow> | RepositoryStubQueryLike;

export type RepositoryExecute = {
  bivarianceHack<TRow = unknown>(query: RepositoryQueryLike<TRow>): Promise<SimplifyResult<TRow>[]>;
}['bivarianceHack'];
