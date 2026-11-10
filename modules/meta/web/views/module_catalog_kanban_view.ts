// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { unwrapListRecord } from '@/web/web/composables/listRowEdit';
import type { ChoyKanbanCard } from '@/web';
import {
  formatModuleCatalogVersionLine,
  moduleCatalogCardApplication,
  moduleCatalogCardAuthor,
  moduleCatalogCardLicense,
  moduleCatalogCardSummary,
} from './module_catalog_chrome';

export function moduleCatalogKanbanRowRecord(row: unknown): Record<string, unknown> {
  const rec = unwrapListRecord(row);
  return rec && typeof rec === 'object' ? (rec as Record<string, unknown>) : {};
}

export function moduleCatalogKanbanRowToCard(
  row: unknown,
  index: number,
  laneKey: string,
): ChoyKanbanCard {
  const payload = moduleCatalogKanbanRowRecord(row);
  const id = String(payload.Id ?? payload.ModuleName ?? `${laneKey}-${index}`);
  return {
    id,
    title: String(payload.ModuleName || id),
    laneKey,
    payload,
  };
}

export function moduleCatalogKanbanResolveMoveRecordId(
  cards: ReadonlyArray<ChoyKanbanCard>,
  moveCardId: string,
): string {
  const hit = cards.find(c => String(c.id) === String(moveCardId));
  const payload = (hit?.payload || {}) as Record<string, unknown>;
  return String(payload.Id ?? moveCardId ?? '').trim();
}

export function moduleCatalogKanbanCardPayload(card: ChoyKanbanCard): Record<string, unknown> {
  const p = card.payload;
  return p && typeof p === 'object' ? (p as Record<string, unknown>) : {};
}

export function moduleCatalogKanbanCardModuleName(card: ChoyKanbanCard): string {
  return String(moduleCatalogKanbanCardPayload(card).ModuleName || card.title || '').trim();
}

export function moduleCatalogKanbanCardSummary(card: ChoyKanbanCard): string {
  return moduleCatalogCardSummary(moduleCatalogKanbanCardPayload(card));
}

export function moduleCatalogKanbanCardApplication(card: ChoyKanbanCard): string {
  return moduleCatalogCardApplication(moduleCatalogKanbanCardPayload(card));
}

export function moduleCatalogKanbanCardAuthor(card: ChoyKanbanCard): string {
  return moduleCatalogCardAuthor(moduleCatalogKanbanCardPayload(card));
}

export function moduleCatalogKanbanCardLicense(card: ChoyKanbanCard): string {
  return moduleCatalogCardLicense(moduleCatalogKanbanCardPayload(card));
}

export function moduleCatalogKanbanCardVersionLine(card: ChoyKanbanCard): string {
  return formatModuleCatalogVersionLine(moduleCatalogKanbanCardPayload(card));
}

export type ModuleCatalogKanbanErrorMessages = {
  loadFailed: string;
  searchFailed: string;
  loadMoreFailed: string;
};

export function createModuleCatalogKanbanErrorHandlers(
  msgs: ModuleCatalogKanbanErrorMessages,
  reportError: (message: string) => void,
) {
  return {
    onLoadError(e: unknown) {
      reportError(msgs.loadFailed);
      console.error('Module catalog kanban load failed:', e);
    },
    onSearchError(e: unknown) {
      reportError(msgs.searchFailed);
      console.error('Module catalog kanban search failed:', e);
    },
    onLoadMoreError(e: unknown) {
      reportError(msgs.loadMoreFailed);
      console.error('Module catalog kanban load-more failed:', e);
    },
  };
}
