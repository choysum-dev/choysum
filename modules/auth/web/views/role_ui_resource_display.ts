// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type UiResourceRowLabel = {
  Title?: string;
  Name?: string;
  Id?: string;
};

export type UiResourceIconKind = 'menu' | 'route' | 'action' | 'other';

/**
 * Plain-string fallback used before translateTerm resolves TitleText.
 */
export function uiResourceLabelFallback(row?: UiResourceRowLabel | null, label?: string): string {
  return String(label || row?.Title || row?.Name || row?.Id || '');
}

/**
 * Map MetaUiResource.Type to a stable icon kind (icons stay in the Vue SFC).
 */
export function uiResourceTypeIconKind(type?: string): UiResourceIconKind {
  switch (type) {
    case 'MENU':
      return 'menu';
    case 'ROUTE':
      return 'route';
    case 'ACTION':
      return 'action';
    default:
      return 'other';
  }
}
