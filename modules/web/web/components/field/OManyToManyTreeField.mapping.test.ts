// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToManyTreeField.vue'), 'utf8');
}

describe('OManyToManyTreeField value mapping', () => {
  it('maps tree checked keys to many2many object array', () => {
    const s = source();

    expect(s).toContain('@check="onCheck"');
    expect(s).toContain('const checkedKeysFromTree = editTreeRef.value?.getCheckedKeys(false);');
    expect(s).toContain('const nodeRawById = collectNodeRawByIdFromTree(treeData.value);');
    expect(s).toContain('clearItems();');
    expect(s).toContain('insertItem(existing as any);');
    expect(s).toContain('insertItem(raw as any);');
    expect(s).toContain('insertItem({ Id: id } as any);');
  });

  it('uses many2many generic typing and relation model fallback', () => {
    const s = source();

    expect(s).toContain('P extends FieldPath<T, ClientModel<BaseModel>[]>');
    expect(s).toContain("targetModel: ''");
    expect(s).toContain('const target = props.targetModel || binding.meta?.relationModel;');
  });

  it('supports children relation loading contract and defaults', () => {
    const s = source();

    expect(s).toContain('childrenField?: string;');
    expect(s).toContain("childrenField: 'Childs'");
    expect(s).toContain('lazy?: boolean;');
    expect(s).toContain('maxDepth?: number;');
    expect(s).toContain('rootCondition?: AnyQueryCondition | AnyQueryCondition[];');
    expect(s).toContain('lazy: false');
    expect(s).toContain('maxDepth: 0');
    expect(s).toContain('rootCondition: undefined');
    expect(s).toContain('nodeKeyField?: string;');
    expect(s).toContain('labelField?: string | string[];');
    expect(s).toContain('fields?: string[];');
    expect(s).toContain("const relationName = String(props.childrenField || 'Childs');");
    expect(s).toContain("return ['Id', relationName] as any[];");
    expect(s).toContain("isLeaf: '__leafKnown'");
  });

  it('supports lazy load and layered preload strategies', () => {
    const s = source();

    expect(s).toContain(':lazy="lazy"');
    expect(s).toContain(':load="loadTreeNode"');
    expect(s).toContain(':check-on-click-node="false"');
    expect(s).toContain(':check-on-click-leaf="false"');
    expect(s).toContain('async function loadTreeNode(node: any, resolve: (data: TreeNode[]) => void): Promise<void>');
    expect(s).toContain('async function preloadTreeByLayers(roots: TreeNode[]): Promise<void>');
    expect(s).toContain('if (!props.lazy) {');
    expect(s).toContain('await preloadTreeByLayers(roots);');
  });
});
