// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildDefinitionScopeCondition,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
} from './propertiesDefinitionHelpers';

describe('propertiesDefinitionHelpers', () => {
  test('emptyDraftItem defaults to char', () => {
    expect(emptyDraftItem().type).toBe('char');
  });

  test('definitionItemsToDrafts / draftsToDefinitionItems round-trip', () => {
    const drafts = definitionItemsToDrafts([
      { name: 'tier', type: 'selection', string: 'Tier', selection: [['a', 'A'], ['b', 'B']] },
      { name: 'flag', type: 'boolean', default: true, readonly: true },
    ]);
    expect(drafts).toHaveLength(2);
    const items = draftsToDefinitionItems(drafts);
    expect(items[0]!.name).toBe('tier');
    expect(items[0]!.selection).toEqual([['a', 'A'], ['b', 'B']]);
    expect(items[1]!.readonly).toBe(true);
    expect(items[1]!.default).toBe(true);
  });

  test('draftsToDefinitionItems rejects bad selection', () => {
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '' }]),
    ).toThrow(/requires selection JSON/);
  });

  test('buildDefinitionScopeCondition', () => {
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'base.Company',
        propertiesField: 'Props',
        containerModel: null,
        containerId: null,
      }),
    ).toEqual([
      ['TargetModel', '=', 'base.Company'],
      ['PropertiesField', '=', 'Props'],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
  });
});
