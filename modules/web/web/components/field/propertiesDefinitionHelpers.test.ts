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

  test('definitionItemsToDrafts skips non-items and stringifies selection', () => {
    expect(definitionItemsToDrafts(null)).toEqual([]);
    expect(definitionItemsToDrafts('x' as unknown as unknown[])).toEqual([]);
    expect(
      definitionItemsToDrafts([
        null,
        [],
        { name: '' },
        { name: 'ok', type: 'char', string: null, default: null },
        {
          name: 'sel',
          type: 'selection',
          selection: [
            ['a', 'A'],
            { value: 'b', label: 'B' },
          ],
        },
      ]),
    ).toEqual([
      {
        name: 'ok',
        type: 'char',
        string: '',
        default: '',
        readonly: false,
        selectionText: '',
      },
      {
        name: 'sel',
        type: 'selection',
        string: '',
        default: '',
        readonly: false,
        selectionText: JSON.stringify([
          ['a', 'A'],
          { value: 'b', label: 'B' },
        ]),
      },
    ]);
  });

  test('definitionItemsToDrafts tolerates selection stringify failure', () => {
    const selection = [
      {
        toJSON(): never {
          throw new Error('boom');
        },
      },
    ];
    const drafts = definitionItemsToDrafts([{ name: 'c', type: 'selection', selection }]);
    expect(drafts).toHaveLength(1);
    expect(drafts[0]!.selectionText).toBe('');
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

  test('draftsToDefinitionItems persists only normalized selection options', () => {
    const items = draftsToDefinitionItems([
      {
        ...emptyDraftItem(),
        name: 'tier',
        type: 'selection',
        selectionText: JSON.stringify([['a', 'A'], 1, { value: 'b', label: 'B' }, { value: 2 }]),
      },
    ]);
    expect(items[0]!.selection).toEqual([
      ['a', 'A'],
      ['b', 'B'],
    ]);
  });

  test('draftsToDefinitionItems coerces defaults and skips blank names', () => {
    const items = draftsToDefinitionItems([
      { ...emptyDraftItem(), name: '  ' },
      { ...emptyDraftItem(), name: 'flag', type: 'boolean', default: 'false' },
      { ...emptyDraftItem(), name: 'flag2', type: 'boolean', default: '0' },
      { ...emptyDraftItem(), name: 'flag3', type: 'boolean', default: '1' },
      { ...emptyDraftItem(), name: 'flag4', type: 'boolean', default: 'maybe' },
      { ...emptyDraftItem(), name: 'n', type: 'integer', default: '3.9' },
      { ...emptyDraftItem(), name: 'n2', type: 'integer', default: 'nope' },
      { ...emptyDraftItem(), name: 'f', type: 'float', default: '1.5' },
      { ...emptyDraftItem(), name: 'f2', type: 'float', default: 'nope' },
      { ...emptyDraftItem(), name: 'c', type: 'char', default: 'x', string: ' Label ' },
      { ...emptyDraftItem(), name: 'ws', type: 'integer', default: '   ' },
    ]);
    expect(items.find(i => i.name === 'flag')!.default).toBe(false);
    expect(items.find(i => i.name === 'flag2')!.default).toBe(false);
    expect(items.find(i => i.name === 'flag3')!.default).toBe(true);
    expect(items.find(i => i.name === 'flag4')!.default).toBe('maybe');
    expect(items.find(i => i.name === 'n')!.default).toBe(3);
    expect(items.find(i => i.name === 'n2')!.default).toBe('nope');
    expect(items.find(i => i.name === 'f')!.default).toBe(1.5);
    expect(items.find(i => i.name === 'f2')!.default).toBe('nope');
    expect(items.find(i => i.name === 'c')!.string).toBe('Label');
    expect(Object.prototype.hasOwnProperty.call(items.find(i => i.name === 'ws')!, 'default')).toBe(
      false,
    );
  });

  test('draftsToDefinitionItems rejects bad selection and unsupported types', () => {
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '' }]),
    ).toThrow(/requires selection JSON/);
    expect(() =>
      draftsToDefinitionItems([
        { ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '{' },
      ]),
    ).toThrow(/invalid JSON/);
    expect(() =>
      draftsToDefinitionItems([
        { ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '[]' },
      ]),
    ).toThrow(/non-empty selection array/);
    expect(() =>
      draftsToDefinitionItems([
        { ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '[1]' },
      ]),
    ).toThrow(/no usable options/);
    expect(() =>
      draftsToDefinitionItems([
        {
          ...emptyDraftItem(),
          name: 's',
          type: 'selection',
          default: 'missing',
          selectionText: JSON.stringify([['a', 'A']]),
        },
      ]),
    ).toThrow(/default is not one of the options/);
    expect(
      draftsToDefinitionItems([
        {
          ...emptyDraftItem(),
          name: 's',
          type: 'selection',
          default: 'a',
          selectionText: JSON.stringify([['a', 'A']]),
        },
      ])[0]!.default,
    ).toBe('a');
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 'x', type: 'binary' }]),
    ).toThrow(/unsupported property type/);
    expect(() =>
      draftsToDefinitionItems([
        { ...emptyDraftItem(), name: 'dup', type: 'char' },
        { ...emptyDraftItem(), name: 'dup', type: 'integer' },
      ]),
    ).toThrow(/duplicate property name/);
    expect(draftsToDefinitionItems(null)).toEqual([]);
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
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'base.Company',
        propertiesField: 'Props',
        containerModel: 'base.Partner',
        containerId: 'p1',
      }),
    ).toEqual([
      ['TargetModel', '=', 'base.Company'],
      ['PropertiesField', '=', 'Props'],
      ['ContainerModel', '=', 'base.Partner'],
      ['ContainerId', '=', 'p1'],
    ]);
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'base.Company',
        propertiesField: 'Props',
        containerModel: '',
        containerId: '',
      }),
    ).toEqual([
      ['TargetModel', '=', 'base.Company'],
      ['PropertiesField', '=', 'Props'],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
  });
});
