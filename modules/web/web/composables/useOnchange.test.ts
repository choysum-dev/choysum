// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
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
  it('maps manytoone -> ManyToOne', () => {
    expect(normalizeDiffRelation('manytoone')).toBe('ManyToOne');
  });
  it('maps manytooneref -> ManyToOne', () => {
    expect(normalizeDiffRelation('manytooneref')).toBe('ManyToOne');
  });
  it('maps onetomany -> OneToMany', () => {
    expect(normalizeDiffRelation('onetomany')).toBe('OneToMany');
  });
  it('maps manytomany -> ManyToMany', () => {
    expect(normalizeDiffRelation('manytomany')).toBe('ManyToMany');
  });
  it('maps manytomanyref -> ManyToMany', () => {
    expect(normalizeDiffRelation('manytomanyref')).toBe('ManyToMany');
  });
  it('returns undefined for unknown types', () => {
    expect(normalizeDiffRelation('string')).toBeUndefined();
  });
  it('returns undefined for undefined input', () => {
    expect(normalizeDiffRelation(undefined)).toBeUndefined();
  });
  it('is case-insensitive', () => {
    expect(normalizeDiffRelation('ManyToOne')).toBe('ManyToOne');
  });
});

// --------------- looksLikeRelation ---------------

describe('looksLikeRelation', () => {
  it('returns false for undefined metadata', () => {
    expect(looksLikeRelation(undefined)).toBe(false);
  });
  it('returns true for manytoone', () => {
    expect(looksLikeRelation({ id: '1', type: 'manytoone', typeAnnotation: '' })).toBe(true);
  });
  it('returns true for onetomany', () => {
    expect(looksLikeRelation({ id: '2', type: 'onetomany', typeAnnotation: '' })).toBe(true);
  });
  it('returns false for string', () => {
    expect(looksLikeRelation({ id: '3', type: 'string', typeAnnotation: '' })).toBe(false);
  });
});

// --------------- buildDiffFieldsMeta ---------------

