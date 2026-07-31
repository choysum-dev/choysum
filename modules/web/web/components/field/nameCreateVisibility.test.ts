// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import {
  deriveModelCreateActionId,
  resolveNameCreateActionId,
  shouldShowNameCreateEntry,
} from './nameCreateVisibility';

describe('nameCreateVisibility', () => {
  it('derives conventional create action ids', () => {
    expect(deriveModelCreateActionId('partner.Partner')).toBe('partner.action.partner_create');
    expect(deriveModelCreateActionId('auth.UserRole')).toBe('auth.action.user_role_create');
    expect(deriveModelCreateActionId('')).toBeUndefined();
    expect(deriveModelCreateActionId('invalid')).toBeUndefined();
  });

  it('resolves create action id from model or explicit prop', () => {
    expect(resolveNameCreateActionId('partner.Partner')).toBe('partner.action.partner_create');
    expect(resolveNameCreateActionId('partner.Partner', 'custom.action.x_create')).toBe('custom.action.x_create');
    expect(resolveNameCreateActionId('partner.Partner', '')).toBe('');
    expect(resolveNameCreateActionId('')).toBeUndefined();
  });

  it('hides when allowCreate false or no keyword', () => {
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

  it('requires hasAction for derived create action', () => {
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
  });

  it('createActionId empty string skips ACL', () => {
    expect(
      shouldShowNameCreateEntry({
        allowCreate: true,
        hasKeyword: true,
        createActionId: '',
        hasAction: () => false,
      })
    ).toBe(true);
  });

  it('hides when create action id cannot be resolved', () => {
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
});
