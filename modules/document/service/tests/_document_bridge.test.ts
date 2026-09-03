// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { requireText, requireUserId, requireCompanyId, assertPrincipal } from '../models/_document_bridge';

test('document._document_bridge: requireText returns trimmed string for valid input', () => {
  expect(requireText('  hello  ', 'testField')).toBe('hello');
  expect(requireText('world', 'testField')).toBe('world');
});

test('document._document_bridge: requireText throws INVALID_ARGUMENT for empty/whitespace', () => {
  let caught: ChoysumError | undefined;
  try {
    requireText('', 'testField');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.domain).toBe('document');
  expect(caught!.code).toBe('INVALID_ARGUMENT');
  expect(caught!.metadata?.field).toBe('testField');

  try {
    requireText('   ', 'anotherField');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught!.metadata?.field).toBe('anotherField');

  try {
    requireText(undefined, 'undefField');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught!.metadata?.field).toBe('undefField');
});

test('document._document_bridge: requireUserId throws UNAUTHENTICATED for empty identity', () => {
  let caught: ChoysumError | undefined;
  try {
    requireUserId('');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.code).toBe('UNAUTHENTICATED');

  try {
    requireUserId(undefined);
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught!.code).toBe('UNAUTHENTICATED');
});

test('document._document_bridge: requireCompanyId throws PERMISSION_DENIED for empty company', () => {
  let caught: ChoysumError | undefined;
  try {
    requireCompanyId('', 'prepare');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.code).toBe('PERMISSION_DENIED');
  expect(caught!.metadata?.stage).toBe('prepare');

  try {
    requireCompanyId(null, 'finalize');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught!.metadata?.stage).toBe('finalize');
});

test('document._document_bridge: assertPrincipal validates required fields', () => {
  const principal = assertPrincipal({
    userId: 'usr_test',
    activeCompanyId: 'cmp_test',
    enabledCompanyIds: ['cmp_a', 'cmp_b'],
  });
  expect(principal.userId).toBe('usr_test');
  expect(principal.activeCompanyId).toBe('cmp_test');
  expect(principal.enabledCompanyIds).toEqual(['cmp_a', 'cmp_b']);
});

test('document._document_bridge: assertPrincipal filters empty enabledCompanyIds entries', () => {
  const principal = assertPrincipal({
    userId: 'usr_test',
    activeCompanyId: 'cmp_test',
    enabledCompanyIds: ['cmp_a', '', '  ', 'cmp_b'],
  });
  expect(principal.enabledCompanyIds).toEqual(['cmp_a', 'cmp_b']);
});

test('document._document_bridge: assertPrincipal treats missing enabledCompanyIds as undefined', () => {
  const principal = assertPrincipal({
    userId: 'usr_test',
    activeCompanyId: 'cmp_test',
  });
  expect(principal.userId).toBe('usr_test');
  expect(principal.activeCompanyId).toBe('cmp_test');
  expect(principal.enabledCompanyIds).toBeUndefined();
});

test('document._document_bridge: assertPrincipal throws for non-array enabledCompanyIds', () => {
  let caught: ChoysumError | undefined;
  try {
    assertPrincipal({
      userId: 'usr_test',
      activeCompanyId: 'cmp_test',
      enabledCompanyIds: 'not-an-array',
    });
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.code).toBe('INVALID_ARGUMENT');
});

test('document._document_bridge: assertPrincipal throws for missing userId', () => {
  let caught: ChoysumError | undefined;
  try {
    assertPrincipal({ activeCompanyId: 'cmp_test' });
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.code).toBe('INVALID_ARGUMENT');
});

test('document._document_bridge: assertPrincipal throws for missing activeCompanyId', () => {
  let caught: ChoysumError | undefined;
  try {
    assertPrincipal({ userId: 'usr_test' });
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.code).toBe('INVALID_ARGUMENT');
});
