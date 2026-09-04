// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  assertSearchCondition,
  DEFAULT_MODULE_INDEX_SEARCH,
  toText,
  toComparableValue,
  parseSortSpecs,
  compareBySpecs,
  applySoftDeleteOptions,
  buildSortPushdownPlan,
  extractGroupedModuleNames,
  buildModuleNamesCondition,
  projectFields,
  toPlainRecord,
  pickNewestTimestamp,
  aggregateRows,
  assertOriginType,
  canReuseRunningSync,
  type ModuleIndexRecord,
} from '../models/_module_index_query';

const describe = (_name: string, fn: () => void) => fn();
const it = (name: string, fn: () => void) => test(name, fn);

function record(fields: Partial<ModuleIndexRecord>): ModuleIndexRecord {
  return fields as ModuleIndexRecord;
}

// --------------- assertSearchCondition ---------------

describe('assertSearchCondition', () => {
  it('rejects empty array', () => {
    expect(() => assertSearchCondition([])).toThrow(/must not be empty/);
  });
  it('rejects empty object', () => {
    expect(() => assertSearchCondition({})).toThrow(/must not be empty/);
  });
  it('returns non-empty array unchanged', () => {
    const cond = ['ModuleName', '=', 'test'];
    expect(assertSearchCondition(cond)).toBe(cond);
  });
  it('returns non-empty object unchanged', () => {
    const cond = { ModuleName: 'test' };
    expect(assertSearchCondition(cond)).toBe(cond);
  });
  it('exposes DEFAULT_MODULE_INDEX_SEARCH for callers that want the catalog default', () => {
    expect(DEFAULT_MODULE_INDEX_SEARCH).toEqual(['Available', '=', true]);
  });
});

// --------------- toText ---------------

describe('toText', () => {
  it('trims and lowercases', () => {
    expect(toText('  HELLO  ')).toBe('hello');
  });
  it('handles null/undefined', () => {
    expect(toText(null)).toBe('');
    expect(toText(undefined)).toBe('');
  });
  it('handles numbers', () => {
    expect(toText(42)).toBe('42');
  });
});

// --------------- toComparableValue ---------------

describe('toComparableValue', () => {
  it('returns null for null/undefined', () => {
    expect(toComparableValue(null)).toBeNull();
    expect(toComparableValue(undefined)).toBeNull();
  });
  it('converts Date to timestamp', () => {
    const d = new Date('2024-01-15T00:00:00Z');
    expect(toComparableValue(d)).toBe(d.getTime());
  });
  it('converts boolean to 0/1', () => {
    expect(toComparableValue(true)).toBe(1);
    expect(toComparableValue(false)).toBe(0);
  });
  it('returns numbers unchanged', () => {
    expect(toComparableValue(42)).toBe(42);
  });
  it('returns bigint unchanged', () => {
    expect(toComparableValue(10n)).toBe(10n);
  });
  it('parses date-like strings to timestamps', () => {
    const v = toComparableValue('2024-01-15');
    expect(typeof v).toBe('number');
  });
  it('lowercases non-date strings', () => {
    expect(toComparableValue('  ABC  ')).toBe('abc');
  });
  it('returns  empty string for blank input', () => {
    expect(toComparableValue('   ')).toBe('');
  });
  it('stringifies other values', () => {
    expect(toComparableValue({ x: 1 })).toBe('[object object]');
  });
});

// --------------- parseSortSpecs ---------------

describe('parseSortSpecs', () => {
  it('returns empty for falsy input', () => {
    expect(parseSortSpecs(null)).toEqual([]);
  });
  it('parses string array', () => {
    expect(parseSortSpecs(['Name', 'Version'])).toEqual([
      { field: 'Name', desc: false },
      { field: 'Version', desc: false },
    ]);
  });
  it('parses single string', () => {
    expect(parseSortSpecs('Name')).toEqual([{ field: 'Name', desc: false }]);
  });
  it('parses object with order', () => {
    expect(parseSortSpecs([{ field: 'Name', order: 'desc' }])).toEqual([{ field: 'Name', desc: true }]);
  });
  it('skips items without field', () => {
    expect(parseSortSpecs([{ x: 1 }])).toEqual([]);
  });
  it('skips falsy and non-object items', () => {
    expect(parseSortSpecs([null, undefined, 42, false])).toEqual([]);
  });
  it('parses Field/Order aliases', () => {
    expect(parseSortSpecs([{ Field: 'Name', Order: 'DESC' }])).toEqual([{ field: 'Name', desc: true }]);
  });
  it('defaults order to asc when omitted', () => {
    expect(parseSortSpecs([{ field: 'Name' }])).toEqual([{ field: 'Name', desc: false }]);
  });
});

