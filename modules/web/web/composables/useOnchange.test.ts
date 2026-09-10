// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  normalizeDiffRelation,
  buildDiffFieldsMeta,
  buildRelationFieldSet,
  relationAwareMinimize,
  augmentCollapsedWithRelationRoots,
  isIndexSeg,
  normalizeArrayIndexInPath,
  extractBaseRoot,
  toOneHopFieldSignal,
  collectAncestorCollectionRoots,
  slimRelationRefsForChanged,
  getArrayAtPath,
  findIndexById,
  deepFindById,
  applyRowPatchToArray,
  toSelectorPath,
  detectStructuralChangedRelations,
  looksLikeRelation,
} from './useOnchange';
import type { WebModelStore, WebFieldMetadata } from '@/web/web/stores/modelStore';

// --------------- minimal store stub ---------------

function stubStore(overrides?: Partial<Record<string, WebFieldMetadata>>): WebModelStore<any> {
  return {
    fieldsMetadata: overrides ?? {},
  } as unknown as WebModelStore<any>;
}

// --------------- normalizeDiffRelation ---------------

describe('normalizeDiffRelation', () => {
  test('maps manytoone -> ManyToOne', () => {
    expect(normalizeDiffRelation('manytoone')).toBe('ManyToOne');
  });
  test('maps manytooneref -> ManyToOne', () => {
    expect(normalizeDiffRelation('manytooneref')).toBe('ManyToOne');
  });
  test('maps onetomany -> OneToMany', () => {
    expect(normalizeDiffRelation('onetomany')).toBe('OneToMany');
  });
  test('maps manytomany -> ManyToMany', () => {
    expect(normalizeDiffRelation('manytomany')).toBe('ManyToMany');
  });
  test('maps manytomanyref -> ManyToMany', () => {
    expect(normalizeDiffRelation('manytomanyref')).toBe('ManyToMany');
  });
  test('returns undefined for unknown types', () => {
    expect(normalizeDiffRelation('string')).toBeUndefined();
  });
  test('returns undefined for undefined input', () => {
    expect(normalizeDiffRelation(undefined)).toBeUndefined();
  });
  test('is case-insensitive', () => {
    expect(normalizeDiffRelation('ManyToOne')).toBe('ManyToOne');
  });
});

// --------------- looksLikeRelation ---------------

describe('looksLikeRelation', () => {
  test('returns false for undefined metadata', () => {
    expect(looksLikeRelation(undefined)).toBe(false);
  });
  test('returns true for manytoone', () => {
    expect(looksLikeRelation({ id: '1', type: 'manytoone', typeAnnotation: '' })).toBe(true);
  });
  test('returns true for onetomany', () => {
    expect(looksLikeRelation({ id: '2', type: 'onetomany', typeAnnotation: '' })).toBe(true);
  });
  test('returns false for string', () => {
    expect(looksLikeRelation({ id: '3', type: 'string', typeAnnotation: '' })).toBe(false);
  });
});

// --------------- buildDiffFieldsMeta ---------------

describe('buildDiffFieldsMeta', () => {
  test('returns empty object for store without fieldsMetadata', () => {
    const s = stubStore();
    expect(buildDiffFieldsMeta(s)).toEqual({});
  });

  test('returns normalized metadata for relation fields', () => {
    const s = stubStore({
      lines: { id: '1', type: 'oneToMany', typeAnnotation: 'Line[]' } as WebFieldMetadata,
      partner: { id: '2', type: 'manyToOne', typeAnnotation: 'Partner' } as WebFieldMetadata,
      name: { id: '3', type: 'string', typeAnnotation: 'string' } as WebFieldMetadata,
    });
    const meta = buildDiffFieldsMeta(s);
    expect(meta.lines?.relation).toBe('OneToMany');
    expect(meta.partner?.relation).toBe('ManyToOne');
    expect(meta.name?.relation).toBeUndefined();
    expect(meta.name?.type).toBe('string');
  });

  test('caches result and returns same normalized object on repeated calls', () => {
    const s = stubStore({ x: { id: 'a', type: 'integer', typeAnnotation: 'number' } as WebFieldMetadata });
    const a = buildDiffFieldsMeta(s);
    const b = buildDiffFieldsMeta(s);
    // Same identity when source reference unchanged
    expect(a).toBe(b);
  });
});

// --------------- buildRelationFieldSet ---------------

