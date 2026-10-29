// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator/model';
import PropertyDefinitionBaseModel from './property_definition_base_model';

@Model('PropertyDefinition', { application: 'pdhelp' })
class PdHelpPropertyDefinition extends PropertyDefinitionBaseModel {}

test('PropertyDefinition FieldsGet returns help for scope fields', async () => {
  const meta = await (PdHelpPropertyDefinition as any).FieldsGet(
    ['TargetModel', 'PropertiesField', 'ContainerModel', 'ContainerId'],
    ['help', 'helpText', 'string'],
  );
  expect(String(meta.TargetModel?.help || '')).toMatch(/properties column/i);
  expect(String(meta.PropertiesField?.help || '')).toMatch(/properties field/i);
  expect(String(meta.ContainerModel?.help || '')).toMatch(/app-level scope/i);
  expect(String(meta.ContainerId?.help || '')).toMatch(/Container Model/i);
  expect(meta.ContainerModel?.helpText?.src).toBeTruthy();
});