// --------------- compareBySpecs ---------------

describe('compareBySpecs', () => {
  it('sorts by field ascending', () => {
    const specs = [{ field: 'ModuleName', desc: false }];
    expect(compareBySpecs(record({ ModuleName: 'A' }), record({ ModuleName: 'B' }), specs)).toBeLessThan(0);
    expect(compareBySpecs(record({ ModuleName: 'B' }), record({ ModuleName: 'A' }), specs)).toBeGreaterThan(0);
  });
  it('sorts by field descending', () => {
    const specs = [{ field: 'ModuleName', desc: true }];
    expect(compareBySpecs(record({ ModuleName: 'A' }), record({ ModuleName: 'B' }), specs)).toBeGreaterThan(0);
  });
  it('handles null values', () => {
    const specs = [{ field: 'Version', desc: false }];
    // null sorts before non-null in ascending order.
    expect(compareBySpecs(record({ Version: undefined }), record({ Version: '1' }), specs)).toBeLessThan(0);
    expect(compareBySpecs(record({ Version: '1' }), record({ Version: undefined }), specs)).toBeGreaterThan(0);
  });
  it('handles null values descending', () => {
    const specs = [{ field: 'Version', desc: true }];
    expect(compareBySpecs(record({ Version: undefined }), record({ Version: '1' }), specs)).toBeGreaterThan(0);
    expect(compareBySpecs(record({ Version: '1' }), record({ Version: undefined }), specs)).toBeLessThan(0);
  });
  it('skips when both sides are null and continues', () => {
    const specs = [
      { field: 'Version', desc: false },
      { field: 'ModuleName', desc: false },
    ];
    expect(compareBySpecs(record({ ModuleName: 'A' }), record({ ModuleName: 'B' }), specs)).toBeLessThan(0);
  });
  it('falls back to ModuleName tiebreaker', () => {
    const specs: any[] = [];
    expect(compareBySpecs(record({ ModuleName: 'A' }), record({ ModuleName: 'B' }), specs)).toBeLessThan(0);
    expect(compareBySpecs(record({ ModuleName: 'B' }), record({ ModuleName: 'A' }), specs)).toBeGreaterThan(0);
  });
  it('returns 0 when ModuleName tiebreaker is equal', () => {
    expect(compareBySpecs(record({ ModuleName: 'same' }), record({ ModuleName: 'same' }), [])).toBe(0);
  });
  it('uses descending compare when av is greater', () => {
    const specs = [{ field: 'Version', desc: true }];
    expect(compareBySpecs(record({ Version: '2' }), record({ Version: '1' }), specs)).toBeLessThan(0);
    expect(compareBySpecs(record({ Version: '1' }), record({ Version: '2' }), specs)).toBeGreaterThan(0);
  });
  it('continues when field values are equal', () => {
    const specs = [
      { field: 'Version', desc: false },
      { field: 'ModuleName', desc: false },
    ];
    expect(compareBySpecs(record({ Version: '1', ModuleName: 'A' }), record({ Version: '1', ModuleName: 'B' }), specs)).toBeLessThan(0);
  });
});

// --------------- applySoftDeleteOptions ---------------

describe('applySoftDeleteOptions', () => {
  it('applies withDeleted and onlyDeleted from source', () => {
    const target: Record<string, unknown> = {};
    applySoftDeleteOptions(target, { withDeleted: true, onlyDeleted: false });
    expect(target.withDeleted).toBe(true);
    expect(target.onlyDeleted).toBe(false);
  });
  it('ignores missing keys', () => {
    const target: Record<string, unknown> = {};
    applySoftDeleteOptions(target, {});
    expect(Object.keys(target)).toHaveLength(0);
  });
});

// --------------- buildSortPushdownPlan ---------------

