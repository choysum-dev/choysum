// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getIntermediateAncestors,
  getMenuBasePath,
  getMenuId,
  getPageTitle,
  isMenuNavigatedCrumb,
  resolveAppRoot,
} from './breadcrumb_helpers';

test('breadcrumb_helpers resolveAppRoot walks parents and tolerates cycles', () => {
  const root = { id: 'root' };
  const child = { id: 'child', __parent: root };
  expect(resolveAppRoot(child)).toBe(root);
  expect(resolveAppRoot(root)).toBe(root);
  expect(resolveAppRoot(null)).toBeNull();

  const cyclic: any = { id: 'cyclic' };
  cyclic.__parent = cyclic;
  expect(resolveAppRoot(cyclic)).toBe(cyclic);
});

test('breadcrumb_helpers getMenuId prefers id then path then name', () => {
  expect(getMenuId(null)).toBe('');
  expect(getMenuId({ id: 'a' })).toBe('a');
  expect(getMenuId({ path: '/a/b' })).toBe('-a-b');
  expect(getMenuId({ name: 'Named' })).toBe('Named');
  expect(getMenuId({})).toBe('unknown-menu');
});

test('breadcrumb_helpers getIntermediateAncestors returns folders between app and leaf', () => {
  const settings = { id: 'web.menu.settings', title: 'Settings' };
  const core = { id: 'web.menu.settings_app_core', title: 'core', __parent: settings };
  const leaf = {
    id: 'web.menu.terminology_core',
    title: 'Translations',
    path: '/web/terminology/core',
    __parent: core,
  };
  expect(getIntermediateAncestors(leaf, settings).map((m) => m.id)).toEqual([
    'web.menu.settings_app_core',
  ]);
  expect(getIntermediateAncestors(leaf, null)).toEqual([]);
  expect(getIntermediateAncestors(core, settings)).toEqual([]);

  const direct = { id: 'currency', title: 'Currency', path: '/base/currencies', __parent: settings };
  expect(getIntermediateAncestors(direct, settings)).toEqual([]);
});

test('breadcrumb_helpers getMenuBasePath falls back to the current route path', () => {
  expect(getMenuBasePath('/now', { path: '/menu' })).toBe('/menu');
  expect(getMenuBasePath('/now', {})).toBe('/now');
  expect(getMenuBasePath('/now', null)).toBe('/now');
});

test('breadcrumb_helpers getPageTitle reads meta title fields and path heuristics', () => {
  expect(getPageTitle({ meta: { pageTitleText: 'T' } }, 'D', 'P')).toBe('T');
  expect(getPageTitle({ meta: { pageTitle: 'Title' } }, 'D', 'P')).toBe('Title');
  expect(
    getPageTitle({ meta: { pageTitle: (r: any) => `F:${r.path}` }, path: '/x' }, 'D', 'P'),
  ).toBe('F:/x');
  expect(getPageTitle({ meta: { title: 'Meta' } }, 'D', 'P')).toBe('Meta');
  expect(getPageTitle({ name: 'RouteName', meta: {} }, 'D', 'P')).toBe('RouteName');
  expect(getPageTitle({ path: '/a/42', meta: {} }, 'D', 'P')).toBe('D');
  expect(getPageTitle({ path: '/a/abcdefghijklmnop', meta: {} }, 'D', 'P')).toBe('D');
  expect(getPageTitle({ path: '/a/plain', meta: {} }, 'D', 'P')).toBe('P');
});

test('breadcrumb_helpers isMenuNavigatedCrumb matches app and pathless folders', () => {
  expect(isMenuNavigatedCrumb({ kind: 'app' })).toBe(true);
  expect(isMenuNavigatedCrumb({ kind: 'folder', path: '' })).toBe(true);
  expect(isMenuNavigatedCrumb({ kind: 'folder' })).toBe(true);
  expect(isMenuNavigatedCrumb({ kind: 'folder', path: '/x' })).toBe(false);
  expect(isMenuNavigatedCrumb({ kind: 'menu', path: '/x' })).toBe(false);
  expect(isMenuNavigatedCrumb({ kind: 'page' })).toBe(false);
});
