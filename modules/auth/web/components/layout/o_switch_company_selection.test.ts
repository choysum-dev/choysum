// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  branchCheckState,
  buildCompanyChildrenMap,
  buildCompanyTreeEntries,
  collectCompanyBranchIds,
  ensureActiveInEnabledScope,
  filterCompanyTreeBySearch,
  isCompanyBranchToggleDisabled,
  isSingleCompanyMode,
  logIntoCompany,
  selectAllCheckState,
  setEqCompanyIds,
  toggleCompanyBranch,
  toggleSelectAllCompanies,
  uniqCompanyIds,
} from './o_switch_company_selection';

const rows = [
  { Id: 'root', DisplayName: 'Root Co', Code: 'ROOT' },
  { Id: 'eu', DisplayName: 'EU', Code: 'EU', ParentId: 'root' },
  { Id: 'be', DisplayName: 'Belgium', Code: 'BE', ParentId: 'eu' },
  { Id: 'us', DisplayName: 'US', Code: 'US', ParentId: 'root' },
  { Id: 'orphan', DisplayName: 'Orphan', Code: 'OR', ParentId: 'missing-parent' },
];

test('uniqCompanyIds: trims, drops blanks, preserves first-seen order', () => {
  expect(uniqCompanyIds([' a ', 'b', 'a', '', 'b', 'c'])).toEqual(['a', 'b', 'c']);
});

test('setEqCompanyIds: compares as sets', () => {
  expect(setEqCompanyIds(['a', 'b'], ['b', 'a'])).toBe(true);
  expect(setEqCompanyIds(['a'], ['a', 'b'])).toBe(false);
});

test('buildCompanyTreeEntries: roots missing parents and sorts siblings by name', () => {
  const entries = buildCompanyTreeEntries(rows);
  expect(entries.map(e => `${e.level}:${e.id}`)).toEqual([
    '0:orphan',
    '0:root',
    '1:eu',
    '2:be',
    '1:us',
  ]);
});

test('buildCompanyChildrenMap: ignores parents outside the allowed set', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(children.get('orphan')).toEqual([]);
  expect(children.get('eu')).toEqual(['be']);
});

test('collectCompanyBranchIds: includes self and descendants', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(collectCompanyBranchIds('eu', children)).toEqual(['eu', 'be']);
});

test('isSingleCompanyMode: one id or a fully selected subtree', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(isSingleCompanyMode(['eu'], children)).toBe(true);
  expect(isSingleCompanyMode(['eu', 'be'], children)).toBe(true);
  expect(isSingleCompanyMode(['eu', 'us'], children)).toBe(false);
  expect(isSingleCompanyMode(['root', 'eu', 'be', 'us'], children)).toBe(true);
});

test('logIntoCompany: single mode replaces enabled with the branch', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(
    logIntoCompany({
      companyId: 'eu',
      draftActiveCompanyId: 'us',
      draftEnabledCompanyIds: ['us'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'eu', enabledCompanyIds: ['eu', 'be'] });
});

test('logIntoCompany: multi mode promotes the target and keeps others', () => {
  const children = buildCompanyChildrenMap(rows);
  // eu + us without root is multi-company mode (siblings, incomplete parent).
  expect(
    logIntoCompany({
      companyId: 'orphan',
      draftActiveCompanyId: 'eu',
      draftEnabledCompanyIds: ['eu', 'us'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'orphan', enabledCompanyIds: ['orphan', 'eu', 'us'] });
});

test('branchCheckState: reports full, empty, and indeterminate', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(branchCheckState('eu', ['eu', 'be'], children)).toBe(true);
  expect(branchCheckState('eu', ['us'], children)).toBe(false);
  expect(branchCheckState('eu', ['be'], children)).toBe('indeterminate');
});

test('toggleCompanyBranch: checks a branch into the draft', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(
    toggleCompanyBranch({
      companyId: 'eu',
      draftActiveCompanyId: 'us',
      draftEnabledCompanyIds: ['us'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'us', enabledCompanyIds: ['us', 'eu', 'be'] });
});

test('toggleCompanyBranch: unchecks a branch and moves active when needed', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(
    toggleCompanyBranch({
      companyId: 'eu',
      draftActiveCompanyId: 'eu',
      draftEnabledCompanyIds: ['eu', 'be', 'us'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'us', enabledCompanyIds: ['us'] });
});

test('toggleCompanyBranch: refuses to clear the last remaining company', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(
    toggleCompanyBranch({
      companyId: 'us',
      draftActiveCompanyId: 'us',
      draftEnabledCompanyIds: ['us'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'us', enabledCompanyIds: ['us'] });
});

test('toggleCompanyBranch: indeterminate becomes a full branch select', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(
    toggleCompanyBranch({
      companyId: 'eu',
      draftActiveCompanyId: 'be',
      draftEnabledCompanyIds: ['be'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'be', enabledCompanyIds: ['be', 'eu'] });
});

test('isCompanyBranchToggleDisabled: last remaining branch cannot uncheck', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(isCompanyBranchToggleDisabled('us', ['us'], children)).toBe(true);
  expect(isCompanyBranchToggleDisabled('us', ['us', 'eu'], children)).toBe(false);
});

test('toggleSelectAllCompanies: selects all, then clears to active only', () => {
  const first = toggleSelectAllCompanies({
    draftActiveCompanyId: 'us',
    draftEnabledCompanyIds: ['us'],
    allowedCompanyIds: ['root', 'eu', 'be', 'us'],
  });
  expect(first.activeCompanyId).toBe('us');
  expect(first.enabledCompanyIds).toEqual(['us', 'root', 'eu', 'be']);

  const second = toggleSelectAllCompanies({
    draftActiveCompanyId: first.activeCompanyId,
    draftEnabledCompanyIds: first.enabledCompanyIds,
    allowedCompanyIds: ['root', 'eu', 'be', 'us'],
  });
  expect(second).toEqual({ activeCompanyId: 'us', enabledCompanyIds: ['us'] });
});

test('selectAllCheckState: mirrors partial and full selection', () => {
  expect(selectAllCheckState(['us'], ['us', 'eu'])).toBe('indeterminate');
  expect(selectAllCheckState(['us', 'eu'], ['us', 'eu'])).toBe(true);
  expect(selectAllCheckState([], ['us', 'eu'])).toBe(false);
});

test('filterCompanyTreeBySearch: keeps matches and their ancestors', () => {
  const entries = buildCompanyTreeEntries(rows);
  const filtered = filterCompanyTreeBySearch(entries, 'bel');
  expect(filtered.map(e => e.id)).toEqual(['root', 'eu', 'be']);
});

test('ensureActiveInEnabledScope: inserts active when missing from enabled', () => {
  expect(ensureActiveInEnabledScope('us', ['eu'])).toEqual({
    activeCompanyId: 'us',
    enabledCompanyIds: ['us', 'eu'],
  });
});