describe('buildDiffFieldsMeta', () => {
  it('returns empty object for store without fieldsMetadata', () => {
    const s = stubStore();
    expect(buildDiffFieldsMeta(s)).toEqual({});
  });

  it('returns normalized metadata for relation fields', () => {
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

  it('caches result and returns same normalized object on repeated calls', () => {
    const s = stubStore({ x: { id: 'a', type: 'integer', typeAnnotation: 'number' } as WebFieldMetadata });
    const a = buildDiffFieldsMeta(s);
    const b = buildDiffFieldsMeta(s);
    // Same identity when source reference unchanged
    expect(a).toBe(b);
  });
});

// --------------- buildRelationFieldSet ---------------

describe('buildRelationFieldSet', () => {
  it('returns empty set for store without fields', () => {
    expect(buildRelationFieldSet(stubStore()).size).toBe(0);
  });

  it('collects only relation field names', () => {
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
  it('returns true for digits-only', () => {
    expect(isIndexSeg('0')).toBe(true);
    expect(isIndexSeg('42')).toBe(true);
  });
  it('returns false for non-digits', () => {
    expect(isIndexSeg('abc')).toBe(false);
    expect(isIndexSeg('a1')).toBe(false);
    expect(isIndexSeg('')).toBe(false);
  });
});

// --------------- normalizeArrayIndexInPath ---------------

describe('normalizeArrayIndexInPath', () => {
  it('replaces bracket indices with dot indices', () => {
    expect(normalizeArrayIndexInPath('Lines[0].Qty')).toBe('Lines.0.Qty');
  });
  it('handles multiple bracket indices', () => {
    expect(normalizeArrayIndexInPath('Lines[0].Batches[1].Qty')).toBe('Lines.0.Batches.1.Qty');
  });
  it('returns same path if no brackets', () => {
    expect(normalizeArrayIndexInPath('a.b.c')).toBe('a.b.c');
  });
  it('returns empty string for falsy input', () => {
    expect(normalizeArrayIndexInPath('')).toBe('');
  });
});

// --------------- extractBaseRoot ---------------

describe('extractBaseRoot', () => {
  it('extracts first identifier segment', () => {
    expect(extractBaseRoot('Lines.0.Qty')).toBe('Lines');
  });
  it('extracts root even with bracket syntax', () => {
    expect(extractBaseRoot('Lines(id=1).Qty')).toBe('Lines');
  });
  it('returns single segment', () => {
    expect(extractBaseRoot('Name')).toBe('Name');
  });
  it('handles empty string', () => {
    expect(extractBaseRoot('')).toBe('');
  });
});

// --------------- toOneHopFieldSignal ---------------

describe('toOneHopFieldSignal', () => {
  it('strips indices from leaf paths', () => {
    expect(toOneHopFieldSignal('Lines.1.UnitPrice')).toBe('Lines.UnitPrice');
  });
  it('strips nested indices', () => {
    expect(toOneHopFieldSignal('Lines.1.Batches.0.Qty')).toBe('Lines.Batches.Qty');
  });
  it('returns null for paths without dots', () => {
    expect(toOneHopFieldSignal('Name')).toBeNull();
  });
  it('returns null for empty input', () => {
    expect(toOneHopFieldSignal('')).toBeNull();
  });
});

// --------------- collectAncestorCollectionRoots ---------------

describe('collectAncestorCollectionRoots', () => {
  it('returns Lines for Lines.1.UnitPrice', () => {
    expect(collectAncestorCollectionRoots('Lines.1.UnitPrice')).toEqual(['Lines']);
  });
  it('returns both ancestor roots for nested collections', () => {
    expect(collectAncestorCollectionRoots('Lines.1.Batches.0.Qty')).toEqual(['Lines', 'Lines.Batches']);
  });
  it('returns empty for empty input', () => {
    expect(collectAncestorCollectionRoots('')).toEqual([]);
  });
  it('returns empty for leaf without indices', () => {
    expect(collectAncestorCollectionRoots('a.b.c')).toEqual([]);
  });
});

// --------------- getArrayAtPath ---------------

describe('getArrayAtPath', () => {
  it('returns array at given field path', () => {
    const root = { Lines: [{ Id: 1 }, { Id: 2 }] };
    expect(getArrayAtPath(root, 'Lines')).toEqual([{ Id: 1 }, { Id: 2 }]);
  });
  it('returns null for non-array field', () => {
    expect(getArrayAtPath({ x: 'string' }, 'x')).toBeNull();
  });
  it('returns null for missing path', () => {
    expect(getArrayAtPath({}, 'a.b')).toBeNull();
  });
  it('traverses nested objects', () => {
    const root = { a: { b: [{ Id: 1 }] } };
    expect(getArrayAtPath(root, 'a.b')).toEqual([{ Id: 1 }]);
  });
});

// --------------- findIndexById ---------------

describe('findIndexById', () => {
  const arr = [{ Id: 10 }, { Id: 20 }, { id: 30 }];

  it('finds by Id', () => {
    expect(findIndexById(arr, 10)).toBe(0);
  });
  it('finds by id (lowercase)', () => {
    expect(findIndexById(arr, 30)).toBe(2);
  });
  it('returns -1 when not found', () => {
    expect(findIndexById(arr, 99)).toBe(-1);
  });
  it('returns -1 for non-array', () => {
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

  it('finds at first level', () => {
    const hit = deepFindById(root, ['Lines'], 2);
    expect(hit?.idx).toBe(1);
  });

  it('finds at nested level', () => {
    const hit = deepFindById(root, ['Lines', 'Batches'], 20);
    expect(hit?.idx).toBe(0);
  });

  it('returns null when not found', () => {
    expect(deepFindById(root, ['Lines', 'Batches'], 99)).toBeNull();
  });

  it('returns null for empty segments', () => {
    expect(deepFindById(root, [], 1)).toBeNull();
  });
});

// --------------- applyRowPatchToArray ---------------

describe('applyRowPatchToArray', () => {
  it('applies patch values excluding meta keys', () => {
    const arr = [{ Id: 1, Qty: 10, Price: 5 }];
    applyRowPatchToArray(arr, 0, { Qty: 20, Price: 8, Id: 99, pos: 3 });
    expect(arr[0].Qty).toBe(20);
    expect(arr[0].Price).toBe(8);
    // meta keys skipped
    expect(arr[0].Id).toBe(1);
  });

  it('does nothing for out-of-range index', () => {
    const arr = [{ Id: 1 }];
    applyRowPatchToArray(arr, 99, { Qty: 5 });
    expect(arr[0].Qty).toBeUndefined();
  });

  it('does nothing for non-array', () => {
    applyRowPatchToArray(null as any, 0, { x: 1 });
    // no throw
  });
});

// --------------- augmentCollapsedWithRelationRoots ---------------

describe('augmentCollapsedWithRelationRoots', () => {
  it('adds relation roots from full leaf paths into target set', () => {
    const target = new Set<string>(['Name']);
    const fullLeaves = new Set<string>(['Lines.0.Qty', 'Lines.0.Price', 'Tags.1.Label']);
    augmentCollapsedWithRelationRoots(target, fullLeaves);
    expect(target.has('Lines')).toBe(true);
    expect(target.has('Tags')).toBe(true);
    expect(target.has('Name')).toBe(true);
  });

  it('no-ops when fullLeafPaths is empty', () => {
    const target = new Set<string>(['A']);
    augmentCollapsedWithRelationRoots(target, new Set());
    expect(target.size).toBe(1);
  });

  it('no-ops for leaf paths without dots', () => {
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

  it('converts index segment to id-based selector', () => {
    expect(toSelectorPath(root, 'Lines.0.UnitPrice')).toBe('Lines(id=100).UnitPrice');
  });

  it('falls back to bracket for missing row', () => {
    expect(toSelectorPath(root, 'Lines.99.UnitPrice')).toBe('Lines[99].UnitPrice');
  });

  it('uses lowercase id', () => {
    expect(toSelectorPath(root, 'Lines2.0.Qty')).toBe('Lines2(id=abc).Qty');
  });

  it('returns null for flat path', () => {
    expect(toSelectorPath(root, 'Name')).toBeNull();
  });

  it('returns null for empty input', () => {
    expect(toSelectorPath(root, '')).toBeNull();
  });
});

// --------------- relationAwareMinimize ---------------

describe('relationAwareMinimize', () => {
  const relStore = stubStore({
    Lines: { id: 'L', type: 'oneToMany', typeAnnotation: '' } as WebFieldMetadata,
  });

  it('returns empty for empty paths', () => {
    expect(relationAwareMinimize([], relStore, true)).toEqual([]);
  });

  it('collapses relation children to root when collapse=true', () => {
    const paths = ['Lines.0.Qty', 'Lines.0.Price', 'Name'];
    const result = relationAwareMinimize(paths, relStore, true);
    expect(result).toContain('Lines');
    expect(result).toContain('Name');
    expect(result).not.toContain('Lines.0.Qty');
  });

  it('keeps all paths when collapse=false', () => {
    const paths = ['Lines.0.Qty', 'Name'];
    const result = relationAwareMinimize(paths, relStore, false);
    expect(result).toEqual(['Lines.0.Qty', 'Name']);
  });

  it('keeps parent path when child is a prefix extension', () => {
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

  it('returns draft unchanged when changed is empty', () => {
    const draft = { a: 1 };
    expect(slimRelationRefsForChanged(draft, [], refMeta)).toEqual(draft);
  });

  it('slims manyToOneRef value to Id only', () => {
    const draft = { partner: { Id: 42, DisplayName: 'Acme' } };
    const result = slimRelationRefsForChanged(draft, ['partner'], refMeta);
    expect(result.partner).toBe(42);
  });

  it('slims manyToManyRef array values to Id', () => {
    const draft = { tags: [{ Id: 1, DisplayName: 'A' }, { Id: 2 }] };
    const result = slimRelationRefsForChanged(draft, ['tags'], refMeta);
    expect(result.tags).toEqual([1, 2]);
  });

  it('filters null/undefined from manyToManyRef arrays', () => {
    const draft = { tags: [null, { Id: 3 }, undefined] };
    const result = slimRelationRefsForChanged(draft, ['tags'], refMeta);
    expect(result.tags).toEqual([3]);
  });

  it('does not slim non-ref relation values', () => {
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

  it('returns empty when collapsed is empty', () => {
    const result = detectStructuralChangedRelations(new Set(), {}, {}, o2mStore);
    expect(result.size).toBe(0);
  });

  it('detects length change for relation arrays', () => {
    const base = { Lines: [{ Id: 1 }] };
    const cur = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const result = detectStructuralChangedRelations(new Set(['Lines']), base, cur, o2mStore);
    expect(result.has('Lines')).toBe(true);
  });

  it('detects Id sequence change', () => {
    const base = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const cur = { Lines: [{ Id: 2 }, { Id: 1 }] };
    const result = detectStructuralChangedRelations(new Set(['Lines']), base, cur, o2mStore);
    expect(result.has('Lines')).toBe(true);
  });

  it('does not flag when sequences are identical', () => {
    const base = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const cur = { Lines: [{ Id: 1 }, { Id: 2 }] };
    const result = detectStructuralChangedRelations(new Set(['Lines']), base, cur, o2mStore);
    expect(result.has('Lines')).toBe(false);
  });

  it('skips non-relation string fields', () => {
    const result = detectStructuralChangedRelations(new Set(['Name']), { Name: 'a' }, { Name: 'b' }, o2mStore);
    expect(result.has('Name')).toBe(false);
  });
});
