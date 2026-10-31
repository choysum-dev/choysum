// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { sandboxRecordsMenuTitle, sandboxRootMenuTitle } from '../menu/titles';
import { sandboxRoutes } from '../route/routes';

const listRouteTitle = createTranslate('sandbox', { scope: 'web/route/routes' })._lt('Sandbox Record List');
const propertyDefinitionsRouteTitle = createTranslate('sandbox', { scope: 'web/route/routes' })._lt(
  'Property Definitions'
);
const fieldDefaultsRouteTitle = createTranslate('sandbox', { scope: 'web/route/routes' })._lt(
  'Field Defaults'
);

test('sandbox menu titles: titles.ts registers shared TermReferences', () => {
  expect(sandboxRootMenuTitle.src).toBe('Sandbox');
  expect(sandboxRootMenuTitle.module).toBe('sandbox');
  expect(sandboxRecordsMenuTitle.src).toBe('Field Kitchen Sink');
});

test('sandbox route titles: list route pins pageTitleText', () => {
  const list = sandboxRoutes.find(route => route.name === 'SandboxRecordList') as any;
  expect(list).toBeTruthy();
  expect(list.meta?.pageTitle).toBe('Sandbox Record List');
  expect(list.meta?.pageTitleText).toEqual(listRouteTitle);
});

test('sandbox route titles: PropertyDefinition list pins pageTitleText', () => {
  const route = sandboxRoutes.find(r => r.name === 'PropertyDefinitionList_sandbox') as any;
  expect(route).toBeTruthy();
  expect(route.meta?.pageTitle).toBe('Property Definitions');
  expect(route.meta?.pageTitleText).toEqual(propertyDefinitionsRouteTitle);
});

test('sandbox route titles: FieldDefault list pins pageTitleText', () => {
  const route = sandboxRoutes.find(r => r.name === 'FieldDefaultList_sandbox') as any;
  expect(route).toBeTruthy();
  expect(route.meta?.pageTitle).toBe('Field Defaults');
  expect(route.meta?.pageTitleText).toEqual(fieldDefaultsRouteTitle);
});