describe('buildRelationFieldSet', () => {
  test('returns empty set for store without fields', () => {
    expect(buildRelationFieldSet(stubStore()).size).toBe(0);
  });

  test('collects only relation field names', () => {
    const s = stubStore({
      orders: { id: 'o', type: 'oneToMany', typeAnnotation: '' } as WebFieldMetadata,
      title: { id: 't', type: 'string', typeAnnotation: '' } as WebFieldMetadata,
      tags: { id: 'g', type: 'manyToMany', typeAnnotation: '' } as WebFieldMetadata,
    });
    const set = buildRelationFieldSet(s);
    expect(set.has('orders')).toBe(true);
    expect(set.has('tags')).toBe(true);
    expect(set.has('title')).toBe(false);
  });
});

// --------------- isIndexSeg ---------------

describe('isIndexSeg', () => {
  test('returns true for digits-only', () => {
    expect(isIndexSeg('0')).toBe(true);
    expect(isIndexSeg('42')).toBe(true);
  });
  test('returns false for non-digits', () => {
    expect(isIndexSeg('abc')).toBe(false);
    expect(isIndexSeg('a1')).toBe(false);
    expect(isIndexSeg('')).toBe(false);
  });
});

// --------------- normalizeArrayIndexInPath ---------------

describe('normalizeArrayIndexInPath', () => {
  test('replaces bracket indices with dot indices', () => {
    expect(normalizeArrayIndexInPath('Lines[0].Qty')).toBe('Lines.0.Qty');
  });
  test('handles multiple bracket indices', () => {
    expect(normalizeArrayIndexInPath('Lines[0].Batches[1].Qty')).toBe('Lines.0.Batches.1.Qty');
  });
  test('returns same path if no brackets', () => {
    expect(normalizeArrayIndexInPath('a.b.c')).toBe('a.b.c');
  });
  test('returns empty string for falsy input', () => {
    expect(normalizeArrayIndexInPath('')).toBe('');
  });
});

// --------------- extractBaseRoot ---------------

describe('extractBaseRoot', () => {
  test('extracts first identifier segment', () => {
    expect(extractBaseRoot('Lines.0.Qty')).toBe('Lines');
  });
  test('extracts root even with bracket syntax', () => {
    expect(extractBaseRoot('Lines(id=1).Qty')).toBe('Lines');
  });
  test('returns single segment', () => {
    expect(extractBaseRoot('Name')).toBe('Name');
  });
  test('handles empty string', () => {
    expect(extractBaseRoot('')).toBe('');
  });
});

// --------------- toOneHopFieldSignal ---------------

describe('toOneHopFieldSignal', () => {
  test('strips indices from leaf paths', () => {
    expect(toOneHopFieldSignal('Lines.1.UnitPrice')).toBe('Lines.UnitPrice');
  });
  test('strips nested indices', () => {
    expect(toOneHopFieldSignal('Lines.1.Batches.0.Qty')).toBe('Lines.Batches.Qty');
  });
  test('returns null for paths without dots', () => {
    expect(toOneHopFieldSignal('Name')).toBeNull();
  });
  test('returns null for empty input', () => {
    expect(toOneHopFieldSignal('')).toBeNull();
  });
});

// --------------- collectAncestorCollectionRoots ---------------

describe('collectAncestorCollectionRoots', () => {
  test('returns Lines for Lines.1.UnitPrice', () => {
    expect(collectAncestorCollectionRoots('Lines.1.UnitPrice')).toEqual(['Lines']);
  });
  test('returns both ancestor roots for nested collections', () => {
    expect(collectAncestorCollectionRoots('Lines.1.Batches.0.Qty')).toEqual(['Lines', 'Lines.Batches']);
  });
  test('returns empty for empty input', () => {
    expect(collectAncestorCollectionRoots('')).toEqual([]);
  });
  test('returns empty for leaf without indices', () => {
    expect(collectAncestorCollectionRoots('a.b.c')).toEqual([]);
  });
});

// --------------- getArrayAtPath ---------------

describe('getArrayAtPath', () => {
  test('returns array at given field path', () => {
    const root = { Lines: [{ Id: 1 }, { Id: 2 }] };
    expect(getArrayAtPath(root, 'Lines')).toEqual([{ Id: 1 }, { Id: 2 }]);
  });
  test('returns null for non-array field', () => {
    expect(getArrayAtPath({ x: 'string' }, 'x')).toBeNull();
  });
  test('returns null for missing path', () => {
    expect(getArrayAtPath({}, 'a.b')).toBeNull();
  });
  test('traverses nested objects', () => {
    const root = { a: { b: [{ Id: 1 }] } };
    expect(getArrayAtPath(root, 'a.b')).toEqual([{ Id: 1 }]);
  });
});

