// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  resolveEffectiveModel,
  normalizePriorityRange,
  priorityInRange,
  matchesMethodPrefix,
} from '@/core/service/orm/metadata/effective_query_helper';

// resolveEffectiveModel

test('resolveEffectiveModel throws on empty identifier', () => {
  expect(() => resolveEffectiveModel('')).toThrow('modelIdentifier cannot be empty');
  expect(() => resolveEffectiveModel('   ')).toThrow('modelIdentifier cannot be empty');
});

test('resolveEffectiveModel throws on unknown model identifier', () => {
  expect(() => resolveEffectiveModel('__completely_unknown_model__')).toThrow('model not found');
});

// normalizePriorityRange

test('normalizePriorityRange returns undefined bounds when options is omitted', () => {
  expect(normalizePriorityRange()).toEqual({ min: undefined, max: undefined });
});

test('normalizePriorityRange returns undefined bounds for empty options', () => {
  expect(normalizePriorityRange({})).toEqual({ min: undefined, max: undefined });
});

test('normalizePriorityRange returns numeric bounds for valid finite inputs', () => {
  expect(normalizePriorityRange({ minPriority: 5, maxPriority: 10 })).toEqual({ min: 5, max: 10 });
});

test('normalizePriorityRange treats NaN/Infinity as undefined', () => {
  expect(normalizePriorityRange({ minPriority: NaN, maxPriority: Infinity })).toEqual({ min: undefined, max: undefined });
});

test('normalizePriorityRange returns only min when max is omitted', () => {
  expect(normalizePriorityRange({ minPriority: 3 })).toEqual({ min: 3, max: undefined });
});

test('normalizePriorityRange returns only max when min is omitted', () => {
  expect(normalizePriorityRange({ maxPriority: 7 })).toEqual({ max: 7, min: undefined });
});

// priorityInRange

test('priorityInRange returns true when range bounds are undefined', () => {
  expect(priorityInRange({ priority: 5 }, { min: undefined, max: undefined })).toBe(true);
});

test('priorityInRange returns true when priority is within range', () => {
  expect(priorityInRange({ priority: 5 }, { min: 1, max: 10 })).toBe(true);
});

test('priorityInRange returns true at exact boundaries', () => {
  expect(priorityInRange({ priority: 0 }, { min: 0, max: 0 })).toBe(true);
});

test('priorityInRange returns false when priority is below min', () => {
  expect(priorityInRange({ priority: 2 }, { min: 5 })).toBe(false);
});

test('priorityInRange returns false when priority is above max', () => {
  expect(priorityInRange({ priority: 10 }, { max: 5 })).toBe(false);
});

test('priorityInRange defaults missing priority to 0', () => {
  expect(priorityInRange({}, { min: -1, max: 1 })).toBe(true);
  expect(priorityInRange({}, { min: 1 })).toBe(false);
});

test('priorityInRange defaults NaN priority to 0', () => {
  expect(priorityInRange({ priority: NaN }, { min: -1 })).toBe(true);
});

// matchesMethodPrefix

test('matchesMethodPrefix returns true for empty/whitespace prefix', () => {
  expect(matchesMethodPrefix({ method: 'anything' }, '')).toBe(true);
  expect(matchesMethodPrefix({ method: 'anything' }, '   ')).toBe(true);
});

test('matchesMethodPrefix returns true when method starts with prefix (case-insensitive)', () => {
  expect(matchesMethodPrefix({ method: 'GetEffective' }, 'get')).toBe(true);
  expect(matchesMethodPrefix({ method: 'geteffective' }, 'GET')).toBe(true);
  expect(matchesMethodPrefix({ method: 'GETEFFECTIVE' }, 'get')).toBe(true);
});

test('matchesMethodPrefix returns false when method does not start with prefix', () => {
  expect(matchesMethodPrefix({ method: 'otherMethod' }, 'get')).toBe(false);
});

test('matchesMethodPrefix handles missing method gracefully', () => {
  expect(matchesMethodPrefix({}, 'get')).toBe(false);
});

test('matchesMethodPrefix trims whitespace from prefix before matching', () => {
  expect(matchesMethodPrefix({ method: 'getEffective' }, '  get  ')).toBe(true);
});
