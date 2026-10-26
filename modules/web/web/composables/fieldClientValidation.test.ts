// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { firstRuleError } from './fieldClientValidation';

describe('fieldClientValidation', () => {
  test('firstRuleError returns empty when no rules', async () => {
    expect(await firstRuleError(undefined, 'x')).toBe('');
    expect(await firstRuleError([], 'x')).toBe('');
  });

  test('firstRuleError honors required', async () => {
    expect(await firstRuleError([{ required: true, message: 'need it' }], '')).toBe('need it');
    expect(await firstRuleError([{ required: true }], null)).toBe('Required');
    expect(await firstRuleError([{ required: true, message: 'need it' }], 'ok')).toBe('');
    expect(await firstRuleError([{ required: true, message: 'pick some' }], [])).toBe('pick some');
    expect(await firstRuleError([{ required: true }], ['a'])).toBe('');
  });

  test('firstRuleError runs callback validators', async () => {
    const rules = [
      {
        validator: (_r: unknown, v: unknown, cb: (e?: Error) => void) => {
          if (v !== 'ok') cb(new Error('bad'));
          else cb();
        },
      },
    ];
    expect(await firstRuleError(rules as any, 'no')).toBe('bad');
    expect(await firstRuleError(rules as any, 'ok')).toBe('');
  });

  test('firstRuleError supports promise validators and throw', async () => {
    expect(
      await firstRuleError(
        [
          {
            validator: async () => {
              throw new Error('boom');
            },
          },
        ] as any,
        1
      )
    ).toBe('boom');
    expect(
      await firstRuleError(
        [
          {
            validator: async () => undefined,
          },
        ] as any,
        1
      )
    ).toBe('');
    expect(
      await firstRuleError(
        [
          {
            validator: (_r: unknown, _v: unknown, cb: (e?: string) => void) => {
              cb('string-err');
            },
          },
        ] as any,
        1
      )
    ).toBe('string-err');
    expect(
      await firstRuleError(
        [
          {
            validator: () => {
              throw 'sync-throw';
            },
          },
        ] as any,
        1
      )
    ).toBe('sync-throw');
  });
});