// --------------- findIndexById ---------------

describe('findIndexById', () => {
  const arr = [{ Id: 10 }, { Id: 20 }, { id: 30 }];

  test('finds by Id', () => {
    expect(findIndexById(arr, 10)).toBe(0);
  });
  test('finds by id (lowercase)', () => {
    expect(findIndexById(arr, 30)).toBe(2);
  });
  test('returns -1 when not found', () => {
    expect(findIndexById(arr, 99)).toBe(-1);
  });
  test('returns -1 for non-array', () => {
    expect(findIndexById(null as any, 1)).toBe(-1);
  });
});

// --------------- deepFindById ---------------

describe('deepFindById', () => {
  const root = {
    Lines: [
      { Id: 1, Batches: [{ Id: 10, Qty: 5 }] },
      { Id: 2, Batches: [{ Id: 20, Qty: 3 }] },
    ],
  };

  test('finds at first level', () => {
    const hit = deepFindById(root, ['Lines'], 2);
    expect(hit?.idx).toBe(1);
  });

  test('finds at nested level', () => {
    const hit = deepFindById(root, ['Lines', 'Batches'], 20);
    expect(hit?.idx).toBe(0);
  });

  test('returns null when not found', () => {
    expect(deepFindById(root, ['Lines', 'Batches'], 99)).toBeNull();
  });

  test('returns null for empty segments', () => {
    expect(deepFindById(root, [], 1)).toBeNull();
  });
});

// --------------- applyRowPatchToArray ---------------

describe('applyRowPatchToArray', () => {
  test('applies patch values excluding meta keys', () => {
    const arr = [{ Id: 1, Qty: 10, Price: 5 }];
    applyRowPatchToArray(arr, 0, { Qty: 20, Price: 8, Id: 99, pos: 3 });
    expect(arr[0].Qty).toBe(20);
    expect(arr[0].Price).toBe(8);
    // meta keys skipped
    expect(arr[0].Id).toBe(1);
  });

  test('does nothing for out-of-range index', () => {
    const arr: any[] = [{ Id: 1 }];
    applyRowPatchToArray(arr, 99, { Qty: 5 });
    expect(arr[0].Qty).toBeUndefined();
  });

  test('does nothing for non-array', () => {
    applyRowPatchToArray(null as any, 0, { x: 1 });
    // no throw
  });
});

// --------------- augmentCollapsedWithRelationRoots ---------------

describe('augmentCollapsedWithRelationRoots', () => {
  test('adds relation roots from full leaf paths into target set', () => {
    const target = new Set<string>(['Name']);
    const fullLeaves = new Set<string>(['Lines.0.Qty', 'Lines.0.Price', 'Tags.1.Label']);
    augmentCollapsedWithRelationRoots(target, fullLeaves);
    expect(target.has('Lines')).toBe(true);
    expect(target.has('Tags')).toBe(true);
    expect(target.has('Name')).toBe(true);
  });

  test('no-ops when fullLeafPaths is empty', () => {
    const target = new Set<string>(['A']);
    augmentCollapsedWithRelationRoots(target, new Set());
    expect(target.size).toBe(1);
  });

  test('no-ops for leaf paths without dots', () => {
    const target = new Set<string>();
    augmentCollapsedWithRelationRoots(target, new Set<string>(['A', 'B']));
    expect(target.size).toBe(0);
  });
});

// --------------- toSelectorPath ---------------

describe('toSelectorPath', () => {
  const root = {
    Lines: [
      { Id: 100, UnitPrice: 5 },
      { Id: 200, UnitPrice: 8 },
    ],
    Lines2: [{ id: 'abc', Qty: 3 }],
  };

  test('converts index segment to id-based selector', () => {
    expect(toSelectorPath(root, 'Lines.0.UnitPrice')).toBe('Lines(id=100).UnitPrice');
  });

  test('falls back to bracket for missing row', () => {
    expect(toSelectorPath(root, 'Lines.99.UnitPrice')).toBe('Lines[99].UnitPrice');
  });

  test('uses lowercase id', () => {
    expect(toSelectorPath(root, 'Lines2.0.Qty')).toBe('Lines2(id=abc).Qty');
  });

  test('returns null for flat path', () => {
    expect(toSelectorPath(root, 'Name')).toBeNull();
  });

  test('returns null for empty input', () => {
    expect(toSelectorPath(root, '')).toBeNull();
  });
});

// --------------- relationAwareMinimize ---------------

