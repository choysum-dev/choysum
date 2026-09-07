// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { appRoutes } from '@/auth/web/route/routes';

const ACCESS_RULE_ROUTE_NAMES = new Set([
  'RecordRuleList',
  'RecordRuleDetail',
  'RecordRuleCreate',
  'FieldRuleList',
  'FieldRuleDetail',
  'FieldRuleCreate',
  'MethodAccessList',
  'MethodAccessDetail',
  'MethodAccessCreate',
  'UiResourceGrantList',
  'UiResourceGrantDetail',
  'UiResourceGrantCreate',
]);

describe('Access Rules admin routes', () => {
  it('registers Access Rules routes as lazy components without page prop injection', () => {
    const accessRoutes = appRoutes.filter(r => ACCESS_RULE_ROUTE_NAMES.has(String(r.name)));
    expect(accessRoutes.length).toBe(12);

    for (const route of accessRoutes) {
      expect(typeof route.component).toBe('function');
      // Detail/create pages resolve record id from the route inside OFormView.
      expect(route.props == null || route.props === false).toBe(true);
    }
  });
});
