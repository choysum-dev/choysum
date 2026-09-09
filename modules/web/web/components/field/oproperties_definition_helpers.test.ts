// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  buildDefinitionScopeCondition,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
} from './oproperties_definition_helpers';

describe('oproperties_definition_helpers', () => {
  test('covers drafts, coerce defaults, selection errors, and scope conditions', () => {
    expect(PROPERTY_DEFINITION_V1_TYPE_OPTIONS).toContain('char');
    expect(emptyDraftItem().type).toBe('char');
    expect(definitionItemsToDrafts(null)).toEqual([]);
    expect(definitionItemsToDrafts('x' as any)).toEqual([]);
    expect(definitionItemsToDrafts([null, 's', [], { name: '' }, { name: 'ok', type: 'char' }])).toEqual([
      { name: 'ok', type: 'char', string: '', default: '', readonly: false, selectionText: '' },
    ]);
    expect(definitionItemsToDrafts([{ name: 'notyped' }])[0]?.type).toBe('char');
    expect(definitionItemsToDrafts([{ name: 'emptytype', type: '' }])[0]?.type).toBe('char');

    const circ: any[] = [];
    circ.push(circ);
    expect(definitionItemsToDrafts([{ name: 'c', type: 'selection', selection: circ }])[0]?.selectionText).toBe('');

    const drafts = definitionItemsToDrafts([
      { name: 'code', type: 'char', string: 'Code', default: 'A', readonly: true },
      { name: 'kind', type: 'selection', selection: [['a', 'A']] },
      { name: 'weird', type: 'not-a-real-type' },
      { name: 'emptyDefaults', type: 'char', string: null, default: null },
      { name: '', type: 'char' },
    ]);
    expect(drafts).toHaveLength(4);
    expect(drafts[1]?.selectionText).toContain('a');
    expect(drafts[2]?.type).toBe('not-a-real-type');

    expect(draftsToDefinitionItems(null)).toEqual([]);
    expect(draftsToDefinitionItems(undefined)).toEqual([]);
    expect(
      draftsToDefinitionItems([
        { ...emptyDraftItem(), name: 'b', type: 'boolean', default: 'true' },
        { ...emptyDraftItem(), name: 'b0', type: 'boolean', default: '0' },
        { ...emptyDraftItem(), name: 'b1', type: 'boolean', default: '1' },
        { ...emptyDraftItem(), name: 'bf', type: 'boolean', default: 'false' },
        { ...emptyDraftItem(), name: 'bx', type: 'boolean', default: 'maybe' },
        { ...emptyDraftItem(), name: 'i', type: 'integer', default: '3.9' },
        { ...emptyDraftItem(), name: 'ibad', type: 'integer', default: 'nope' },
        { ...emptyDraftItem(), name: 'f', type: 'float', default: '1.5' },
        { ...emptyDraftItem(), name: 'fbad', type: 'float', default: 'nope' },
        { ...emptyDraftItem(), name: 'c', type: 'char', default: 'x', string: 'L', readonly: true },
        {
          ...emptyDraftItem(),
          name: 'sel',
          type: 'selection',
          selectionText: '[["a","A"]]',
        },
        { ...emptyDraftItem(), name: '', type: 'char' },
        { ...emptyDraftItem(), name: 'ndef', type: 'char', default: null as any },
        { ...emptyDraftItem(), name: 'udef', type: 'char', default: undefined as any },
      ])
    ).toEqual([
      { name: 'b', type: 'boolean', default: true },
      { name: 'b0', type: 'boolean', default: false },
      { name: 'b1', type: 'boolean', default: true },
      { name: 'bf', type: 'boolean', default: false },
      { name: 'bx', type: 'boolean', default: 'maybe' },
      { name: 'i', type: 'integer', default: 3 },
      { name: 'ibad', type: 'integer', default: 'nope' },
      { name: 'f', type: 'float', default: 1.5 },
      { name: 'fbad', type: 'float', default: 'nope' },
      { name: 'c', type: 'char', string: 'L', default: 'x', readonly: true },
      { name: 'sel', type: 'selection', selection: [['a', 'A']] },
      { name: 'ndef', type: 'char' },
      { name: 'udef', type: 'char' },
    ]);

    expect(() => draftsToDefinitionItems([{ ...emptyDraftItem(), name: 'x', type: 'html' }])).toThrow(
      /unsupported property type/
    );
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '' }])
    ).toThrow(/requires selection JSON/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '{' }])
    ).toThrow(/invalid JSON/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '{}' }])
    ).toThrow(/non-empty selection array/);
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '[]' }])
    ).toThrow(/non-empty selection array/);

    expect(
      buildDefinitionScopeCondition({
        targetModel: 'Partner',
        propertiesField: 'PartnerProperties',
      })
    ).toEqual([
      ['TargetModel', '=', 'Partner'],
      ['PropertiesField', '=', 'PartnerProperties'],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'Task',
        propertiesField: 'TaskProperties',
        containerModel: 'Project',
        containerId: 'p1',
      })
    ).toEqual([
      ['TargetModel', '=', 'Task'],
      ['PropertiesField', '=', 'TaskProperties'],
      ['ContainerModel', '=', 'Project'],
      ['ContainerId', '=', 'p1'],
    ]);
    expect(
      buildDefinitionScopeCondition({
        targetModel: 'Task',
        propertiesField: 'TaskProperties',
        containerModel: '',
        containerId: '',
      })
    ).toEqual([
      ['TargetModel', '=', 'Task'],
      ['PropertiesField', '=', 'TaskProperties'],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
    expect(
      buildDefinitionScopeCondition({
        targetModel: '',
        propertiesField: null as any,
        containerModel: null,
        containerId: null,
      })
    ).toEqual([
      ['TargetModel', '=', ''],
      ['PropertiesField', '=', ''],
      ['ContainerModel', '=', null],
      ['ContainerId', '=', null],
    ]);
  });
});
