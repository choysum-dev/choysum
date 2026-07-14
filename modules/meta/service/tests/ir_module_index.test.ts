// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeSearchCondition,
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
  normalizeOriginType,
  canReuseRunningSync,
} from '../models/ir_module_index';

type ModuleIndexRecord = {
  Id?: string;
  ModuleName?: string;
  OriginType?: string;
  OriginRef?: string;
  Available?: boolean;
  Version?: string;
  ManifestJson?: Record<string, unknown> | null;
  LocalPath?: string;
  LastSyncAt?: Date | string | null;
  LastBatchSyncAt?: Date | string | null;
  SyncRevision?: string;
  LastErrorMessage?: string;
  InstalledStatus?: string;
  InstalledVersion?: string;
  OriginTypes?: string;
  LocalVersion?: string;
  RegistryVersion?: string;
};

const describe = (_name: string, fn: () => void) => fn();
const it = (name: string, fn: () => void) => test(name, fn);

function record(fields: Partial<ModuleIndexRecord>): ModuleIndexRecord {
  return fields as ModuleIndexRecord;
}

// --------------- normalizeSearchCondition ---------------

describe('normalizeSearchCondition', () => {
  it('returns empty array as a default condition', () => {
    expect(normalizeSearchCondition([])).toEqual(['Available', '=', true]);
  });
  it('returns empty object as default condition', () => {
    expect(normalizeSearchCondition({})).toEqual(['Available', '=', true]);
  });
  it('returns non-empty array unchanged', () => {
    const cond = ['ModuleName', '=', 'test'];
    expect(normalizeSearchCondition(cond)).toBe(cond);
  });
  it('returns non-empty object unchanged', () => {
    const cond = { ModuleName: 'test' };
    expect(normalizeSearchCondition(cond)).toBe(cond);
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
  it('parses Field/Order aliases', () => {
    expect(parseSortSpecs([{ Field: 'Name', Order: 'DESC' }])).toEqual([{ field: 'Name', desc: true }]);
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
  it('falls back to ModuleName tiebreaker', () => {
    const specs: any[] = [];
    expect(compareBySpecs(record({ ModuleName: 'A' }), record({ ModuleName: 'B' }), specs)).toBeLessThan(0);
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
  it('falls back to enumerable keys', () => {
    expect(toPlainRecord({ ModuleName: 'test' }).ModuleName).toBe('test');
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
});

// --------------- normalizeOriginType ---------------

describe('normalizeOriginType', () => {
  it('returns all for empty/undefined', () => {
    expect(normalizeOriginType()).toBe('all');
    expect(normalizeOriginType('')).toBe('all');
  });
  it('returns all for "all"', () => {
    expect(normalizeOriginType('all')).toBe('all');
  });
  it('returns local/registry', () => {
    expect(normalizeOriginType('local')).toBe('local');
    expect(normalizeOriginType('registry')).toBe('registry');
  });
  it('returns empty string for invalid', () => {
    expect(normalizeOriginType('invalid')).toBe('');
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
