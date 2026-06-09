// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(): string {
  return readFileSync(resolve(__dirname, 'OManyToManyRefTreeField.vue'), 'utf8');
}

describe('OManyToManyRefTreeField value mapping', () => {
  it('maps tree checked keys to field id array', () => {
    const s = source();

    expect(s).toContain('@check="onCheck"');
    expect(s).toContain('const checkedKeysFromTree = editTreeRef.value?.getCheckedKeys(false);');
    expect(s).toContain('clearItems();');
    expect(s).toContain('insertItem(id as any);');
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

  it('normalizes model rows and supports toEntity payload', () => {
    const s = source();

    expect(s).toContain("if (!input || typeof input !== 'object') return {};");
    expect(s).toContain("if (typeof (input as any).toEntity === 'function') {");
    expect(s).toContain('const entity = (input as any).toEntity();');
    expect(s).toContain('function parseChildrenValue(raw: any): any[] {');
    expect(s).toContain("for (const key of ['value', 'values', 'items']) {");
    expect(s).toContain('return Array.isArray(parsed) ? parsed : [];');
  });

  it('supports lazy load and layered preload strategies', () => {
    const s = source();

    expect(s).toContain(':lazy="lazy"');
    expect(s).toContain(':load="loadTreeNode"');
    expect(s).toContain('async function loadTreeNode(node: any, resolve: (data: TreeNode[]) => void): Promise<void>');
    expect(s).toContain('async function preloadTreeByLayers(roots: TreeNode[]): Promise<void>');
    expect(s).toContain('if (!props.lazy) {');
    expect(s).toContain('await preloadTreeByLayers(roots);');
  });

  it('supports customizable node slots and injects row data', () => {
    const s = source();

    expect(s).toContain('<slot name="node-edit"');
    expect(s).toContain('<slot name="node-display"');
    expect(s).toContain('<slot name="node"');
    expect(s).toContain(':row="data?.raw"');
    expect(s).toContain(':label="data?.__label"');
    expect(s).toContain(':id="data?.__id"');
    expect(s).toContain('mode="edit"');
    expect(s).toContain('mode="display"');
  });

  it('keeps node click expand/collapse behavior with custom slots', () => {
    const s = source();

    expect(s).toContain('@node-click="onNodeClick"');
    expect(s).toContain('@click.capture="onDisplayTreeClickCapture"');
    expect(s).toContain(':check-on-click-node="false"');
    expect(s).toContain(':check-on-click-leaf="false"');
    expect(s).toContain(':expand-on-click-node="false"');
    expect(s).toContain('if (!props.expandOnClickNode) return;');
    expect(s).toContain('function onDisplayTreeClickCapture(event: MouseEvent) {');
    expect(s).toContain("if (!target?.closest('.el-checkbox')) return;");
    expect(s).toContain('event.preventDefault();');
    expect(s).toContain("if (target?.closest('.el-checkbox')) return;");
    expect(s).toContain('toggleNodeExpanded(node);');
  });
});
