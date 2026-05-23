// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToManyField.vue'), 'utf8');
}

describe('OManyToManyField mapping/contract', () => {
  it('uses many2many generic typing and field registry contract', () => {
    const s = source();

    expect(s).toContain('P extends FieldPath<T, ClientModel<BaseModel>[]>');
    expect(s).toContain('binding.registerFields(`${binding.prop}.DisplayName`);');
    expect(s).toContain('const { getItems, insertItem, removeItemAt } = binding.asMutableArray();');
  });

  it('resolves relationStore by binding first then fallback model metadata', () => {
    const s = source();

    expect(s).toContain('targetModel?: string;');
    expect(s).toContain("targetModel: ''");
    expect(s).toContain('const relationStore = computed<WebModelStore<any> | undefined>(() => {');
    expect(s).toContain('if (binding.relationStore) return binding.relationStore as WebModelStore<any>;');
    expect(s).toContain('const target = props.targetModel || binding.meta?.relationModel;');
    expect(s).toContain('return createStoreByModel(target);');
    expect(s).toContain('if (!relationStore.value) {');
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

  it('maps picker selected records into inserted many2many objects', () => {
    const s = source();

    expect(s).toContain('const picked = unwrap(expose?.selectedItems) as any[] | undefined;');
    expect(s).toContain('const selected: any[] = Array.isArray(picked) ? picked.map(toRecord) : [];');
    expect(s).toContain('const ids = selected.map(x => x?.Id ?? x?.id).filter(Boolean);');
    expect(s).toContain('const existIds = new Set((getItems() || []).map((x: any) => x?.Id).filter(Boolean));');
    expect(s).toContain('const records = selected.filter(x => newIds.includes(String(x?.Id ?? x?.id)));');
    expect(s).toContain('for (const rec of records) insertItem(createRowKey({ ...(rec || {}) }) as any);');
  });

  it('keeps table row key hydration contract for inserted/displayed rows', () => {
    const s = source();

    expect(s).toContain('function defineHiddenRowKey(obj: any, key: string, val?: any) {');
    expect(s).toContain("defineHiddenRowKey(row, '__rowKey', seed);");
    expect(s).toContain('onMounted(hydrateRowKeys);');
    expect(s).toContain('() => getItems().length,');
  });
});
