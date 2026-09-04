// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import PartnerContact from '@/partner/service/models/partner_contact';
import { ChoysumError } from '@/core/service/error';

// ---------------------------------------------------------------------------
// assertAddressType
// ---------------------------------------------------------------------------

test('partner_contact: assertAddressType returns undefined for undefined', () => {
  const result = (PartnerContact as any).assertAddressType(undefined);
  expect(result).toBeUndefined();
});

test('partner_contact: assertAddressType returns null for null', () => {
  const result = (PartnerContact as any).assertAddressType(null);
  expect(result).toBeNull();
});

test('partner_contact: assertAddressType returns null for empty string', () => {
  const result = (PartnerContact as any).assertAddressType('');
  expect(result).toBeNull();
});

test('partner_contact: assertAddressType lowercases valid type', () => {
  const result = (PartnerContact as any).assertAddressType('BILLING');
  expect(result).toBe('billing');
});

test('partner_contact: assertAddressType returns valid type unchanged when already lowercase', () => {
  const result = (PartnerContact as any).assertAddressType('shipping');
  expect(result).toBe('shipping');
});

test('partner_contact: assertAddressType accepts all valid types', () => {
  const validTypes = ['billing', 'shipping', 'office', 'registered', 'other'];
  for (const t of validTypes) {
    expect((PartnerContact as any).assertAddressType(t)).toBe(t);
  }
});

test('partner_contact: assertAddressType throws for invalid type', () => {
  let err: unknown;
  try {
    (PartnerContact as any).assertAddressType('invalid');
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

test('partner_contact: ensureRowHasValue passes when Name is a translated map', () => {
  expect(() => (PartnerContact as any).ensureRowHasValue({ Name: { en_US: '', zh_CN: '甲' } })).not.toThrow();
});

test('partner_contact: validateEntity Browse catch skips persisted backfill', async () => {
  const originalBrowse = (PartnerContact as any).Browse;
  (PartnerContact as any).Browse = async () => {
    throw new Error('missing row');
  };
  try {
    let err: unknown;
    try {
      await (PartnerContact as any).validateEntity(
        {
          CompanyId: 'co-1',
          Name: 'Solo',
        },
        'missing-id'
      );
    } catch (e) {
      err = e;
    }
    expect(String((err as any)?.message || err)).toMatch(/PartnerId is required/);
  } finally {
    (PartnerContact as any).Browse = originalBrowse;
  }
});

test('partner_contact: validateEntity trims Title/Department lang maps', async () => {
  const values: Record<string, any> = {
    PartnerId: 'p-1',
    CompanyId: 'c-1',
    Name: 'Jane',
    AddressType: 'BILLING',
    Title: { en_US: ' VP ', zh_CN: ' 经理 ' },
    Department: { en_US: ' Sales ' },
  };
  await (PartnerContact as any).validateEntity(values, undefined);
  expect(values.Title).toEqual({ en_US: 'VP', zh_CN: '经理' });
  expect(values.Department).toEqual({ en_US: 'Sales' });
  expect(values.AddressType).toBe('billing');
});

test('partner_contact: validateEntity backfills AddressType from persisted row', async () => {
  const originalBrowse = (PartnerContact as any).Browse;
  (PartnerContact as any).Browse = async () => ({
    PartnerId: { Id: 'p-persisted' },
    CompanyId: 'c-persisted',
    AddressType: 'shipping',
    AddressId: 'addr-persisted',
  });
  try {
    const values: Record<string, any> = {
      Name: 'Backfill',
    };
    await (PartnerContact as any).validateEntity(values, 'contact-1');
    expect(values.PartnerId).toBe('p-persisted');
    expect(values.CompanyId).toBe('c-persisted');
    expect(values.AddressType).toBe('shipping');
    expect(values.AddressId).toBe('addr-persisted');
  } finally {
    (PartnerContact as any).Browse = originalBrowse;
  }
});
