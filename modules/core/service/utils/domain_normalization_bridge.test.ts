// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';
import { createDomainNormalizationBridge } from '@/core/service/utils/domain_normalization_bridge';
import { NormalizationError } from '@/core/service/utils/normalization';

const { _t } = createTranslate('core');
const bridge = createDomainNormalizationBridge('core', _t);

test('createDomainNormalizationBridge fail raises domain InvalidArgument', () => {
  let err: unknown;
  try {
    bridge.fail('bad input');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  const ce = err as ChoysumError;
  expect(ce.domain).toBe('core');
  expect(ce.code).toBe('InvalidArgument');
  expect(ce.message).toBe('bad input');
});

test('createDomainNormalizationBridge mapNormalizationError maps NormalizationError', () => {
  let err: unknown;
  try {
    bridge.mapNormalizationError(
      () => {
        throw new NormalizationError('required');
      },
      () => 'Mapped'
    );
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Mapped');
});

test('createDomainNormalizationBridge mapNormalizationError passes through other errors', () => {
  expect(() =>
    bridge.mapNormalizationError(
      () => {
        throw new Error('boom');
      },
      () => 'Mapped'
    )
  ).toThrow('boom');
});

test('createDomainNormalizationBridge normalizeRequiredText uses field name', () => {
  expect(bridge.normalizeRequiredText('  ok  ', 'Name')).toBe('ok');
  let err: unknown;
  try {
    bridge.normalizeRequiredText('', 'Name');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Name is required');
});

test('createDomainNormalizationBridge covers translated helpers', () => {
  expect(bridge.translatedTextHasValue({ en_US: 'x' })).toBe(true);
  expect(bridge.normalizeOptionalTranslatedText({ en_US: ' A ' })).toEqual({ en_US: 'A' });

  let err: unknown;
  try {
    bridge.normalizeRequiredTranslatedText({ en_US: '  ', zh_CN: '' }, 'Name');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('Name is required');
  expect(bridge.normalizeRequiredTranslatedText({ en_US: 'Hello' }, 'Name')).toEqual({ en_US: 'Hello' });
});
