// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared provide/inject keys and types for ChoyViewScope.
 * String-literal keys keep legacy inject('view-mode') call sites matching.
 */
export type ViewMode = 'display' | 'edit' | 'create';
export type ViewContainer = 'Form' | 'List' | 'Kanban';

export const VIEW_MODE_KEY = 'view-mode';
export const VIEW_CONTAINER_KEY = 'view-container';
export const FIELD_PREFIX_KEY = 'field-prefix';
