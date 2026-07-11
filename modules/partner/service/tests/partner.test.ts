// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Partner from '@/partner/service/models/partner';

// ---------------------------------------------------------------------------
// sortContacts
// ---------------------------------------------------------------------------

test('partner: sortContacts returns empty array for undefined', () => {
  const result = (Partner as any).sortContacts(undefined);
  expect(result).toEqual([]);
});

test('partner: sortContacts returns empty array for null', () => {
  const result = (Partner as any).sortContacts(null);
  expect(result).toEqual([]);
});

test('partner: sortContacts returns empty array for empty list', () => {
  const result = (Partner as any).sortContacts([]);
  expect(result).toEqual([]);
});

test('partner: sortContacts filters out items without Id', () => {
  const result = (Partner as any).sortContacts([{ Name: 'no-id' }, { Id: 'A', Name: 'has-id', IsActive: true }]);
  expect(result).toHaveLength(1);
  expect(result[0].Id).toBe('A');
});

test('partner: sortContacts filters out inactive items', () => {
  const result = (Partner as any).sortContacts([
    { Id: 'A', Name: 'active', IsActive: true },
    { Id: 'B', Name: 'inactive', IsActive: false },
  ]);
  expect(result).toHaveLength(1);
  expect(result[0].Id).toBe('A');
});

test('partner: sortContacts treats missing IsActive as active', () => {
  const result = (Partner as any).sortContacts([
    { Id: 'A', Name: 'no-flag' },
    { Id: 'B', Name: 'explicit-inactive', IsActive: false },
  ]);
  expect(result).toHaveLength(1);
  expect(result[0].Id).toBe('A');
});

test('partner: sortContacts puts IsDefault first, then by Sequence, then by Id', () => {
  const result = (Partner as any).sortContacts([
    { Id: 'C', Name: 'c', IsActive: true, Sequence: 5 },
    { Id: 'A', Name: 'a', IsActive: true, IsDefault: true, Sequence: 20 },
    { Id: 'B', Name: 'b', IsActive: true, Sequence: 5 },
  ]);
  expect(result.map((c: any) => c.Id)).toEqual(['A', 'B', 'C']);
});

test('partner: sortContacts sorts by Sequence when IsDefault is same', () => {
  const result = (Partner as any).sortContacts([
    { Id: 'B', Name: 'b', IsActive: true, Sequence: 50 },
    { Id: 'A', Name: 'a', IsActive: true, Sequence: 10 },
  ]);
  expect(result.map((c: any) => c.Id)).toEqual(['A', 'B']);
});

test('partner: sortContacts sorts by Id when Sequence is same', () => {
  const result = (Partner as any).sortContacts([
    { Id: 'B', Name: 'b', IsActive: true, Sequence: 10 },
    { Id: 'A', Name: 'a', IsActive: true, Sequence: 10 },
  ]);
  expect(result.map((c: any) => c.Id)).toEqual(['A', 'B']);
});

test('partner: sortContacts uses default Sequence 10 for missing/undefined', () => {
  const result = (Partner as any).sortContacts([
    { Id: 'B', Name: 'b', IsActive: true },
    { Id: 'A', Name: 'a', IsActive: true, Sequence: 5 },
  ]);
  expect(result.map((c: any) => c.Id)).toEqual(['A', 'B']);
});

// ---------------------------------------------------------------------------
// hasAddress
// ---------------------------------------------------------------------------

test('partner: hasAddress returns true when contact has string AddressId', () => {
  const result = (Partner as any).hasAddress({ AddressId: 'addr-1' });
  expect(result).toBe(true);
});

test('partner: hasAddress returns true when contact has object AddressId with Id', () => {
  const result = (Partner as any).hasAddress({ AddressId: { Id: 'addr-2' } });
  expect(result).toBe(true);
});

test('partner: hasAddress returns false when AddressId is missing', () => {
  const result = (Partner as any).hasAddress({ Name: 'test' });
  expect(result).toBe(false);
});

test('partner: hasAddress returns false when AddressId is null', () => {
  const result = (Partner as any).hasAddress({ AddressId: null });
  expect(result).toBe(false);
});

