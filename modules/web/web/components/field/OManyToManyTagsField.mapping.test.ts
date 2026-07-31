// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToManyTagsField.vue'), 'utf8');
}

describe('OManyToManyTagsField mapping/contract', () => {
  it('uses object-array many2many typing and field registration contract', () => {
    const s = source();

    expect(s).toContain('P extends FieldPath<T, ClientModel<BaseModel>[]>');
    expect(s).toContain('binding.registerFields(`${binding.prop}.DisplayName`);');
    expect(s).toContain('const { getItems, insertItem, clearItems } = binding.asMutableArray<any>();');
  });

  it('keeps row key hydration contract for object rows', () => {
    const s = source();

    expect(s).toContain('function defineHiddenRowKey(obj: any, key: string, val?: any) {');
    expect(s).toContain("defineHiddenRowKey(v, '__rowKey', seed);");
    expect(s).toContain('onMounted(hydrateRowKeys);');
    expect(s).toContain('() => getItems().length,');
  });

  it('maps picker selected rows into merged object-array values', () => {
    const s = source();

    expect(s).toContain('const selected: any[] = Array.isArray(picked) ? picked.map(toRecord) : [];');
    expect(s).toContain('const mergedIds = Array.from(new Set([...currentIds, ...ids]));');
    expect(s).toContain('insertItem(createRowKey({ ...(row || {}) }) as any);');
  });

  it('supports keyboard interactions and suggestion highlight', () => {
    const s = source();

    expect(s).toContain('@keydown="handleKeydown"');
    expect(s).toContain("if (event.key === 'Backspace') {");
    expect(s).toContain("if (event.key !== 'Enter') return;");
    expect(s).toContain('function highlightSuggestion(label: string): string {');
    expect(s).toContain('o-m2m-tags__suggestion-hit');
  });

  it('wires remote typeahead to NameSearch instead of FE keyword Search', () => {
    const s = source();

    expect(s).toContain('store.NameSearch(');
    expect(s).toContain('...buildRelationalForField(');
    expect(s).not.toContain('const searchableFields = computed<string[]>(() => {');
    expect(s).not.toContain('buildKeywordCondition(keyword, searchableFields.value, {');
    expect(s).not.toContain("operator: 'ilike'");
    expect(s).not.toContain('const condition = mergeCondition(effectiveConditions.value, keyword);');
    expect(s).not.toContain('const records = await store.Search(');
  });

  it('wires NameCreate quick-create entry and props', () => {
    const s = source();
    expect(s).toContain('store.NameCreate(');
    expect(s).toContain('shouldShowNameCreateEntry');
    expect(s).toContain('allowCreate');
    expect(s).toContain('o-m2m-name-create');
  });

  it('keeps selected options in the options list for el-select-v2 tag rendering', () => {
    const s = source();

    expect(s).toContain('@visible-change="onDropdownVisibleChange"');
    expect(s).toContain('const dropdownVisible = ref(false);');
    expect(s).toContain('Selected values must always remain in options so el-select-v2 can render tags.');
    expect(s).toContain('if (picked.has(key)) continue;');
    expect(s).not.toContain('if (!dropdownVisible.value) {');
    expect(s).not.toContain('if (dropdownVisible.value && picked.has(key)) continue;');
  });

  it('clears input keyword after selecting an option', () => {
    const s = source();

    expect(s).toContain(':reserve-keyword="false"');
    expect(s).toContain("searchKeyword.value = '';");
  });

  it('refreshes search options after removing a tag while dropdown is open', () => {
    const s = source();

    expect(s).toContain('const removed = currentIds.some(id => !next.has(id));');
    expect(s).toContain('if (dropdownVisible.value && removed) {');
    expect(s).toContain('void handleRemoteSearch(searchKeyword.value);');
  });

  it('supports lightweight tag click mode with auto switch', () => {
    const s = source();

    expect(s).toContain("tagClickable?: boolean | 'auto';");
    expect(s).toContain("tagClickable: 'auto',");
    expect(s).toContain("from '@/web/web/components/field/manyToManyTagsTypes'");
    expect(s).toContain("(e: 'tag-click', payload: TagClickPayload<any>): void;");
    expect(s).toContain("return Boolean(p.onTagClick || p['onTag-click']);");
    expect(s).toContain("emit('tag-click', { id: item.id, item: item.record, label: item.label, source: 'display', event });");
  });
});
