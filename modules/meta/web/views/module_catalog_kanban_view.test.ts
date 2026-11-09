// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createModuleCatalogKanbanErrorHandlers,
  moduleCatalogKanbanCardAuthor,
  moduleCatalogKanbanCardLicense,
  moduleCatalogKanbanCardPayload,
  moduleCatalogKanbanResolveMoveRecordId,
  moduleCatalogKanbanRowRecord,
  moduleCatalogKanbanRowToCard,
} from './module_catalog_kanban_view';

test('moduleCatalogKanbanRowToCard and resolveMoveRecordId', () => {
  const row = {
    Id: 'id-1',
    ModuleName: 'partner',
    Author: 'Acme',
    License: 'MIT',
  };
  const card = moduleCatalogKanbanRowToCard(row, 0, 'all');
  expect(card.id).toBe('id-1');
  expect(card.title).toBe('partner');
  expect(moduleCatalogKanbanRowRecord({ payload: row }).ModuleName).toBe('partner');

  const fallback = moduleCatalogKanbanRowToCard({}, 2, 'lane');
  expect(String(fallback.id)).toContain('lane-2');

  expect(moduleCatalogKanbanResolveMoveRecordId([card], card.id)).toBe('id-1');
  expect(moduleCatalogKanbanResolveMoveRecordId([], 'orphan')).toBe('orphan');
});

test('moduleCatalogKanban card chrome helpers read payload fields', () => {
  const card = {
    id: 'c1',
    title: 't',
    laneKey: 'all',
    payload: {
      ModuleName: 'meta',
      Author: 'Team',
      License: 'Apache-2.0',
      Summary: 'Meta registry tooling',
      Application: 'meta',
    },
  };
  expect(moduleCatalogKanbanCardAuthor(card)).toBe('Team');
  expect(moduleCatalogKanbanCardLicense(card)).toBe('Apache-2.0');
  expect(moduleCatalogKanbanCardPayload({ id: 'x', title: 'y', laneKey: 'z', payload: null as any })).toEqual(
    {},
  );
});

test('createModuleCatalogKanbanErrorHandlers reports localized messages', () => {
  const reported: string[] = [];
  const handlers = createModuleCatalogKanbanErrorHandlers(
    {
      loadFailed: 'load-fail',
      searchFailed: 'search-fail',
      loadMoreFailed: 'more-fail',
    },
    msg => reported.push(msg),
  );
  handlers.onLoadError(new Error('x'));
  handlers.onSearchError(new Error('y'));
  handlers.onLoadMoreError(new Error('z'));
  expect(reported).toEqual(['load-fail', 'search-fail', 'more-fail']);
});