describe('buildSortPushdownPlan', () => {
  it('supports ModuleName, Available, LastSyncAt, LastBatchSyncAt', () => {
    const plan = buildSortPushdownPlan([
      { field: 'ModuleName', desc: true },
      { field: 'Available', desc: false },
      { field: 'LastSyncAt', desc: false },
      { field: 'LastBatchSyncAt', desc: true },
    ]);
    expect(plan.supported).toBe(true);
    // ModuleName already in sortSpecs, so fallback does not add a duplicate.
    expect(plan.orderBy).toHaveLength(4);
    expect(plan.aggregateFields).toHaveLength(3); // Available, LastSyncAt, LastBatchSyncAt
  });
  it('returns unsupported for unknown field', () => {
    const plan = buildSortPushdownPlan([{ field: 'UnknownField', desc: false }]);
    expect(plan.supported).toBe(false);
  });
  it('appends ModuleName as final sort when not in spec list', () => {
    const plan = buildSortPushdownPlan([{ field: 'Available', desc: false }]);
    expect(plan.supported).toBe(true);
    expect(plan.orderBy).toHaveLength(2); // Available + fallback ModuleName
    expect(plan.orderBy[1].field).toBe('ModuleName');
  });
  it('reuses aggregate alias when the same field appears twice', () => {
    const plan = buildSortPushdownPlan([
      { field: 'Available', desc: false },
      { field: 'Available', desc: true },
    ]);
    expect(plan.supported).toBe(true);
    expect(plan.aggregateFields).toHaveLength(1);
    expect(plan.orderBy[0].field).toBe(plan.orderBy[1].field);
  });
  it('supports empty sort specs with ModuleName fallback', () => {
    const plan = buildSortPushdownPlan([]);
    expect(plan.supported).toBe(true);
    expect(plan.orderBy).toEqual([{ field: 'ModuleName', order: 'asc' }]);
  });
});

// --------------- extractGroupedModuleNames ---------------

describe('extractGroupedModuleNames', () => {
  it('extracts ModuleName from rows', () => {
    expect(extractGroupedModuleNames([{ ModuleName: 'auth' }, { ModuleName: 'base' }])).toEqual(['auth', 'base']);
  });
  it('handles module_name fallback', () => {
    expect(extractGroupedModuleNames([{ module_name: 'core' }])).toEqual(['core']);
  });
  it('skips empty names', () => {
    expect(extractGroupedModuleNames([{ ModuleName: '' }, { ModuleName: '  ' }])).toEqual([]);
  });
  it('handles null/undefined rows list', () => {
    expect(extractGroupedModuleNames(null as any)).toEqual([]);
    expect(extractGroupedModuleNames(undefined as any)).toEqual([]);
  });
  it('prefers ModuleName over module_name when both exist', () => {
    expect(extractGroupedModuleNames([{ ModuleName: 'auth', module_name: 'ignored' }])).toEqual(['auth']);
  });
  it('skips null rows and nullish names', () => {
    expect(extractGroupedModuleNames([null, { ModuleName: null }, { module_name: undefined }])).toEqual([]);
  });
});

// --------------- buildModuleNamesCondition ---------------

describe('buildModuleNamesCondition', () => {
  it('builds IN condition for non-empty names', () => {
    const cond = buildModuleNamesCondition([], ['auth', 'base']);
    expect(cond).toEqual(['ModuleName', 'in', ['auth', 'base']]);
  });
  it('returns never-match for empty names', () => {
    const cond = buildModuleNamesCondition([], []);
    expect(cond).toEqual(['Id', '=', '__never_match__']);
  });
  it('wraps existing condition with AND', () => {
    const cond = buildModuleNamesCondition(['Available', '=', true], ['auth']);
    expect(cond).toEqual({
      And: [
        ['Available', '=', true],
        ['ModuleName', 'in', ['auth']],
      ],
    });
  });
  it('treats null and empty-object base as only IN condition', () => {
    expect(buildModuleNamesCondition(null, ['auth'])).toEqual(['ModuleName', 'in', ['auth']]);
    expect(buildModuleNamesCondition({}, ['auth'])).toEqual(['ModuleName', 'in', ['auth']]);
  });
});

// --------------- projectFields ---------------

describe('projectFields', () => {
  const rows = [record({ ModuleName: 'auth', Version: '1.0', Available: true }), record({ ModuleName: 'base', Version: '2.0', Available: false })];

  it('returns rows unchanged when requestedFields is empty', () => {
    expect(projectFields(rows, [])).toBe(rows);
  });
  it('projects only requested fields', () => {
    const result = projectFields(rows, ['ModuleName', 'Version']);
    expect(result[0]).toEqual({ ModuleName: 'auth', Version: '1.0' });
    expect(result[0].Available).toBeUndefined();
  });
  it('deduplicates and filters blocked fields', () => {
    const result = projectFields(rows, ['ModuleName', 'ModuleName', '__proto__']);
    expect(result[0]).toEqual({ ModuleName: 'auth' });
  });
  it('returns rows unchanged when all requested fields are blocked or blank', () => {
    expect(projectFields(rows, ['__proto__', 'constructor', 'prototype', '  ', ''])).toBe(rows);
  });
});

