// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  deriveModelCreateActionId,
  resolveNameCreateActionId,
  shouldShowNameCreateEntry,
  toModelActionSnake,
} from './nameCreateVisibility';

test('nameCreateVisibility > toModelActionSnake handles empty and separators', () => {
  expect(toModelActionSnake('')).toBe('');
  expect(toModelActionSnake('   ')).toBe('');
  expect(toModelActionSnake(null as any)).toBe('');
  expect(toModelActionSnake('UserRole')).toBe('user_role');
  expect(toModelActionSnake('foo-bar baz')).toBe('foo_bar_baz');
});

test('nameCreateVisibility > derives conventional create action ids', () => {
  expect(deriveModelCreateActionId('partner.Partner')).toBe('partner.action.partner_create');
  expect(deriveModelCreateActionId('auth.UserRole')).toBe('auth.action.user_role_create');
  expect(deriveModelCreateActionId('demo.Foo-Bar Baz')).toBe('demo.action.foo_bar_baz_create');
  expect(deriveModelCreateActionId(null)).toBeUndefined();
  expect(deriveModelCreateActionId(undefined)).toBeUndefined();
  expect(deriveModelCreateActionId('')).toBeUndefined();
  expect(deriveModelCreateActionId('   ')).toBeUndefined();
  expect(deriveModelCreateActionId('invalid')).toBeUndefined();
  expect(deriveModelCreateActionId('partner.Bank.Account')).toBeUndefined();
  expect(deriveModelCreateActionId('.Partner')).toBeUndefined();
  expect(deriveModelCreateActionId('partner.')).toBeUndefined();
  expect(deriveModelCreateActionId(' .Partner')).toBeUndefined();
  expect(deriveModelCreateActionId('partner. ')).toBeUndefined();
});

test('nameCreateVisibility > resolves create action id from model or explicit prop', () => {
  expect(resolveNameCreateActionId('partner.Partner')).toBe('partner.action.partner_create');
  expect(resolveNameCreateActionId('partner.Partner', 'custom.action.x_create')).toBe('custom.action.x_create');
  expect(resolveNameCreateActionId('partner.Partner', '')).toBe('');
  expect(resolveNameCreateActionId('')).toBeUndefined();
});

test('nameCreateVisibility > hides when allowCreate false or no keyword', () => {
  expect(
    shouldShowNameCreateEntry({
      allowCreate: false,
      hasKeyword: true,
      relationQualifiedName: 'partner.Partner',
      hasAction: () => true,
    })
  ).toBe(false);
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: false,
      relationQualifiedName: 'partner.Partner',
      hasAction: () => true,
    })
  ).toBe(false);
});

test('nameCreateVisibility > requires hasAction for derived create action', () => {
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: true,
      relationQualifiedName: 'partner.Partner',
      hasAction: id => id === 'partner.action.partner_create',
    })
  ).toBe(true);
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: true,
      relationQualifiedName: 'partner.Partner',
      hasAction: () => false,
    })
  ).toBe(false);
  // Missing hasAction predicate → canShowAction allows non-empty action ids.
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: true,
      relationQualifiedName: 'partner.Partner',
    })
  ).toBe(true);
});

test('nameCreateVisibility > createActionId empty string skips ACL', () => {
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: true,
      createActionId: '',
      hasAction: () => false,
    })
  ).toBe(true);
});

test('nameCreateVisibility > hides when create action id cannot be resolved', () => {
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: true,
      relationQualifiedName: '',
      hasAction: () => true,
    })
  ).toBe(false);
  expect(
    shouldShowNameCreateEntry({
      allowCreate: true,
      hasKeyword: true,
      relationQualifiedName: 'invalid',
      hasAction: () => true,
    })
  ).toBe(false);
});

