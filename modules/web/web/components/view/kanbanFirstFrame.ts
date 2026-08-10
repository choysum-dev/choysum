// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * OSearchView emits a mount-time query-update (including UserFilter defaults).
 * When that component is the searchView, Kanban should wait instead of applying on mount.
 */
export function shouldDeferKanbanFirstFrame(searchView: unknown, oSearchView: unknown): boolean {
  return searchView === oSearchView;
}