// --------------- toPlainRecord ---------------

describe('toPlainRecord', () => {
  it('returns empty record for null/undefined', () => {
    expect(toPlainRecord(null)).toEqual({});
    expect(toPlainRecord(undefined)).toEqual({});
  });
  it('uses toPlainObject when available', () => {
    const input = { toPlainObject: () => ({ ModuleName: 'test' }), extra: 'ignored' };
    expect(toPlainRecord(input).ModuleName).toBe('test');
  });
  it('falls back when toPlainObject throws', () => {
    const input = {
      ModuleName: 'fallback',
      toPlainObject: () => {
        throw new Error('boom');
      },
    };
    expect(toPlainRecord(input).ModuleName).toBe('fallback');
  });
  it('falls back to enumerable keys', () => {
    expect(toPlainRecord({ ModuleName: 'test' }).ModuleName).toBe('test');
  });
  it('skips dangerous keys when copying enumerable fields', () => {
    const input = JSON.parse('{"ModuleName":"safe","__proto__":{"polluted":true}}');
    const result = toPlainRecord(input);
    expect(result.ModuleName).toBe('safe');
    expect(Object.prototype.hasOwnProperty.call(result, '__proto__')).toBe(false);
    expect((result as any).polluted).toBeUndefined();
  });
  it('returns empty for non-object values', () => {
    expect(toPlainRecord('x')).toEqual({});
    expect(toPlainRecord(42)).toEqual({});
  });
});

// --------------- pickNewestTimestamp ---------------

describe('pickNewestTimestamp', () => {
  it('picks the newest Date', () => {
    const result = pickNewestTimestamp([new Date('2024-01-01'), new Date('2025-06-15')]);
    expect(result).toEqual(new Date('2025-06-15'));
  });
  it('handles string dates', () => {
    const result = pickNewestTimestamp(['2024-01-01', '2025-06-15']);
    expect(result).toBe('2025-06-15');
  });
  it('skips null/undefined', () => {
    expect(pickNewestTimestamp([null, undefined])).toBeUndefined();
  });
  it('returns first value when all are unparseable', () => {
    const result = pickNewestTimestamp(['not-a-date', 'also-not']);
    expect(result).toBe('not-a-date');
  });
  it('keeps prior pick when later value is unparseable', () => {
    const result = pickNewestTimestamp([new Date('2024-01-01'), 'not-a-date']);
    expect(result).toEqual(new Date('2024-01-01'));
  });
});

// --------------- aggregateRows ---------------

