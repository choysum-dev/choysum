// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { Model } from '../decorator';
import { lookupFieldDefaultModel, __setLookupFieldDefaultModelForTest } from './field_default_lookup';

@Model('FieldDefault', { application: 'lookuppool' })
class LookupPoolFieldDefault extends BaseModel {
  static async GetEffective() {
    return { Code: 'from-pool' };
  }
}

@Model('FieldDefault', { application: 'lookupnoge' })
class LookupNoGetEffectiveFieldDefault extends BaseModel {}

test('lookupFieldDefaultModel returns undefined for empty application', () => {
  expect(lookupFieldDefaultModel(undefined)).toBeUndefined();
  expect(lookupFieldDefaultModel('')).toBeUndefined();
  expect(lookupFieldDefaultModel('   ')).toBeUndefined();
});

test('__setLookupFieldDefaultModelForTest ignores empty application keys', () => {
  __setLookupFieldDefaultModelForTest('', {
    async GetEffective() {
      return { Code: 'should-not-stick' };
    },
  });
  __setLookupFieldDefaultModelForTest('   ', {
    async GetEffective() {
      return { Code: 'should-not-stick' };
    },
  });
  expect(lookupFieldDefaultModel('')).toBeUndefined();
});

test('lookupFieldDefaultModel resolves pool model with GetEffective', () => {
  __setLookupFieldDefaultModelForTest('lookuppool', undefined);
  const ctor = lookupFieldDefaultModel('lookuppool');
  expect(ctor).toBe(LookupPoolFieldDefault);
  expect(typeof ctor?.GetEffective).toBe('function');
});

test('lookupFieldDefaultModel rejects pool model without GetEffective', () => {
  __setLookupFieldDefaultModelForTest('lookupnoge', undefined);
  expect(lookupFieldDefaultModel('lookupnoge')).toBeUndefined();
  expect(LookupNoGetEffectiveFieldDefault).toBeTruthy();
});
