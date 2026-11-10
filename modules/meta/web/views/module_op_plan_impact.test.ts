// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  changedApplicationsFromModules,
  formatModuleOpPlanChangeSummary,
  formatPlanVersionDisplay,
  isChangingPlanEffect,
  isSatisfiedPlanEffect,
  mapPlanModuleEntry,
  moduleShowsApplicationHint,
  partitionPlanModules,
  planEffectSummaryToken,
  shouldShowAffectedApplications,
  shouldShowPlanVersionArrow,
} from './module_op_plan_impact';

test('partitionPlanModules: target first; satisfied collapsed separately', () => {
  const { changing, satisfied } = partitionPlanModules(
    [
      { moduleName: 'core', role: 'dependency', effect: 'already_satisfied', application: 'core' },
      { moduleName: 'partner', role: 'target', effect: 'install', application: 'partner' },
      { moduleName: 'auth', role: 'dependency', effect: 'already_satisfied', application: 'auth' },
    ],
    [],
  );
  expect(changing.map(m => m.moduleName)).toEqual(['partner']);
  expect(satisfied.map(m => m.moduleName)).toEqual(['core', 'auth']);
});

test('changedApplicationsFromModules + shouldShow: hide when single same-named app', () => {
  const changing = [{ moduleName: 'partner', application: 'partner', effect: 'install', role: 'target' }];
  const apps = changedApplicationsFromModules(changing);
  expect(apps).toEqual(['partner']);
  expect(shouldShowAffectedApplications(changing, apps)).toBe(false);
});

test('shouldShowAffectedApplications: multi-module collapse to one app', () => {
  const changing = [
    { moduleName: 'partner', application: 'partner', effect: 'uninstall', role: 'target' },
    {
      moduleName: 'partner_commercial',
      application: 'partner',
      effect: 'uninstall',
      role: 'dependent',
    },
  ];
  const apps = changedApplicationsFromModules(changing);
  expect(apps).toEqual(['partner']);
  expect(shouldShowAffectedApplications(changing, apps)).toBe(true);
});

test('shouldShowAffectedApplications: name mismatch alone does not open the apps block', () => {
  const changing = [
    { moduleName: 'partner_bank', application: 'partner', effect: 'install', role: 'target' },
  ];
  const apps = changedApplicationsFromModules(changing);
  expect(shouldShowAffectedApplications(changing, apps)).toBe(false);
  expect(moduleShowsApplicationHint(changing[0]!)).toBe(true);
});

test('formatPlanVersionDisplay: keeps the full version string', () => {
  expect(formatPlanVersionDisplay('0.0.0-20260626031438')).toBe('0.0.0-20260626031438');
  expect(formatPlanVersionDisplay(' 1.2.3 ')).toBe('1.2.3');
});

test('formatModuleOpPlanChangeSummary: uses effect not legacy reason', () => {
  expect(
    formatModuleOpPlanChangeSummary([
      { moduleName: 'partner', role: 'target', effect: 'install', reason: 'root' },
      { moduleName: 'core', role: 'dependency', effect: 'already_satisfied', reason: 'dependency' },
    ]),
  ).toBe('partner (will install); 1 already satisfied dependency module(s) skipped');
});

test('shouldShowPlanVersionArrow: skip uninstall and equal versions', () => {
  expect(
    shouldShowPlanVersionArrow({
      moduleName: 'x',
      effect: 'uninstall',
      currentVersion: '1',
      targetVersion: '1',
    }),
  ).toBe(false);
  expect(
    shouldShowPlanVersionArrow({
      moduleName: 'x',
      effect: 'install',
      currentVersion: '1',
      targetVersion: '1',
    }),
  ).toBe(false);
  expect(
    shouldShowPlanVersionArrow({
      moduleName: 'x',
      effect: 'upgrade',
      currentVersion: '1',
      targetVersion: '2',
    }),
  ).toBe(true);
});

test('mapPlanModuleEntry + effect helpers', () => {
  expect(
    mapPlanModuleEntry({
      ModuleName: ' core ',
      Application: 'base',
      Role: 'target',
      Effect: 'upgrade',
      CurrentVersion: '1',
      TargetVersion: '2',
      Origin: 'registry',
    }),
  ).toMatchObject({
    moduleName: 'core',
    application: 'base',
    role: 'target',
    effect: 'upgrade',
    origin: 'registry',
  });
  expect(isChangingPlanEffect('install')).toBe(true);
  expect(isChangingPlanEffect('upgrade')).toBe(true);
  expect(isChangingPlanEffect('uninstall')).toBe(true);
  expect(isChangingPlanEffect('already_satisfied')).toBe(false);
  expect(isSatisfiedPlanEffect('already_satisfied')).toBe(true);
  expect(planEffectSummaryToken('upgrade')).toBe('will upgrade');
  expect(planEffectSummaryToken('uninstall')).toBe('will uninstall');
  expect(planEffectSummaryToken('already_satisfied')).toBe('already satisfied');
  expect(planEffectSummaryToken('weird')).toBe('will change');
});

test('partitionPlanModules: legacy missing effect and dedupe', () => {
  const { changing, satisfied } = partitionPlanModules(
    [
      { moduleName: 'a', effect: 'install' },
      { moduleName: 'a', effect: 'install' },
      { moduleName: 'b' },
      { moduleName: 'c', effect: 'already_satisfied' },
      { moduleName: 'd', effect: 'noop' },
    ],
    [{ moduleName: 'e', effect: 'already_satisfied' }],
  );
  expect(changing.map(m => m.moduleName)).toEqual(['a', 'b', 'd']);
  expect(satisfied.map(m => m.moduleName)).toEqual(['c', 'e']);
});

test('formatModuleOpPlanChangeSummary: satisfied-only suffix', () => {
  expect(
    formatModuleOpPlanChangeSummary(
      [{ moduleName: 'core', effect: 'already_satisfied' }],
      [],
    ),
  ).toBe('1 already satisfied dependency module(s) skipped');
  expect(formatModuleOpPlanChangeSummary([], [])).toBe('');
});
