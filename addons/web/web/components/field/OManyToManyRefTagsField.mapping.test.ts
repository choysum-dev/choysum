// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToManyRefTagsField.vue'), 'utf8');
}

describe('OManyToManyRefTagsField mapping/contract', () => {
  it('uses ref many2many generic typing and mutable array contract', () => {
    const s = source();

    expect(s).toContain('P extends FieldPath<T, string[]>');
    expect(s).toContain('const { getItems, insertItem, clearItems } = binding.asMutableArray<any>();');
    expect(s).toContain("defineOptions({ name: 'OManyToManyRefTagsField'");
  });

  it('builds effective conditions by combining exclude-picked, external and onchange', () => {
    const s = source();

    expect(s).toContain('const excludePicked = computed<QueryCondition<any> | undefined>(() => {');
    expect(s).toContain("return ['Id', 'not in', ids] as unknown as QueryCondition<any>;");
    expect(s).toContain('const externalConditions = computed<QueryCondition<any>[]>(() => toArray(props.condition));');
    expect(s).toContain('const onchangeConditions = computed<QueryCondition<any>[]>(() => {');
    expect(s).toContain('if (parts.length === 0) return [] as any;');
    expect(s).toContain('return { And: parts } as any;');
  });

  it('supports picker mapping and selectedItems wrapper unwrapping', () => {
    const s = source();

    expect(s).toContain('const picked = unwrap(expose?.selectedItems) as any[] | undefined;');
    expect(s).toContain('const selected: any[] = Array.isArray(picked) ? picked.map(toRecord) : [];');
    expect(s).toContain('const merged = Array.from(new Set([...prevIds, ...ids]));');
    expect(s).toContain('insertItem(id as any);');
  });

  it('supports keyboard interactions and suggestion highlight', () => {
    const s = source();

    expect(s).toContain('@keydown="handleKeydown"');
    expect(s).toContain("if (event.key === 'Backspace') {");
    expect(s).toContain("if (event.key !== 'Enter') return;");
    expect(s).toContain('function highlightSuggestion(label: string): string {');
    expect(s).toContain('o-m2m-tags__suggestion-hit');
  });

  it('builds keyword condition from metadata-aware searchable fields', () => {
    const s = source();

    expect(s).toContain('const searchableFields = computed<string[]>(() => {');
    expect(s).toContain('resolveKeywordFieldsByMeta(candidates, {');
    expect(s).toContain('fieldsMeta: relationStore.value?.fieldsMetadata');
    expect(s).toContain('buildKeywordCondition(keyword, searchableFields.value, {');
    expect(s).toContain("operator: 'ilike'");
  });

  it('hides already selected options when dropdown is visible', () => {
    const s = source();

    expect(s).toContain('@visible-change="onDropdownVisibleChange"');
    expect(s).toContain('const dropdownVisible = ref(false);');
    expect(s).toContain('if (!dropdownVisible.value) {');
    expect(s).toContain('if (dropdownVisible.value && picked.has(key)) continue;');
  });

  it('clears input keyword after selecting an option', () => {
    const s = source();

    expect(s).toContain(':reserve-keyword="false"');
    expect(s).toContain("searchKeyword.value = '';");
  });

  it('refreshes search options after removing a tag while dropdown is open', () => {
    const s = source();

    expect(s).toContain('const removed = prevIds.some(id => !next.has(id));');
    expect(s).toContain('if (dropdownVisible.value && removed) {');
    expect(s).toContain('void handleRemoteSearch(searchKeyword.value);');
  });

  it('supports lightweight tag click mode with auto switch', () => {
    const s = source();

    expect(s).toContain("tagClickable?: boolean | 'auto';");
    expect(s).toContain("tagClickable: 'auto',");
    expect(s).toContain('export type TagClickPayload<T = any> = {');
    expect(s).toContain("(e: 'tag-click', payload: TagClickPayload<any>): void;");
    expect(s).toContain("return Boolean(p.onTagClick || p['onTag-click']);");
    expect(s).toContain("emit('tag-click', { id: item.id, item: item.record, label: item.label, source: 'display', event });");
  });
});