describe('relationAwareMinimize', () => {
  const relStore = stubStore({
    Lines: { id: 'L', type: 'oneToMany', typeAnnotation: '' } as WebFieldMetadata,
  });

  test('returns empty for empty paths', () => {
    expect(relationAwareMinimize([], relStore, true)).toEqual([]);
  });

  test('collapses relation children to root when collapse=true', () => {
    const paths = ['Lines.0.Qty', 'Lines.0.Price', 'Name'];
    const result = relationAwareMinimize(paths, relStore, true);
    expect(result).toContain('Lines');
    expect(result).toContain('Name');
    expect(result).not.toContain('Lines.0.Qty');
  });

  test('keeps all paths when collapse=false', () => {
    const paths = ['Lines.0.Qty', 'Name'];
    const result = relationAwareMinimize(paths, relStore, false);
    expect(result).toEqual(['Lines.0.Qty', 'Name']);
  });

  test('keeps parent path when child is a prefix extension', () => {
    const paths = ['A', 'A.B'];
    const result = relationAwareMinimize(paths, stubStore(), false);
    // Parent (shorter path) is kept; child is skipped.
    expect(result).toEqual(['A']);
  });
});

// --------------- slimRelationRefsForChanged ---------------

describe('slimRelationRefsForChanged', () => {
  const refMeta: Record<string, WebFieldMetadata> = {
    partner: { id: 'p', type: 'manyToOneRef', typeAnnotation: '' } as WebFieldMetadata,
    tags: { id: 't', type: 'manyToManyRef', typeAnnotation: '' } as WebFieldMetadata,
    name: { id: 'n', type: 'string', typeAnnotation: '' } as WebFieldMetadata,
  };

  test('returns draft unchanged when changed is empty', () => {
    const draft = { a: 1 };
    expect(slimRelationRefsForChanged(draft, [], refMeta)).toEqual(draft);
  });

  test('slims manyToOneRef value to Id only', () => {
    const draft = { partner: { Id: 42, DisplayName: 'Acme' } };
    const result = slimRelationRefsForChanged(draft, ['partner'], refMeta);
    expect(result.partner).toBe(42);
  });

  test('slims manyToManyRef array values to Id', () => {
    const draft = { tags: [{ Id: 1, DisplayName: 'A' }, { Id: 2 }] };
    const result = slimRelationRefsForChanged(draft, ['tags'], refMeta);
    expect(result.tags).toEqual([1, 2]);
  });

  test('filters null/undefined from manyToManyRef arrays', () => {
    const draft = { tags: [null, { Id: 3 }, undefined] };
    const result = slimRelationRefsForChanged(draft, ['tags'], refMeta);
    expect(result.tags).toEqual([3]);
  });

  test('does not slim non-ref relation values', () => {
    const draft = { name: 'hello' };
    const result = slimRelationRefsForChanged(draft, ['name'], refMeta);
    expect(result.name).toBe('hello');
  });
});

// --------------- detectStructuralChangedRelations ---------------

describe('detectStructuralChangedRelations', () => {
  const o2mStore = stubStore({
    Lines: { id: 'L', type: 'oneToMany', typeAnnotation: '' } as WebFieldMetadata,
    Items: { id: 'I', type: 'manyToMany', typeAnnotation: '' } as WebFieldMetadata,
    Name: { id: 'N', type: 'string', typeAnnotation: '' } as WebFieldMetadata,
  });

  test('returns empty when collapsed is empty', () => {
    const result = detectStructuralChangedRelations(new Set(), {}, {}, o2mStore);
    expect(result.size).toBe(0);
  });

  test('detects length change for relation arrays', () => {
    const base = { Lines: [{ Id: 1 }] };
    const cur = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const result = detectStructuralChangedRelations(new Set(['Lines']), base, cur, o2mStore);
    expect(result.has('Lines')).toBe(true);
  });

  test('detects Id sequence change', () => {
    const base = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const cur = { Lines: [{ Id: 2 }, { Id: 1 }] };
    const result = detectStructuralChangedRelations(new Set(['Lines']), base, cur, o2mStore);
    expect(result.has('Lines')).toBe(true);
  });

  test('does not flag when sequences are identical', () => {
    const base = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const cur = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const result = detectStructuralChangedRelations(new Set(['Lines']), base, cur, o2mStore);
    expect(result.has('Lines')).toBe(false);
  });

  test('skips non-relation string fields', () => {
    const result = detectStructuralChangedRelations(new Set(['Name']), { Name: 'a' }, { Name: 'b' }, o2mStore);
    expect(result.has('Name')).toBe(false);
  });
});
