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
  // Input order is US then EU; tree must still emit EU before US by DisplayName.
  const unsorted = [
    { Id: 'root', DisplayName: 'Root Co', Code: 'ROOT' },
    { Id: 'us', DisplayName: 'US', Code: 'US', ParentId: 'root' },
    { Id: 'eu', DisplayName: 'EU', Code: 'EU', ParentId: 'root' },
    { Id: 'be', DisplayName: 'Belgium', Code: 'BE', ParentId: 'eu' },
    { Id: 'orphan', DisplayName: 'Orphan', Code: 'OR', ParentId: 'missing-parent' },
  ];
  const entries = buildCompanyTreeEntries(unsorted);
  expect(entries.map(e => `${e.level}:${e.id}`)).toEqual([
    '0:orphan',
    '0:root',
    '1:eu',
    '2:be',
    '1:us',
  ]);
  expect(entries.find(e => e.id === 'eu')?.childIds).toEqual(['be']);
});

test('buildCompanyTreeEntries: ParentId cycles stay selectable without looping', () => {
  const cyclic = [
    { Id: 'a', DisplayName: 'A', ParentId: 'b' },
    { Id: 'b', DisplayName: 'B', ParentId: 'a' },
  ];
  const entries = buildCompanyTreeEntries(cyclic);
  expect(entries.map(e => e.id).sort()).toEqual(['a', 'b']);
  const children = buildCompanyChildrenMap(cyclic);
  expect(collectCompanyBranchIds('a', children)).toEqual(['a', 'b']);
});

test('buildCompanyChildrenMap: ignores parents outside the allowed set', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(children.get('orphan')).toEqual([]);
  expect(children.get('eu')).toEqual(['be']);
  expect(children.has('missing-parent')).toBe(false);
});

test('buildCompanyChildrenMap: duplicate ParentId edges are de-duplicated', () => {
  const children = buildCompanyChildrenMap([
    { Id: 'p', DisplayName: 'Parent' },
    { Id: 'c', DisplayName: 'Child', ParentId: 'p' },
    { Id: 'c', DisplayName: 'Child again', ParentId: 'p' },
  ]);
  expect(children.get('p')).toEqual(['c']);
});

test('isSingleCompanyMode: ParentId cycles stop parent walks and queue revisits', () => {
  // Mutual parents: distinct climb roots → not single-company mode (covers seen.has).
  const mutual = new Map<string, string[]>([
    ['a', ['b']],
    ['b', ['a']],
  ]);
  expect(isSingleCompanyMode(['a', 'b'], mutual)).toBe(false);

  // Shared root with a cyclic sibling edge covers queued.has during the BFS check.
  const withRoot = new Map<string, string[]>([
    ['root', ['a', 'b']],
    ['a', ['b']],
    ['b', ['a']],
  ]);
  expect(isSingleCompanyMode(['root', 'a', 'b'], withRoot)).toBe(true);
});

test('collectCompanyBranchIds: includes self and descendants', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(collectCompanyBranchIds('eu', children)).toEqual(['eu', 'be']);
  expect(collectCompanyBranchIds('', children)).toEqual([]);
});

test('isSingleCompanyMode: one id or a fully selected subtree', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(isSingleCompanyMode(['eu'], children)).toBe(true);
  expect(isSingleCompanyMode(['eu', 'be'], children)).toBe(true);
  expect(isSingleCompanyMode(['eu', 'us'], children)).toBe(false);
  expect(isSingleCompanyMode(['root', 'eu', 'be', 'us'], children)).toBe(true);
  expect(isSingleCompanyMode([], children)).toBe(true);
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

test('logIntoCompany: empty company id is a no-op draft', () => {
  const children = buildCompanyChildrenMap(rows);
  expect(
    logIntoCompany({
      companyId: '  ',
      draftActiveCompanyId: 'us',
      draftEnabledCompanyIds: ['us'],
      children,
    }),
  ).toEqual({ activeCompanyId: 'us', enabledCompanyIds: ['us'] });
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
  expect(filterCompanyTreeBySearch(entries, '   ').map(e => e.id)).toEqual(entries.map(e => e.id));
});

test('ensureActiveInEnabledScope: inserts active when missing from enabled', () => {
  expect(ensureActiveInEnabledScope('us', ['eu'])).toEqual({
    activeCompanyId: 'us',
    enabledCompanyIds: ['us', 'eu'],
  });
  expect(ensureActiveInEnabledScope('', ['eu', 'us'])).toEqual({
    activeCompanyId: 'eu',
    enabledCompanyIds: ['eu', 'us'],
  });
  expect(ensureActiveInEnabledScope('', [])).toEqual({
    activeCompanyId: '',
    enabledCompanyIds: [],
  });
});

test('company rows: boolean ParentId and Id-only labels', () => {
  const entries = buildCompanyTreeEntries([
    { Id: 'solo', ParentId: true as any },
    { Id: 'kid', DisplayName: '', Code: '', ParentId: 'solo' },
  ]);
  expect(entries.map(e => e.id)).toEqual(['solo', 'kid']);
  expect(entries[0].name).toBe('solo');
});

test('company rows: Code is used when DisplayName is empty', () => {
  const entries = buildCompanyTreeEntries([{ Id: 'x', DisplayName: '', Code: 'CODE-X' }]);
  expect(entries[0].name).toBe('CODE-X');
});
