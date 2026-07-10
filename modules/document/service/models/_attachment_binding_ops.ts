// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type {
  BindReq, BindResp, UnbindReq, UnbindResp,
  BatchDescribeReq, BatchDescribeResp,
  ResolveDownloadContentReq, ResolveDownloadContentResp,
} from '../contracts';

export type BindingModelOps = {
  // BaseModel context
  readonly userId: unknown;
  readonly companyId: unknown;
  readonly companyIds: string[];
  // BaseModel methods
  Browse(condition: unknown, options?: unknown): Promise<unknown[]>;
  Search(condition: unknown, options?: unknown): Promise<unknown[]>;
  Create(values: unknown, fields?: unknown): Promise<unknown>;
  Update(id: string, values: unknown, fields?: unknown): Promise<unknown>;
  // Private helper access (passthrough to model class)
  [key: string]: unknown;
};
