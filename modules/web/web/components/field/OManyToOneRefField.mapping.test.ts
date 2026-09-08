// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToOneRefField.vue'), 'utf8');
}

test('OManyToOneRefField mapping/contract > uses shared value click payload and emit contract', () => {
  const s = source();

  expect(s).toContain("from '@/web/web/components/field/manyToOneTypes'");
  expect(s).toContain("(e: 'value-click', payload: ValueClickPayload<any>): void;");
});

test('OManyToOneRefField mapping/contract > supports auto value click mode by listener detection', () => {
  const s = source();

  expect(s).toContain("valueClickable?: boolean | 'auto';");
  expect(s).toContain("valueClickable: 'auto',");
  expect(s).toContain("return Boolean(p.onValueClick || p['onValue-click']);");
  expect(s).toContain('if (props.valueClickable === true) return true;');
  expect(s).toContain('if (props.valueClickable === false) return false;');
});

test('OManyToOneRefField mapping/contract > wires display mode click and keyboard activation', () => {
  const s = source();

  expect(s).toContain('@click="onDisplayValueClick(fieldValue().value as any, $event)"');
  expect(s).toContain('@keydown="onDisplayValueKeydown(fieldValue().value as any, $event)"');
  expect(s).toContain("if (event.key !== 'Enter' && event.key !== ' ') return;");
  expect(s).toContain("emit('value-click', { id, item: resolveDisplayItem(val), label: getDisplayLabel(val), source: 'display', event });");
  expect(s).toContain('.o-field-display-text--clickable');
});

test('OManyToOneRefField mapping/contract > wires remote typeahead to NameSearch with DisplayName fields', () => {
  const s = source();

  expect(s).toContain('relationStore.value?.NameSearch(');
  expect(s).toContain('...buildRelationalForField(');
  expect(s).toContain("fields: ['Id', 'DisplayName']");
  expect(s).not.toContain('buildKeywordCondition');
  expect(s).not.toContain('relationStore.value?.Search(final');
});

test('OManyToOneRefField mapping/contract > wires NameCreate quick-create entry and props', () => {
  const s = source();
  expect(s).toContain('runNameCreateQuickCreate');
  expect(s).toContain('shouldShowNameCreateEntry');
  expect(s).toContain('allowCreate');
  expect(s).toContain('o-m2o-name-create');
});

