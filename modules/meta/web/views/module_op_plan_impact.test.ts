// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  changedApplicationsFromModules,
  formatModuleOpPlanChangeSummary,
  formatPlanVersionDisplay,
  moduleShowsApplicationHint,
  partitionPlanModules,
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
