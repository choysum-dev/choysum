// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isActiveCompanyEnabledLocked, syncCompanyDraftsFromJwt } from './o_switch_company_draft';
import { pickAlternativeCompanyOptionValue } from './switch_company_option_pick';

test('pickAlternativeCompanyOptionValue: keeps current when it is already a valid alternative', () => {
  expect(pickAlternativeCompanyOptionValue(['c1', 'c2'], 'c2', 'c1')).toBe('c2');
});

test('pickAlternativeCompanyOptionValue: picks another option when current matches active', () => {
  expect(pickAlternativeCompanyOptionValue(['c1', 'c2'], 'c1', 'c1')).toBe('c2');
});

test('pickAlternativeCompanyOptionValue: returns empty when no alternative exists', () => {
  expect(pickAlternativeCompanyOptionValue(['c1'], 'c1', 'c1')).toBe('');
  expect(pickAlternativeCompanyOptionValue([], '', 'c1')).toBe('');
});

test('pickAlternativeCompanyOptionValue: prefers current over other candidates when already alternative', () => {
  // Two options; JWT active moved to c1 while select still shows c2.
  expect(pickAlternativeCompanyOptionValue(['c1', 'c2'], 'c2', 'c1')).toBe('c2');
});

test('SwitchCompany draft guard: ignores JWT sync while panel is open', () => {
  const applied: Array<{ active: string; enabled: string[] }> = [];
  syncCompanyDraftsFromJwt({
    panelVisible: true,
    activeCompanyId: 'c2',
    enabledCompanyIds: ['c2'],
    apply: (active, enabled) => applied.push({ active, enabled }),
  });
  expect(applied.length).toBe(0);
});

test('SwitchCompany draft guard: syncs drafts from JWT when panel is closed', () => {
  const applied: Array<{ active: string; enabled: string[] }> = [];
  syncCompanyDraftsFromJwt({
    panelVisible: false,
    activeCompanyId: 'c2',
    enabledCompanyIds: ['c1', 'c2'],
    apply: (active, enabled) => applied.push({ active, enabled }),
  });
  expect(applied).toEqual([{ active: 'c2', enabled: ['c1', 'c2'] }]);
});

test('isActiveCompanyEnabledLocked: locks only the draft active company', () => {
  expect(isActiveCompanyEnabledLocked('c1', 'c1')).toBe(true);
  expect(isActiveCompanyEnabledLocked('c2', 'c1')).toBe(false);
  expect(isActiveCompanyEnabledLocked('c1', '')).toBe(false);
  expect(isActiveCompanyEnabledLocked('  c1  ', 'c1')).toBe(true);
});
