// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import PartnerContact from '@/partner/service/models/partner_contact';
import { ChoysumError } from '@/core/service/error';

// ---------------------------------------------------------------------------
// normalizeAddressType
// ---------------------------------------------------------------------------

test('partner_contact: normalizeAddressType returns undefined for undefined', () => {
  const result = (PartnerContact as any).normalizeAddressType(undefined);
  expect(result).toBeUndefined();
});

test('partner_contact: normalizeAddressType returns null for null', () => {
  const result = (PartnerContact as any).normalizeAddressType(null);
  expect(result).toBeNull();
});

test('partner_contact: normalizeAddressType returns null for empty string', () => {
  const result = (PartnerContact as any).normalizeAddressType('');
  expect(result).toBeNull();
});

test('partner_contact: normalizeAddressType lowercases valid type', () => {
  const result = (PartnerContact as any).normalizeAddressType('BILLING');
  expect(result).toBe('billing');
});

test('partner_contact: normalizeAddressType returns valid type unchanged when already lowercase', () => {
  const result = (PartnerContact as any).normalizeAddressType('shipping');
  expect(result).toBe('shipping');
});

test('partner_contact: normalizeAddressType accepts all valid types', () => {
  const validTypes = ['billing', 'shipping', 'office', 'registered', 'other'];
  for (const t of validTypes) {
    expect((PartnerContact as any).normalizeAddressType(t)).toBe(t);
  }
});

test('partner_contact: normalizeAddressType throws for invalid type', () => {
  let err: unknown;
  try {
    (PartnerContact as any).normalizeAddressType('invalid');
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('AddressType must be one of billing, shipping, office, registered, other');
});

// ---------------------------------------------------------------------------
// ensureRowHasValue
// ---------------------------------------------------------------------------

test('partner_contact: ensureRowHasValue passes when Name is present', () => {
  expect(() => (PartnerContact as any).ensureRowHasValue({ Name: 'John' })).not.toThrow();
});

test('partner_contact: ensureRowHasValue passes when AddressId is present', () => {
  expect(() => (PartnerContact as any).ensureRowHasValue({ AddressId: 'addr-1' })).not.toThrow();
});

test('partner_contact: ensureRowHasValue passes when Email is present', () => {
  expect(() => (PartnerContact as any).ensureRowHasValue({ Email: 'test@example.com' })).not.toThrow();
});

test('partner_contact: ensureRowHasValue passes when Phone is present', () => {
  expect(() => (PartnerContact as any).ensureRowHasValue({ Phone: '123456' })).not.toThrow();
});

test('partner_contact: ensureRowHasValue passes when Mobile is present', () => {
  expect(() => (PartnerContact as any).ensureRowHasValue({ Mobile: '987654' })).not.toThrow();
});

test('partner_contact: ensureRowHasValue throws when all fields are empty', () => {
  let err: unknown;
  try {
    (PartnerContact as any).ensureRowHasValue({ Name: '', Email: '', Phone: '', Mobile: '' });
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('PartnerContact requires at least Name, AddressId, Email, Phone, or Mobile');
});

test('partner_contact: ensureRowHasValue throws when all fields are missing', () => {
  let err: unknown;
  try {
    (PartnerContact as any).ensureRowHasValue({});
  } catch (e) {
    err = e;
  }
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).message).toBe('PartnerContact requires at least Name, AddressId, Email, Phone, or Mobile');
});