test('partner: hasAddress returns false for undefined contact', () => {
  const result = (Partner as any).hasAddress(undefined);
  expect(result).toBe(false);
});

// ---------------------------------------------------------------------------
// pickDefaultContactId
// ---------------------------------------------------------------------------

test('partner: pickDefaultContactId prefers IsDefault contact without AddressType and with Name', () => {
  const contacts = [
    { Id: 'B', Name: 'b', IsActive: true, Sequence: 10 },
    { Id: 'A', Name: 'default', IsActive: true, IsDefault: true, Sequence: 5 },
  ];
  const result = (Partner as any).pickDefaultContactId(contacts);
  expect(result).toBe('A');
});

test('partner: pickDefaultContactId falls back to contact with Name and no AddressType', () => {
  const contacts = [{ Id: 'B', Name: 'fallback', IsActive: true, Sequence: 10, AddressType: null }];
  const result = (Partner as any).pickDefaultContactId(contacts);
  expect(result).toBe('B');
});

test('partner: pickDefaultContactId skips contacts with AddressType set', () => {
  const contacts = [
    { Id: 'A', Name: 'shipping', IsActive: true, IsDefault: true, AddressType: 'shipping', AddressId: 'addr-1' },
    { Id: 'B', Name: 'contact', IsActive: true, Sequence: 10 },
  ];
  const result = (Partner as any).pickDefaultContactId(contacts);
  expect(result).toBe('B');
});

test('partner: pickDefaultContactId returns first sorted contact as last resort', () => {
  const contacts = [
    { Id: 'A', Name: '', IsActive: true, Sequence: 10 },
    { Id: 'B', Name: '', IsActive: true, Sequence: 5 },
  ];
  // Both have empty names; sorted by sequence → B first, then A
  // Name is empty, AddressType is undefined, so the fallback checks
  // !AddressType && (Name.trim() || !hasAddress) — Name is empty but !hasAddress is true (no address)
  // So B should be picked
  const result = (Partner as any).pickDefaultContactId(contacts);
  expect(result).toBe('B');
});

test('partner: pickDefaultContactId returns null for empty contacts', () => {
  const result = (Partner as any).pickDefaultContactId([]);
  expect(result).toBeNull();
});

// ---------------------------------------------------------------------------
// pickDefaultAddressId
// ---------------------------------------------------------------------------

test('partner: pickDefaultAddressId picks matching type with IsDefault and hasAddress', () => {
  const contacts = [
    { Id: 'A', AddressType: 'shipping', IsDefault: true, AddressId: 'addr-1', IsActive: true },
    { Id: 'B', AddressType: 'billing', IsDefault: true, AddressId: 'addr-2', IsActive: true },
  ];
  const result = (Partner as any).pickDefaultAddressId(contacts, 'shipping');
  expect(result).toBe('A');
});

test('partner: pickDefaultAddressId returns null when no match for type', () => {
  const contacts = [{ Id: 'A', AddressType: 'billing', IsDefault: true, AddressId: 'addr-1', IsActive: true }];
  const result = (Partner as any).pickDefaultAddressId(contacts, 'shipping');
  expect(result).toBeNull();
});

test('partner: pickDefaultAddressId ignores non-default entries', () => {
  const contacts = [{ Id: 'A', AddressType: 'shipping', IsDefault: false, AddressId: 'addr-1', IsActive: true }];
  const result = (Partner as any).pickDefaultAddressId(contacts, 'shipping');
  expect(result).toBeNull();
});

test('partner: pickDefaultAddressId ignores entries without address', () => {
  const contacts = [{ Id: 'A', AddressType: 'shipping', IsDefault: true, AddressId: null, IsActive: true }];
  const result = (Partner as any).pickDefaultAddressId(contacts, 'shipping');
  expect(result).toBeNull();
});

test('partner: pickDefaultAddressId returns null for undefined contacts', () => {
  const result = (Partner as any).pickDefaultAddressId(undefined, 'shipping');
  expect(result).toBeNull();
});
