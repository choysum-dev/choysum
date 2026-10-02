// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Maps List store limit/offset (+ optional controlled page/pageSize) onto
 * ChoyPagination's 1-based page models without mounting while total is still
 * unknown and a restored offset would clamp to page 1 too early.
 */

import { clampChoyPage, choyPageOffset, choyTotalPages } from './paginationHelpers';

export type ListPaginationProps = {
  total?: number;
  limit?: number;
  offset?: number;
  page?: number;
  pageSize?: number;
};

export type ListPaginationEffective = {
  total: number;
  limit: number;
  offset: number;
  currentPage: number;
  pageSize: number;
};

export function resolveListPaginationEffective(props: ListPaginationProps): ListPaginationEffective {
  const totalRaw = Number(props.total);
  const total = Number.isFinite(totalRaw) && totalRaw > 0 ? totalRaw : 0;
  const pageSizeControlled = Number(props.pageSize ?? 0) > 0 ? Number(props.pageSize) : undefined;
  const pageControlled = Number(props.page ?? 0) > 0 ? Number(props.page) : undefined;
  const limitCandidate = pageSizeControlled ?? Number(props.limit ?? 20);
  const limitRaw = Number.isFinite(limitCandidate) && limitCandidate > 0 ? limitCandidate : 20;
  const offsetRaw =
    pageControlled != null
      ? Math.max(0, (pageControlled - 1) * limitRaw)
      : Number.isFinite(Number(props.offset))
        ? Math.max(0, Number(props.offset))
        : 0;
  const currentPage = limitRaw > 0 ? Math.floor(offsetRaw / limitRaw) + 1 : 1;
  return { total, limit: limitRaw, offset: offsetRaw, currentPage, pageSize: limitRaw };
}

/** True when clamping to totalPages is safe (known total, or already on page 1). */
export function canMountListPagination(effective: ListPaginationEffective): boolean {
  return effective.total > 0 || effective.offset === 0;
}

export type ListPaginateState = { limit: number; offset: number };

export function listPageToPaginateState(
  page: number,
  effective: ListPaginationEffective,
): ListPaginateState {
  const size = effective.pageSize;
  const totalPages = choyTotalPages(effective.total, size);
  const validPage = clampChoyPage(page, totalPages);
  return { limit: size, offset: choyPageOffset(validPage, size) };
}

export function sameListPaginateState(a: ListPaginateState, b: ListPaginateState): boolean {
  return a.limit === b.limit && a.offset === b.offset;
}

export function listPageSizeToPaginateState(
  size: number,
  effective: ListPaginationEffective,
): ListPaginateState | null {
  const parsed = Number(size);
  const next = Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 0;
  if (next <= 0 || next === effective.pageSize) return null;
  return { limit: next, offset: 0 };
}
