// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  isActiveCompanyEnabledLocked,
  syncCompanyDraftsFromJwt,
  toggleEnabledCompanyId,
} from './o_switch_company_draft';

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

test('toggleEnabledCompanyId: checks a company into the enabled draft', () => {
  expect(
    toggleEnabledCompanyId({
      companyId: 'c2',
      on: true,
      draftActiveCompanyId: 'c1',
      draftEnabledCompanyIds: ['c1'],
    }),
  ).toEqual(['c1', 'c2']);
});

test('toggleEnabledCompanyId: unchecks a non-active company', () => {
  expect(
    toggleEnabledCompanyId({
      companyId: 'c2',
      on: false,
      draftActiveCompanyId: 'c1',
      draftEnabledCompanyIds: ['c1', 'c2'],
    }),
  ).toEqual(['c1']);
});

test('toggleEnabledCompanyId: refuses to uncheck the active company', () => {
  expect(
    toggleEnabledCompanyId({
      companyId: 'c1',
      on: false,
      draftActiveCompanyId: 'c1',
      draftEnabledCompanyIds: ['c1', 'c2'],
    }),
  ).toEqual(['c1', 'c2']);
});

test('toggleEnabledCompanyId: indeterminate is treated as unchecked when not locked', () => {
  expect(
    toggleEnabledCompanyId({
      companyId: 'c2',
      on: 'indeterminate',
      draftActiveCompanyId: 'c1',
      draftEnabledCompanyIds: ['c1', 'c2'],
    }),
  ).toEqual(['c1']);
});

test('toggleEnabledCompanyId: ensures active stays in enabled after check', () => {
  expect(
    toggleEnabledCompanyId({
      companyId: 'c2',
      on: true,
      draftActiveCompanyId: 'c1',
      draftEnabledCompanyIds: [],
    }),
  ).toEqual(['c1', 'c2']);
});
