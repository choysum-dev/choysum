// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { ACCOUNT_TYPES, maskAccountNo, normalizeAccountType, pickDefaultBankAccountId } from '@/partner_bank/service/models/_helpers';

// ---------------------------------------------------------------------------
// maskAccountNo
// ---------------------------------------------------------------------------

test('partner_bank._helpers: maskAccountNo returns nulls for empty string', () => {
  const result = maskAccountNo('');
  expect(result.last4).toBe(null);
  expect(result.masked).toBe(null);
});

test('partner_bank._helpers: maskAccountNo returns nulls for whitespace-only', () => {
  const result = maskAccountNo('   ');
  expect(result.last4).toBe(null);
  expect(result.masked).toBe(null);
});

test('partner_bank._helpers: maskAccountNo handles short account number (< 4 chars)', () => {
  const result = maskAccountNo('12');
  expect(result.last4).toBe('12');
  expect(result.masked).toBe('12');
});

test('partner_bank._helpers: maskAccountNo handles exactly 4 chars', () => {
  const result = maskAccountNo('1234');
  expect(result.last4).toBe('1234');
  expect(result.masked).toBe('1234');
});

test('partner_bank._helpers: maskAccountNo masks long account number', () => {
  const result = maskAccountNo('1234567890123456');
  expect(result.last4).toBe('3456');
  // 16 chars total, visible 4 = 12 hidden, capped at 8
  expect(result.masked).toBe('********3456');
});

test('partner_bank._helpers: maskAccountNo handles account number with spaces', () => {
  const result = maskAccountNo('1234 5678 9012 3456');
  expect(result.last4).toBe('3456');
  expect(result.masked).toBe('********3456');
});

// ---------------------------------------------------------------------------
// normalizeAccountType
// ---------------------------------------------------------------------------

test('partner_bank._helpers: normalizeAccountType returns valid type', () => {
  expect(normalizeAccountType('checking')).toBe('checking');
  expect(normalizeAccountType('savings')).toBe('savings');
  expect(normalizeAccountType('corporate')).toBe('corporate');
  expect(normalizeAccountType('other')).toBe('other');
});

test('partner_bank._helpers: normalizeAccountType trims value', () => {
  expect(normalizeAccountType('  checking  ')).toBe('checking');
});

test('partner_bank._helpers: normalizeAccountType rejects case-mismatched input', () => {
  let err: unknown;
  try {
    normalizeAccountType('Checking');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
});

test('partner_bank._helpers: normalizeAccountType returns undefined for undefined', () => {
  expect(normalizeAccountType(undefined)).toBe(undefined);
});

test('partner_bank._helpers: normalizeAccountType returns null for null', () => {
  expect(normalizeAccountType(null)).toBe(null);
});

test('partner_bank._helpers: normalizeAccountType returns null for empty', () => {
  expect(normalizeAccountType('')).toBe(null);
});

test('partner_bank._helpers: normalizeAccountType throws for invalid type', () => {
  let err: unknown;
  try {
    normalizeAccountType('invalid');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('AccountType must be one of checking, savings, corporate, other');
});

test('partner_bank._helpers: ACCOUNT_TYPES contains the four expected values', () => {
  expect(ACCOUNT_TYPES.has('checking')).toBe(true);
  expect(ACCOUNT_TYPES.has('savings')).toBe(true);
  expect(ACCOUNT_TYPES.has('corporate')).toBe(true);
  expect(ACCOUNT_TYPES.has('other')).toBe(true);
  expect(ACCOUNT_TYPES.has('invalid')).toBe(false);
});

// ---------------------------------------------------------------------------
// pickDefaultBankAccountId
// ---------------------------------------------------------------------------

test('partner_bank._helpers: pickDefaultBankAccountId returns null for undefined', () => {
  expect(pickDefaultBankAccountId(undefined, 'inbound')).toBe(null);
});

test('partner_bank._helpers: pickDefaultBankAccountId returns null for empty array', () => {
  expect(pickDefaultBankAccountId([], 'inbound')).toBe(null);
});

test('partner_bank._helpers: pickDefaultBankAccountId picks default inbound', () => {
  const accounts = [
    { Id: 'a', IsDefaultInbound: false },
    { Id: 'b', IsDefaultInbound: true },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe('b');
});

test('partner_bank._helpers: pickDefaultBankAccountId picks default outbound', () => {
  const accounts = [
    { Id: 'a', IsDefaultOutbound: false },
    { Id: 'b', IsDefaultOutbound: true },
  ];
  expect(pickDefaultBankAccountId(accounts, 'outbound')).toBe('b');
});

test('partner_bank._helpers: pickDefaultBankAccountId returns null when no default', () => {
  const accounts = [
    { Id: 'a', IsDefaultInbound: false },
    { Id: 'b', IsDefaultInbound: false },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe(null);
});

test('partner_bank._helpers: pickDefaultBankAccountId excludes inactive accounts', () => {
  const accounts = [
    { Id: 'a', IsDefaultInbound: true, IsActive: false },
    { Id: 'b', IsDefaultInbound: false, IsActive: true },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe(null);
});

test('partner_bank._helpers: pickDefaultBankAccountId excludes rows without Id', () => {
  const accounts = [
    { Id: '', IsDefaultInbound: true },
    { Id: 'b', IsDefaultInbound: false },
  ];
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe(null);
});

test('partner_bank._helpers: pickDefaultBankAccountId sorts deterministically by Id', () => {
  const accounts = [
    { Id: 'c', IsDefaultInbound: true },
    { Id: 'a', IsDefaultInbound: true },
    { Id: 'b', IsDefaultInbound: false },
  ];
  // Both 'a' and 'c' are defaults; after sorting by Id, 'a' comes first
  expect(pickDefaultBankAccountId(accounts, 'inbound')).toBe('a');
});
