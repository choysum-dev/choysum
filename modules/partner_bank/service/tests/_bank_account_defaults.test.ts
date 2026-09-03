// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { ACCOUNT_TYPES, maskAccountNo, assertAccountType, pickDefaultBankAccountId } from '@/partner_bank/service/models/_bank_account_defaults';

// ---------------------------------------------------------------------------
// maskAccountNo
// ---------------------------------------------------------------------------

test('partner_bank._bank_account_defaults: maskAccountNo returns nulls for empty string', () => {
  const result = maskAccountNo('');
  expect(result.last4).toBe(null);
  expect(result.masked).toBe(null);
});

test('partner_bank._bank_account_defaults: maskAccountNo returns nulls for whitespace-only', () => {
  const result = maskAccountNo('   ');
  expect(result.last4).toBe(null);
  expect(result.masked).toBe(null);
});

test('partner_bank._bank_account_defaults: maskAccountNo handles short account number (< 4 chars)', () => {
  const result = maskAccountNo('12');
  expect(result.last4).toBe('12');
  expect(result.masked).toBe('12');
});

test('partner_bank._bank_account_defaults: maskAccountNo handles exactly 4 chars', () => {
  const result = maskAccountNo('1234');
  expect(result.last4).toBe('1234');
  expect(result.masked).toBe('1234');
});

test('partner_bank._bank_account_defaults: maskAccountNo masks long account number', () => {
  const result = maskAccountNo('1234567890123456');
  expect(result.last4).toBe('3456');
  // 16 chars total, visible 4 = 12 hidden, capped at 8
  expect(result.masked).toBe('********3456');
});

test('partner_bank._bank_account_defaults: maskAccountNo handles account number with spaces', () => {
  const result = maskAccountNo('1234 5678 9012 3456');
  expect(result.last4).toBe('3456');
  expect(result.masked).toBe('********3456');
});

// ---------------------------------------------------------------------------
// assertAccountType
// ---------------------------------------------------------------------------

test('partner_bank._bank_account_defaults: assertAccountType returns valid type', () => {
  expect(assertAccountType('checking')).toBe('checking');
  expect(assertAccountType('savings')).toBe('savings');
  expect(assertAccountType('corporate')).toBe('corporate');
  expect(assertAccountType('other')).toBe('other');
});

test('partner_bank._bank_account_defaults: assertAccountType trims value', () => {
  expect(assertAccountType('  checking  ')).toBe('checking');
});

test('partner_bank._bank_account_defaults: assertAccountType rejects case-mismatched input', () => {
  let err: unknown;
  try {
    assertAccountType('Checking');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
});

test('partner_bank._bank_account_defaults: assertAccountType returns undefined for undefined', () => {
  expect(assertAccountType(undefined)).toBe(undefined);
});

test('partner_bank._bank_account_defaults: assertAccountType returns null for null', () => {
  expect(assertAccountType(null)).toBe(null);
});

test('partner_bank._bank_account_defaults: assertAccountType returns null for empty', () => {
  expect(assertAccountType('')).toBe(null);
});

test('partner_bank._bank_account_defaults: assertAccountType throws for invalid type', () => {
  let err: unknown;
  try {
    assertAccountType('invalid');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('AccountType must be one of checking, savings, corporate, other');
});

test('partner_bank._bank_account_defaults: ACCOUNT_TYPES contains the four expected values', () => {
  expect(ACCOUNT_TYPES.has('checking')).toBe(true);
  expect(ACCOUNT_TYPES.has('savings')).toBe(true);
  expect(ACCOUNT_TYPES.has('corporate')).toBe(true);
  expect(ACCOUNT_TYPES.has('other')).toBe(true);
  expect(ACCOUNT_TYPES.has('invalid')).toBe(false);
});

// ---------------------------------------------------------------------------
// pickDefaultBankAccountId
// ---------------------------------------------------------------------------

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId returns null for undefined', () => {
  expect(pickDefaultBankAccountId(undefined, 'inbound')).toBe(null);
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId returns null for empty array', () => {
  expect(pickDefaultBankAccountId([], 'inbound')).toBe(null);
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId picks default inbound', () => {
  const accounts = [
    { Id: 'a', IsDefaultInbound: false },
    { Id: 'b', IsDefaultInbound: true },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe('b');
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId picks default outbound', () => {
  const accounts = [
    { Id: 'a', IsDefaultOutbound: false },
    { Id: 'b', IsDefaultOutbound: true },
  ];
  expect(pickDefaultBankAccountId(accounts, 'outbound')).toBe('b');
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId returns null when no default', () => {
  const accounts = [
    { Id: 'a', IsDefaultInbound: false },
    { Id: 'b', IsDefaultInbound: false },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe(null);
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId excludes inactive accounts', () => {
  const accounts = [
    { Id: 'a', IsDefaultInbound: true, IsActive: false },
    { Id: 'b', IsDefaultInbound: false, IsActive: true },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe(null);
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId excludes rows without Id', () => {
  const accounts = [
    { Id: '', IsDefaultInbound: true },
    { Id: 'b', IsDefaultInbound: false },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe(null);
});

test('partner_bank._bank_account_defaults: pickDefaultBankAccountId sorts deterministically by Id', () => {
  const accounts = [
    { Id: 'c', IsDefaultInbound: true },
    { Id: 'a', IsDefaultInbound: true },
    { Id: 'b', IsDefaultInbound: false },
  ];
  // Both 'a' and 'c' are defaults; after sorting by Id, 'a' comes first
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe('a');
});
