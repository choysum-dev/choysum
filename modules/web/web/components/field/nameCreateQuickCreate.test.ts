// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import {
  extractNameCreateRecordId,
  formatNameCreateError,
  runNameCreateQuickCreate,
  trimSearchKeyword,
} from './nameCreateQuickCreate';

describe('trimSearchKeyword', () => {
  it('trims and nullish-coalesces', () => {
    expect(trimSearchKeyword('  a  ')).toBe('a');
    expect(trimSearchKeyword(null)).toBe('');
    expect(trimSearchKeyword(undefined)).toBe('');
    expect(trimSearchKeyword('')).toBe('');
  });
});

describe('extractNameCreateRecordId', () => {
  it('reads Id or id and rejects empty', () => {
    expect(extractNameCreateRecordId({ Id: 'a1' })).toBe('a1');
    expect(extractNameCreateRecordId({ id: 'b2' })).toBe('b2');
    expect(extractNameCreateRecordId({ Id: '  c3  ' })).toBe('c3');
    expect(extractNameCreateRecordId({ Id: '' })).toBeUndefined();
    expect(extractNameCreateRecordId({ Id: '   ' })).toBeUndefined();
    expect(extractNameCreateRecordId({ Id: null })).toBeUndefined();
    expect(extractNameCreateRecordId(null)).toBeUndefined();
    expect(extractNameCreateRecordId(undefined)).toBeUndefined();
    expect(extractNameCreateRecordId('x')).toBeUndefined();
  });
});

describe('formatNameCreateError', () => {
  it('prefers message, then error, then fallback', () => {
    expect(formatNameCreateError(new Error('boom'), 'fallback')).toBe('boom');
    expect(formatNameCreateError('raw', 'fallback')).toBe('raw');
    expect(formatNameCreateError(null, 'fallback')).toBe('fallback');
    expect(formatNameCreateError(undefined, 'fallback')).toBe('fallback');
    // Empty message falls through to String(error) for plain objects.
    expect(formatNameCreateError({ message: '' }, 'fallback')).toBe('[object Object]');
    expect(formatNameCreateError(0, 'fallback')).toBe('fallback');
  });
});

describe('runNameCreateQuickCreate', () => {
  it('guards busy, missing store, and empty keyword', async () => {
    const onError = vi.fn();
    const onSuccess = vi.fn();
    const NameCreate = vi.fn();

    expect(
      await runNameCreateQuickCreate({
        busy: { value: true },
        store: { NameCreate },
        keyword: 'x',
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(NameCreate).not.toHaveBeenCalled();

    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: null,
        keyword: 'x',
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(onError).toHaveBeenCalledWith('fail');

    onError.mockClear();
    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: { NameCreate },
        keyword: '   ',
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(NameCreate).not.toHaveBeenCalled();
    expect(onError).not.toHaveBeenCalled();

    // Cover keyword ?? '' when keyword is null/undefined.
    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: { NameCreate },
        keyword: null as any,
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: { NameCreate },
        keyword: undefined as any,
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
  });

  it('creates, passes nameField, and clears busy', async () => {
    const busy = { value: false };
    const onError = vi.fn();
    const onSuccess = vi.fn();
    const NameCreate = vi.fn(async () => ({ Id: 'n1', Name: 'Acme' }));

    const ok = await runNameCreateQuickCreate({
      busy,
      store: { NameCreate },
      keyword: '  Acme  ',
      nameField: 'Code',
      failedMessage: 'fail',
      onError,
      onSuccess,
    });

    expect(ok).toBe(true);
    expect(NameCreate).toHaveBeenCalledWith('Acme', undefined, { nameField: 'Code' });
    expect(onSuccess).toHaveBeenCalledWith({ Id: 'n1', Name: 'Acme' }, 'n1');
    expect(busy.value).toBe(false);
  });

  it('omits options when nameField is unset', async () => {
    const NameCreate = vi.fn(async () => ({ id: 'legacy' }));
    await runNameCreateQuickCreate({
      busy: { value: false },
      store: { NameCreate },
      keyword: 'x',
      failedMessage: 'fail',
      onError: vi.fn(),
      onSuccess: vi.fn(),
    });
    expect(NameCreate).toHaveBeenCalledWith('x', undefined, undefined);
  });

  it('errors when created row has no id', async () => {
    const onError = vi.fn();
    const busy = { value: false };
    const ok = await runNameCreateQuickCreate({
      busy,
      store: { NameCreate: vi.fn(async () => ({ Name: 'no-id' })) },
      keyword: 'x',
      failedMessage: 'fail',
      onError,
      onSuccess: vi.fn(),
    });
    expect(ok).toBe(false);
    expect(onError).toHaveBeenCalledWith('fail');
    expect(busy.value).toBe(false);
  });

  it('surfaces NameCreate throw via onError', async () => {
    const onError = vi.fn();
    const busy = { value: false };
    const ok = await runNameCreateQuickCreate({
      busy,
      store: {
        NameCreate: vi.fn(async () => {
          throw new Error('denied');
        }),
      },
      keyword: 'x',
      failedMessage: 'fail',
      onError,
      onSuccess: vi.fn(),
    });
    expect(ok).toBe(false);
    expect(onError).toHaveBeenCalledWith('denied');
    expect(busy.value).toBe(false);
  });
});