describe('aggregateRows', () => {
  it('merges local and registry rows for same module', () => {
    const rows = [
      record({ ModuleName: 'auth', OriginType: 'local', Version: '1.0.0', Available: true }),
      record({ ModuleName: 'auth', OriginType: 'registry', Version: '1.0.1', Available: true }),
    ];
    const result = aggregateRows(rows);
    expect(result).toHaveLength(1);
    expect(result[0].ModuleName).toBe('auth');
    expect(result[0].OriginTypes).toBe('local, registry');
    expect(result[0].LocalVersion).toBe('1.0.0');
    expect(result[0].RegistryVersion).toBe('1.0.1');
    expect(result[0].Version).toBe('1.0.1'); // registry preferred
    expect(result[0].Available).toBe(true);
  });

  it('marks module as unavailable when all rows are unavailable', () => {
    const rows = [record({ ModuleName: 'x', OriginType: 'local', Available: false }), record({ ModuleName: 'x', OriginType: 'registry', Available: false })];
    const result = aggregateRows(rows);
    expect(result[0].Available).toBe(false);
  });

  it('skips rows with empty ModuleName', () => {
    const rows = [record({ ModuleName: '', OriginType: 'local' })];
    expect(aggregateRows(rows)).toHaveLength(0);
  });

  it('handles single-origin rows gracefully', () => {
    const rows = [record({ ModuleName: 'solo', OriginType: 'local', Version: '2.0' })];
    const result = aggregateRows(rows);
    expect(result[0].ModuleName).toBe('solo');
    expect(result[0].OriginType).toBe('local');
    expect(result[0].InstalledStatus).toBe('uninstalled');
  });

  it('prefers registry-first bucket then local for OriginTypes order', () => {
    const rows = [
      record({ ModuleName: 'auth', OriginType: 'registry', Version: '2.0', Available: true }),
      record({ ModuleName: 'auth', OriginType: 'local', Version: '1.0', Available: true }),
    ];
    const result = aggregateRows(rows);
    expect(result[0].OriginTypes).toBe('local, registry');
    expect(result[0].LocalVersion).toBe('1.0');
    expect(result[0].RegistryVersion).toBe('2.0');
  });

  it('aggregates registry-only and unknown origin rows', () => {
    const rows = [
      record({
        ModuleName: 'r',
        OriginType: 'registry',
        Version: '3',
        InstalledStatus: 'installed',
        InstalledVersion: '3',
        LastErrorMessage: 'e',
        SyncRevision: 'rev',
      }),
      record({ ModuleName: 'r', OriginType: 'other', Version: '9' }),
    ];
    const result = aggregateRows(rows);
    expect(result[0].OriginType).toBe('registry');
    expect(result[0].OriginTypes).toBe('registry');
    expect(result[0].InstalledStatus).toBe('installed');
    expect(result[0].InstalledVersion).toBe('3');
    expect(result[0].LastErrorMessage).toBe('e');
    expect(result[0].SyncRevision).toBe('rev');
  });

  it('falls back to bucket[0] and empty OriginTypes for unknown origins', () => {
    const rows = [record({ ModuleName: 'x', OriginType: 'mirror', Version: '1', Id: 'id-1' })];
    const result = aggregateRows(rows);
    expect(result[0].Id).toBe('id-1');
    expect(result[0].OriginType).toBe('mirror');
    expect(result[0].OriginTypes).toBe('mirror');
  });

  it('defaults OriginType to local when base origin is blank', () => {
    const rows = [record({ ModuleName: 'blank', OriginType: '   ', Id: '' })];
    const result = aggregateRows(rows);
    expect(result[0].OriginType).toBe('local');
    expect(result[0].OriginTypes).toBe('');
  });

  it('defaults OriginType to local when OriginType is nullish', () => {
    const rows = [record({ ModuleName: 'nullish', OriginType: undefined, Id: 'n1' })];
    const result = aggregateRows(rows);
    expect(result[0].OriginType).toBe('local');
    expect(result[0].OriginTypes).toBe('');
  });

  it('fills Id from registry when local Id is empty', () => {
    const rows = [
      record({ ModuleName: 'x', OriginType: 'local', Id: '', Version: '1' }),
      record({ ModuleName: 'x', OriginType: 'registry', Id: 'reg-1', Version: '2' }),
    ];
    expect(aggregateRows(rows)[0].Id).toBe('reg-1');
  });

  it('fills OriginRef and ManifestJson from registry when local is empty', () => {
    const rows = [
      record({ ModuleName: 'x', OriginType: 'local', OriginRef: '', ManifestJson: null }),
      record({
        ModuleName: 'x',
        OriginType: 'registry',
        OriginRef: 'npm:x',
        ManifestJson: { name: 'x' },
        LastErrorMessage: 'oops',
      }),
    ];
    const result = aggregateRows(rows)[0];
    expect(result.OriginRef).toBe('npm:x');
    expect(result.ManifestJson).toEqual({ name: 'x' });
    expect(result.LastErrorMessage).toBe('oops');
  });
});

// --------------- assertOriginType ---------------

describe('assertOriginType', () => {
  it('rejects empty/undefined (callers apply ?? all)', () => {
    expect(() => assertOriginType()).toThrow(/originType/);
    expect(() => assertOriginType('')).toThrow(/originType/);
  });
  it('returns all for "all"', () => {
    expect(assertOriginType('all')).toBe('all');
  });
  it('returns local/registry', () => {
    expect(assertOriginType('local')).toBe('local');
    expect(assertOriginType('registry')).toBe('registry');
  });
  it('rejects invalid', () => {
    expect(() => assertOriginType('invalid')).toThrow(/originType/);
  });
});

// --------------- canReuseRunningSync ---------------

describe('canReuseRunningSync', () => {
  it('all reuses all', () => {
    expect(canReuseRunningSync('all', 'all')).toBe(true);
  });
  it('all cannot reuse specific', () => {
    expect(canReuseRunningSync('all', 'local')).toBe(false);
  });
  it('specific can reuse all', () => {
    expect(canReuseRunningSync('local', 'all')).toBe(true);
  });
  it('specific reuses same specific', () => {
    expect(canReuseRunningSync('registry', 'registry')).toBe(true);
  });
  it('specific cannot reuse different specific', () => {
    expect(canReuseRunningSync('local', 'registry')).toBe(false);
  });
});
