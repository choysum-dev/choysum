// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { coerceToBucketStart, enumerateBuckets, nextBucket } from '../time_bucket_runtime';

test('repository time bucket runtime aligns utc boundaries for quarter and iso week', () => {
  expect(coerceToBucketStart('2026-05-17T18:30:00.000Z', 'quarter').toISOString()).toBe('2026-04-01T00:00:00.000Z');
  expect(coerceToBucketStart('2026-05-17T18:30:00.000Z', 'week').toISOString()).toBe('2026-05-11T00:00:00.000Z');
});

test('repository time bucket runtime falls back to utc alignment when Intl is unavailable', () => {
  const originalIntl = (globalThis as any).Intl;
  try {
    delete (globalThis as any).Intl;
    expect(coerceToBucketStart('2026-05-17T23:30:00.000Z', 'day', 'Asia/Shanghai').toISOString()).toBe('2026-05-17T00:00:00.000Z');
  } finally {
    (globalThis as any).Intl = originalIntl;
  }
});

test('repository time bucket runtime uses timezone parts when Intl is available and enumerates bounded sequences', () => {
  const originalIntl = (globalThis as any).Intl;
  try {
    (globalThis as any).Intl = {
      DateTimeFormat: function () {
        return {
          formatToParts() {
            return [
              { type: 'year', value: '2026' },
              { type: 'month', value: '05' },
              { type: 'day', value: '18' },
              { type: 'weekday', value: 'Monday' },
            ];
          },
        };
      },
    };

    const start = coerceToBucketStart('2026-05-17T23:30:00.000Z', 'day', 'Asia/Shanghai');
    expect(start.toISOString()).toBe('2026-05-18T00:00:00.000Z');

    const sequence = enumerateBuckets({
      start: '2026-05-17T23:30:00.000Z',
      end: '2026-07-20T00:00:00.000Z',
      granularity: 'month',
      maxBuckets: 2,
    }).map(item => item.toISOString());

    expect(sequence).toEqual(['2026-05-01T00:00:00.000Z', '2026-06-01T00:00:00.000Z']);
    expect(nextBucket(new Date('2026-05-01T00:00:00.000Z'), 'month').toISOString()).toBe('2026-06-01T00:00:00.000Z');
  } finally {
    (globalThis as any).Intl = originalIntl;
  }
});

test('repository time bucket runtime covers timezone switch variants, default fallback and weekday fallback mapping', () => {
  const originalIntl = (globalThis as any).Intl;
  try {
    (globalThis as any).Intl = {
      DateTimeFormat: function () {
        return {
          formatToParts() {
            return [
              { type: 'year', value: '2026' },
              { type: 'month', value: '11' },
              { type: 'day', value: '09' },
              { type: 'weekday', value: 'UnknownDay' },
            ];
          },
        };
      },
    };

    expect(coerceToBucketStart('2026-11-21T08:00:00.000Z', 'year', 'Asia/Shanghai').toISOString()).toBe('2026-01-01T00:00:00.000Z');
    expect(coerceToBucketStart('2026-11-21T08:00:00.000Z', 'quarter', 'Asia/Shanghai').toISOString()).toBe('2026-10-01T00:00:00.000Z');
    expect(coerceToBucketStart('2026-11-21T08:00:00.000Z', 'week', 'Asia/Shanghai').toISOString()).toBe('2026-11-09T00:00:00.000Z');
    expect(coerceToBucketStart('2026-11-21T08:00:00.000Z', 'noop' as any, 'Asia/Shanghai').toISOString()).toBe('2026-11-21T08:00:00.000Z');
  } finally {
    (globalThis as any).Intl = originalIntl;
  }
});

test('repository time bucket runtime covers utc and next-bucket year/day/default branches', () => {
  expect(coerceToBucketStart('2026-01-04T10:00:00.000Z', 'year').toISOString()).toBe('2026-01-01T00:00:00.000Z');
  expect(coerceToBucketStart('2026-01-04T10:00:00.000Z', 'noop' as any).toISOString()).toBe('2026-01-04T10:00:00.000Z');

  const sunday = coerceToBucketStart('2026-01-04T10:00:00.000Z', 'week');
  expect(sunday.toISOString()).toBe('2025-12-29T00:00:00.000Z');

  const seed = new Date('2026-01-01T00:00:00.000Z');
  expect(nextBucket(seed, 'year').toISOString()).toBe('2027-01-01T00:00:00.000Z');
  expect(nextBucket(seed, 'quarter').toISOString()).toBe('2026-04-01T00:00:00.000Z');
  expect(nextBucket(seed, 'week').toISOString()).toBe('2026-01-08T00:00:00.000Z');
  expect(nextBucket(seed, 'day').toISOString()).toBe('2026-01-02T00:00:00.000Z');
  expect(nextBucket(seed, 'noop' as any).toISOString()).toBe('2026-01-01T00:00:00.000Z');
});

test('repository time bucket runtime falls back when timezone formatter construction throws', () => {
  const originalIntl = (globalThis as any).Intl;
  try {
    (globalThis as any).Intl = {
      DateTimeFormat: function () {
        throw new Error('invalid timezone');
      },
    };

    expect(coerceToBucketStart('2026-05-17T23:30:00.000Z', 'day', 'Bad/Timezone').toISOString()).toBe('2026-05-17T00:00:00.000Z');
  } finally {
    (globalThis as any).Intl = originalIntl;
  }
});

test('repository time bucket runtime covers timezone month bucket and enumerate default max-buckets path', () => {
  const originalIntl = (globalThis as any).Intl;
  try {
    (globalThis as any).Intl = {
      DateTimeFormat: function () {
        return {
          formatToParts() {
            return [
              { type: 'year', value: '2026' },
              { type: 'month', value: '02' },
              { type: 'day', value: '17' },
              { type: 'weekday', value: 'Tuesday' },
            ];
          },
        };
      },
    };

    expect(coerceToBucketStart('2026-02-17T18:00:00.000Z', 'month', 'Asia/Shanghai').toISOString()).toBe('2026-02-01T00:00:00.000Z');
  } finally {
    (globalThis as any).Intl = originalIntl;
  }

  const sequence = enumerateBuckets({
    start: '2026-01-01T00:00:00.000Z',
    end: '2026-01-03T00:00:00.000Z',
    granularity: 'day',
  }).map(item => item.toISOString());
  expect(sequence).toEqual(['2026-01-01T00:00:00.000Z', '2026-01-02T00:00:00.000Z', '2026-01-03T00:00:00.000Z']);
});

test('repository time bucket runtime week alignment covers non-sunday utc branch', () => {
  expect(coerceToBucketStart('2026-01-05T10:00:00.000Z', 'week').toISOString()).toBe('2026-01-05T00:00:00.000Z');
});
