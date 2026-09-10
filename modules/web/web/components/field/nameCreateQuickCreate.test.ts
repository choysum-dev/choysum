// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  extractNameCreateRecordId,
  formatNameCreateError,
  runNameCreateQuickCreate,
  trimSearchKeyword,
} from './nameCreateQuickCreate';

import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

describe('trimSearchKeyword', () => {
  test('trims and nullish-coalesces', () => {
    expect(trimSearchKeyword('  a  ')).toBe('a');
    expect(trimSearchKeyword(null)).toBe('');
    expect(trimSearchKeyword(undefined)).toBe('');
    expect(trimSearchKeyword('')).toBe('');
  });
});

describe('extractNameCreateRecordId', () => {
  test('reads Id or id and rejects empty', () => {
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
  test('prefers message, then error, then fallback', () => {
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
  test('guards busy, missing store, and empty keyword', async () => {
    const onError = asyncFnRecorder();
    const onSuccess = asyncFnRecorder();
    const NameCreate = asyncFnRecorder();

    expect(
      await runNameCreateQuickCreate({
        busy: { value: true },
        store: { NameCreate } as any,
        keyword: 'x',
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(NameCreate.calls.length).toBe(0);

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
    expect(onError.calls).toEqual([['fail']]);

    onError.calls.length = 0;
    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: { NameCreate } as any,
        keyword: '   ',
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(NameCreate.calls.length).toBe(0);
    expect(onError.calls.length).toBe(0);

    // Cover keyword ?? '' when keyword is null/undefined.
    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: { NameCreate } as any,
        keyword: null as any,
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
    expect(
      await runNameCreateQuickCreate({
        busy: { value: false },
        store: { NameCreate } as any,
        keyword: undefined as any,
        failedMessage: 'fail',
        onError,
        onSuccess,
      })
    ).toBe(false);
  });

  test('creates, passes nameField, and clears busy', async () => {
    const busy = { value: false };
    const onError = asyncFnRecorder();
    const onSuccess = asyncFnRecorder();
    const NameCreate = asyncFnRecorder(async () => ({ Id: 'n1', Name: 'Acme' }));

    const ok = await runNameCreateQuickCreate({
      busy,
      store: { NameCreate } as any,
      keyword: '  Acme  ',
      nameField: 'Code',
      failedMessage: 'fail',
      onError,
      onSuccess,
    });

    expect(ok).toBe(true);
    expect(NameCreate.calls).toEqual([['Acme', undefined, { nameField: 'Code' }]]);
    expect(onSuccess.calls).toEqual([[{ Id: 'n1', Name: 'Acme' }, 'n1']]);
    expect(busy.value).toBe(false);
  });

  test('omits options when nameField is unset', async () => {
    const NameCreate = asyncFnRecorder(async () => ({ id: 'legacy' }));
    await runNameCreateQuickCreate({
      busy: { value: false },
      store: { NameCreate } as any,
      keyword: 'x',
      failedMessage: 'fail',
      onError: asyncFnRecorder(),
      onSuccess: asyncFnRecorder(),
    });
    expect(NameCreate.calls).toEqual([['x', undefined, undefined]]);
  });

  test('errors when created row has no id', async () => {
    const onError = asyncFnRecorder();
    const busy = { value: false };
    const ok = await runNameCreateQuickCreate({
      busy,
      store: { NameCreate: asyncFnRecorder(async () => ({ Name: 'no-id' })) },
      keyword: 'x',
      failedMessage: 'fail',
      onError,
      onSuccess: asyncFnRecorder(),
    });
    expect(ok).toBe(false);
    expect(onError.calls).toEqual([['fail']]);
    expect(busy.value).toBe(false);
  });

  test('surfaces NameCreate throw via onError', async () => {
    const onError = asyncFnRecorder();
    const busy = { value: false };
    const ok = await runNameCreateQuickCreate({
      busy,
      store: {
        NameCreate: asyncFnRecorder(async () => {
          throw new Error('denied');
        }),
      },
      keyword: 'x',
      failedMessage: 'fail',
      onError,
      onSuccess: asyncFnRecorder(),
    });
    expect(ok).toBe(false);
    expect(onError.calls).toEqual([['denied']]);
    expect(busy.value).toBe(false);
  });
});
