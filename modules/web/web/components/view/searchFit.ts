// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Action-bar width at which the center search stays a field. Narrower bars use an icon. */
export const SEARCH_FIT_WIDE_PX = 640;

export function searchFieldFits(width: number): boolean {
  return width >= SEARCH_FIT_WIDE_PX;
}
