// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { deriveCreateRouteName, resolveCreateRouteLocation } from './resolveCreateRoute';
import type { Router } from 'vue-router';

describe('deriveCreateRouteName', () => {
  it('maps List/Detail/Kanban stems to Create', () => {
    expect(deriveCreateRouteName('PartnerList')).toBe('PartnerCreate');
    expect(deriveCreateRouteName('PartnerDetail')).toBe('PartnerCreate');
    expect(deriveCreateRouteName('TokenKanban')).toBe('TokenCreate');
    expect(deriveCreateRouteName('FieldRuleList')).toBe('FieldRuleCreate');
  });

  it('keeps Create names for form New on the create screen', () => {
    expect(deriveCreateRouteName('PartnerCreate')).toBe('PartnerCreate');
  });

  it('returns undefined for non-surface names', () => {
    expect(deriveCreateRouteName('MetaModuleListTable')).toBeUndefined();
    expect(deriveCreateRouteName('MetaModuleHistory')).toBeUndefined();
    expect(deriveCreateRouteName('login')).toBeUndefined();
    expect(deriveCreateRouteName('')).toBeUndefined();
    expect(deriveCreateRouteName(undefined)).toBeUndefined();
  });
});

describe('resolveCreateRouteLocation', () => {
  it('returns a named location when the Create route matches', () => {
    const router = {
      resolve: vi.fn(() => ({ name: 'PartnerCreate', matched: [{ path: '/partner/partners/new' }] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toEqual({ name: 'PartnerCreate' });
    expect(router.resolve).toHaveBeenCalledWith({ name: 'PartnerCreate' });
  });

  it('returns undefined when resolve has no match', () => {
    const router = {
      resolve: vi.fn(() => ({ name: 'MetaModuleCreate', matched: [] })),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'MetaModuleList')).toBeUndefined();
  });

  it('returns undefined when resolve throws', () => {
    const router = {
      resolve: vi.fn(() => {
        throw new Error('No match');
      }),
    } as unknown as Router;
    expect(resolveCreateRouteLocation(router, 'PartnerList')).toBeUndefined();
  });
});
