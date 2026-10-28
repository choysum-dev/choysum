// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  PROPERTY_DEFINITION_V1_TYPE_OPTIONS,
  applyDraftCurrentLangToMaps,
  buildDefinitionScopeCondition,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
  selectionRawLabel,
  selectionRowsFromDraft,
  syncDraftLabelsForLang,
  writeSelectionRowsToDraft,
} from './propertiesDefinitionHelpers';

describe('propertiesDefinitionHelpers', () => {
  test('emptyDraftItem defaults to char', () => {
    expect(PROPERTY_DEFINITION_V1_TYPE_OPTIONS).toContain('char');
    expect(emptyDraftItem().type).toBe('char');
    expect(emptyDraftItem().stringMap).toEqual({});
    expect(emptyDraftItem().help).toBe('');
    expect(emptyDraftItem().helpMap).toEqual({});
  });

  test('selectionRawLabel covers tuple, object, and scalar fallback', () => {
    expect(selectionRawLabel(['a', 'A'])).toBe('A');
    expect(selectionRawLabel(['a'])).toBe('a');
    expect(selectionRawLabel({ value: 'a', label: 'A' })).toBe('A');
    expect(selectionRawLabel({ value: 'a' })).toBe('a');
    expect(selectionRawLabel('scalar')).toBeUndefined();
    expect(selectionRawLabel(null)).toBeUndefined();
  });

  test('definitionItemsToDrafts unwraps lang maps for the requested lang', () => {
    expect(definitionItemsToDrafts(null)).toEqual([]);
    expect(definitionItemsToDrafts('x' as unknown as unknown[])).toEqual([]);
    const drafts = definitionItemsToDrafts(
      [
        null,
        [],
        { name: '' },
        { name: 'ok', type: 'char', string: null, default: null },
        {
          name: 'sel',
          type: 'selection',
          string: { en_US: 'Priority', zh_CN: '优先级' },
          help: { en_US: 'Record urgency', zh_CN: '记录紧急程度' },
          selection: [
            ['a', { en_US: 'A', zh_CN: '甲' }],
            { value: 'b', label: 'B' },
          ],
        },
      ],
      'zh_CN',
    );
    expect(drafts).toEqual([
      {
        name: 'ok',
        type: 'char',
        string: '',
        stringMap: {},
        help: '',
        helpMap: {},
        default: '',
        readonly: false,
        selectionText: '',
        selectionLabelMaps: {},
        selectionRowMapKeys: [],
      },
      {
        name: 'sel',
        type: 'selection',
        string: '优先级',
        stringMap: { en_US: 'Priority', zh_CN: '优先级' },
        help: '记录紧急程度',
        helpMap: { en_US: 'Record urgency', zh_CN: '记录紧急程度' },
        default: '',
        readonly: false,
        selectionText: JSON.stringify([
          ['a', '甲'],
          ['b', 'B'],
        ]),
        selectionLabelMaps: {
          a: { en_US: 'A', zh_CN: '甲' },
          b: { en_US: 'B' },
        },
        selectionRowMapKeys: ['a', 'b'],
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

    const circ: any[] = [];
    circ.push(circ);
    expect(definitionItemsToDrafts([{ name: 'c2', type: 'selection', selection: circ }])[0]?.selectionText).toBe(
      '',
    );
    expect(definitionItemsToDrafts([{ name: 'notyped' }])[0]?.type).toBe('char');
    expect(definitionItemsToDrafts([{ name: 'emptytype', type: '' }])[0]?.type).toBe('char');
  });

  test('draftsToDefinitionItems merges current lang without wiping other langs', () => {
    const previous = [
      {
        name: 'tier',
        type: 'selection',
        string: { en_US: 'Tier', zh_CN: '等级' },
        selection: [
          ['a', { en_US: 'A', zh_CN: '甲' }],
          ['b', { en_US: 'B', zh_CN: '乙' }],
        ],
      },
    ];
    const drafts = definitionItemsToDrafts(previous, 'zh_CN');
    drafts[0]!.string = '级别';
    drafts[0]!.selectionText = JSON.stringify([
      ['a', '甲级'],
      ['b', '乙'],
    ]);
    const items = draftsToDefinitionItems(drafts, 'zh_CN', previous as any);
    expect(items[0]!.string).toEqual({ en_US: 'Tier', zh_CN: '级别' });
    expect(items[0]!.selection).toEqual([
      ['a', { en_US: 'A', zh_CN: '甲级' }],
      ['b', { en_US: 'B', zh_CN: '乙' }],
    ]);
  });

  test('draftsToDefinitionItems keeps translate-dialog maps even when previous exists', () => {
    const previous = [
      {
        name: 'tax_id',
        type: 'char',
        string: { en_US: 'Tax', zh_CN: '税号' },
        help: { en_US: 'Help', zh_CN: '帮助' },
      },
    ];
    const drafts = definitionItemsToDrafts(previous, 'en_US');
    drafts[0]!.stringMap = { en_US: 'Tax', zh_CN: '税号', ja_JP: '税' };
    drafts[0]!.helpMap = { en_US: 'Help', zh_CN: '帮助', ja_JP: 'ヘルプ' };
    // Drop zh_CN from string map (user cleared it in translate dialog).
    delete drafts[0]!.stringMap.zh_CN;
    const items = draftsToDefinitionItems(drafts, 'en_US', previous as any);
    expect(items[0]!.string).toEqual({ en_US: 'Tax', ja_JP: '税' });
    expect(items[0]!.help).toEqual({ en_US: 'Help', zh_CN: '帮助', ja_JP: 'ヘルプ' });
  });

  test('draftsToDefinitionItems selection labels follow draft maps after value rename', () => {
    const drafts = [
      {
        ...emptyDraftItem(),
        name: 'prio',
        type: 'selection',
        string: 'Priority',
        stringMap: { en_US: 'Priority' },
        selectionText: JSON.stringify([['hi', 'High']]),
        selectionLabelMaps: {
          hi: { en_US: 'High', zh_CN: '高' },
        },
      },
    ];
    const items = draftsToDefinitionItems(drafts, 'en_US', [
      {
        name: 'prio',
        type: 'selection',
        string: { en_US: 'Priority' },
        selection: [['high', { en_US: 'High', zh_CN: '高' }]],
      },
    ] as any);
    expect(items[0]!.selection).toEqual([['hi', { en_US: 'High', zh_CN: '高' }]]);
  });

  test('definitionItemsToDrafts / draftsToDefinitionItems round-trip', () => {
    const drafts = definitionItemsToDrafts(
      [
        { name: 'tier', type: 'selection', string: 'Tier', selection: [['a', 'A'], ['b', 'B']] },
        { name: 'flag', type: 'boolean', default: true, readonly: true },
      ],
      'en_US',
    );
    expect(drafts).toHaveLength(2);
    const items = draftsToDefinitionItems(drafts, 'en_US');
    expect(items[0]!.name).toBe('tier');
    expect(items[0]!.selection).toEqual([
      ['a', 'A'],
      ['b', 'B'],
    ]);
    expect(items[1]!.readonly).toBe(true);
    expect(items[1]!.default).toBe(true);
  });

  test('draftsToDefinitionItems persists only normalized selection options', () => {
    const items = draftsToDefinitionItems(
      [
        {
          ...emptyDraftItem(),
          name: 'tier',
          type: 'selection',
          selectionText: JSON.stringify([['a', 'A'], 1, { value: 'b', label: 'B' }, { value: 2 }]),
        },
      ],
      'en_US',
    );
    expect(items[0]!.selection).toEqual([
      ['a', 'A'],
      ['b', 'B'],
    ]);
  });

  test('draftsToDefinitionItems coerces defaults and skips blank names', () => {
    const items = draftsToDefinitionItems(
      [
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
      ],
      'en_US',
    );
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
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '' }], 'en_US'),
    ).toThrow(/requires selection JSON/);
    expect(() =>
      draftsToDefinitionItems(
        [{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '{' }],
        'en_US',
      ),
    ).toThrow(/invalid JSON/);
    expect(() =>
      draftsToDefinitionItems(
        [{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '[]' }],
        'en_US',
      ),
    ).toThrow(/non-empty selection array/);
    expect(() =>
      draftsToDefinitionItems(
        [{ ...emptyDraftItem(), name: 's', type: 'selection', selectionText: '[1]' }],
        'en_US',
      ),
    ).toThrow(/no usable options/);
    expect(() =>
      draftsToDefinitionItems(
        [
          {
            ...emptyDraftItem(),
            name: 's',
            type: 'selection',
            default: 'missing',
            selectionText: JSON.stringify([['a', 'A']]),
          },
        ],
        'en_US',
      ),
    ).toThrow(/default is not one of the options/);
    expect(
      draftsToDefinitionItems(
        [
          {
            ...emptyDraftItem(),
            name: 's',
            type: 'selection',
            default: 'a',
            selectionText: JSON.stringify([['a', 'A']]),
          },
        ],
        'en_US',
      )[0]!.default,
    ).toBe('a');
    expect(() =>
      draftsToDefinitionItems([{ ...emptyDraftItem(), name: 'x', type: 'binary' }], 'en_US'),
    ).toThrow(/unsupported property type/);
    expect(() =>
      draftsToDefinitionItems(
        [
          { ...emptyDraftItem(), name: 'dup', type: 'char' },
          { ...emptyDraftItem(), name: 'dup', type: 'integer' },
        ],
        'en_US',
      ),
    ).toThrow(/duplicate property name/);
    expect(draftsToDefinitionItems(null)).toEqual([]);
  });

  test('applyDraftCurrentLangToMaps / syncDraftLabelsForLang round-trip UI fields', () => {
    const draft = emptyDraftItem();
    draft.name = 'prio';
    draft.type = 'selection';
    draft.stringMap = { en_US: 'Priority', zh_CN: '优先级' };
    draft.helpMap = { en_US: 'Record urgency', zh_CN: '记录紧急程度' };
    draft.selectionLabelMaps = { high: { en_US: 'High', zh_CN: '高' } };
    syncDraftLabelsForLang(draft, 'zh_CN');
    expect(draft.string).toBe('优先级');
    expect(draft.help).toBe('记录紧急程度');
    expect(draft.selectionText).toBe(JSON.stringify([['high', '高']]));
    draft.string = '优先';
    draft.help = '紧急程度';
    applyDraftCurrentLangToMaps(draft, 'zh_CN');
    expect(draft.stringMap).toEqual({ en_US: 'Priority', zh_CN: '优先' });
    expect(draft.helpMap).toEqual({ en_US: 'Record urgency', zh_CN: '紧急程度' });
  });

  test('draftsToDefinitionItems writes help lang maps and omits blank help', () => {
    const withHelp = draftsToDefinitionItems(
      [
        {
          ...emptyDraftItem(),
          name: 'code',
          type: 'char',
          help: '外部编码',
          helpMap: { en_US: 'External reference code', zh_CN: '旧' },
        },
      ],
      'zh_CN',
    );
    expect(withHelp[0]!.help).toEqual({
      en_US: 'External reference code',
      zh_CN: '外部编码',
    });
    const blank = draftsToDefinitionItems([{ ...emptyDraftItem(), name: 'x', type: 'char' }], 'en_US');
    expect(blank[0]!.help).toBeUndefined();
  });

  test('selectionRowsFromDraft / writeSelectionRowsToDraft keep maps across value rename', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionText = JSON.stringify([
      ['low', '低'],
      ['normal', '普通'],
    ]);
    draft.selectionLabelMaps = {
      low: { en_US: 'Low', zh_CN: '低' },
      normal: { en_US: 'Normal', zh_CN: '普通' },
    };
    draft.selectionRowMapKeys = ['low', 'normal'];
    expect(selectionRowsFromDraft(draft)).toEqual([
      { value: 'low', label: '低' },
      { value: 'normal', label: '普通' },
    ]);
    writeSelectionRowsToDraft(
      draft,
      [
        { value: 'lo', label: '低' },
        { value: 'normal', label: '普通' },
        { value: '', label: '' },
      ],
      'zh_CN',
    );
    expect(draft.selectionLabelMaps.lo).toEqual({ en_US: 'Low', zh_CN: '低' });
    expect(draft.selectionLabelMaps.low).toBeUndefined();
    expect(draft.selectionLabelMaps.normal).toEqual({ en_US: 'Normal', zh_CN: '普通' });
    expect(JSON.parse(draft.selectionText)).toEqual([
      ['lo', '低'],
      ['normal', '普通'],
      ['', ''],
    ]);
  });

  test('writeSelectionRowsToDraft keeps maps across clear-and-retype of Value', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionText = JSON.stringify([['low', '低']]);
    draft.selectionLabelMaps = { low: { en_US: 'Low', zh_CN: '低' } };
    draft.selectionRowMapKeys = ['low'];

    writeSelectionRowsToDraft(draft, [{ value: '', label: '低' }], 'zh_CN');
    expect(draft.selectionLabelMaps.low).toEqual({ en_US: 'Low', zh_CN: '低' });
    expect(draft.selectionRowMapKeys).toEqual(['low']);

    writeSelectionRowsToDraft(draft, [{ value: 'lo', label: '低' }], 'zh_CN');
    expect(draft.selectionLabelMaps.lo).toEqual({ en_US: 'Low', zh_CN: '低' });
    expect(draft.selectionLabelMaps.low).toBeUndefined();
    expect(draft.selectionRowMapKeys).toEqual(['lo']);
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

  test('selectionRowsFromDraft falls back to selectionLabelMaps on JSON parse failure', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionText = '{not-json';
    draft.selectionLabelMaps = {
      low: { en_US: 'Low', zh_CN: '低' },
      high: { en_US: 'High' },
    };
    const rows = selectionRowsFromDraft(draft);
    expect(rows.map(r => r.value).sort()).toEqual(['high', 'low']);
    expect(rows.find(r => r.value === 'low')?.label).toBe('Low');
  });

  test('selectionRowsFromDraft empty selectionText yields empty rows', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionText = '';
    expect(selectionRowsFromDraft(draft)).toEqual([]);
  });

  test('syncDraftLabelsForLang tolerates invalid selectionText JSON', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionText = 'not-json';
    draft.selectionLabelMaps = { a: { en_US: 'A', zh_CN: '甲' } };
    syncDraftLabelsForLang(draft, 'zh_CN');
    expect(draft.selectionText).toBe(JSON.stringify([['a', '甲']]));
  });

  test('applyDraftCurrentLangToMaps returns early on invalid / empty selection JSON', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionLabelMaps = { a: { en_US: 'A' } };

    draft.selectionText = '';
    applyDraftCurrentLangToMaps(draft, 'en_US');
    expect(draft.selectionLabelMaps).toEqual({ a: { en_US: 'A' } });

    draft.selectionText = '{broken';
    applyDraftCurrentLangToMaps(draft, 'en_US');
    expect(draft.selectionLabelMaps).toEqual({ a: { en_US: 'A' } });

    draft.selectionText = '[]';
    applyDraftCurrentLangToMaps(draft, 'en_US');
    expect(draft.selectionLabelMaps).toEqual({ a: { en_US: 'A' } });
  });

  test('writeSelectionRowsToDraft / syncDraftLabelsForLang tolerate stringify failures', () => {
    const draft = emptyDraftItem();
    draft.type = 'selection';
    draft.selectionText = JSON.stringify([['a', 'A']]);
    draft.selectionLabelMaps = { a: { en_US: 'A' } };

    const origStringify = JSON.stringify;
    try {
      (JSON as any).stringify = () => {
        throw new Error('stringify boom');
      };
      writeSelectionRowsToDraft(draft, [{ value: 'b', label: 'B' }], 'en_US');
      expect(draft.selectionText).toBe('');
      expect(draft.selectionLabelMaps.b).toEqual({ en_US: 'B' });

      draft.selectionLabelMaps = { c: { en_US: 'C' } };
      draft.selectionText = 'not-json';
      syncDraftLabelsForLang(draft, 'en_US');
      expect(draft.selectionText).toBe('');
    } finally {
      JSON.stringify = origStringify;
    }
  });

  test('draftsToDefinitionItems omits help when map values are blank whitespace', () => {
    const items = draftsToDefinitionItems(
      [
        {
          ...emptyDraftItem(),
          name: 'x',
          type: 'char',
          help: '   ',
          helpMap: { en_US: '  ', zh_CN: '' },
        },
      ],
      'en_US',
    );
    expect(items[0]!.help).toBeUndefined();
  });

  test('syncDraftLabelsForLang no-ops selection for non-selection types', () => {
    const draft = emptyDraftItem();
    draft.type = 'char';
    draft.stringMap = { en_US: 'Label' };
    draft.helpMap = { en_US: 'Help' };
    draft.selectionText = 'should-stay';
    syncDraftLabelsForLang(draft, '');
    expect(draft.string).toBe('Label');
    expect(draft.help).toBe('Help');
    expect(draft.selectionText).toBe('should-stay');
  });
});
