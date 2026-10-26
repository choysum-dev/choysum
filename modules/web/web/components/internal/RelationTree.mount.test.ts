// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Non-strict cascade: down writes ids; up is display-only (checked / half).
 */

import { nextTick } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import RelationTree from './RelationTree.vue';

const treeData = [
  {
    __id: 'root',
    __label: 'Root',
    __leafKnown: false,
    children: [
      { __id: 'a', __label: 'A', __leafKnown: true, children: [] },
      { __id: 'b', __label: 'B', __leafKnown: true, children: [] },
    ],
  },
];

describe('RelationTree cascade selection', () => {
  test('checkStrictly=false: checking parent writes parent and descendants', async () => {
    const m = mountApp(RelationTree as any, {
      props: {
        data: treeData,
        checkStrictly: false,
        defaultExpandAll: true,
        showCheckbox: true,
      },
    });
    await nextTick();
    const ss = m.setupState() as any;
    const rootRow = { key: 'root', data: treeData[0] };
    ss.onCheckboxChange(rootRow, true);
    await nextTick();
    const keys = ss.getCheckedKeys() as string[];
    expect(keys.sort()).toEqual(['a', 'b', 'root']);
    m.unmount();
  });

  test('checkStrictly=false: all children selected shows parent checked without writing parent id', async () => {
    const m = mountApp(RelationTree as any, {
      props: {
        data: treeData,
        checkStrictly: false,
        defaultExpandAll: true,
        showCheckbox: true,
      },
    });
    await nextTick();
    const ss = m.setupState() as any;
    ss.setCheckedKeys(['a', 'b']);
    await nextTick();

    expect(ss.getCheckedKeys().sort()).toEqual(['a', 'b']);
    expect(ss.checkboxDisplayState(treeData[0])).toBe(true);
    expect(ss.getHalfCheckedKeys()).toEqual([]);
    m.unmount();
  });

  test('checkStrictly=false: partial children yields indeterminate parent', async () => {
    const m = mountApp(RelationTree as any, {
      props: {
        data: treeData,
        checkStrictly: false,
        defaultExpandAll: true,
        showCheckbox: true,
      },
    });
    await nextTick();
    const ss = m.setupState() as any;
    ss.setCheckedKeys(['a']);
    await nextTick();

    expect(ss.checkboxDisplayState(treeData[0])).toBe('indeterminate');
    expect(ss.getHalfCheckedKeys()).toEqual(['root']);
    expect(ss.getCheckedKeys()).toEqual(['a']);
    m.unmount();
  });

  test('unchecking a display-only full parent clears descendants without inventing parent id', async () => {
    const m = mountApp(RelationTree as any, {
      props: {
        data: treeData,
        checkStrictly: false,
        defaultExpandAll: true,
        showCheckbox: true,
      },
    });
    await nextTick();
    const ss = m.setupState() as any;
    ss.setCheckedKeys(['a', 'b']);
    expect(ss.checkboxDisplayState(treeData[0])).toBe(true);

    const rootRow = { key: 'root', data: treeData[0] };
    ss.onCheckboxChange(rootRow, false);
    await flushPromises();
    expect(ss.getCheckedKeys()).toEqual([]);
    expect(ss.checkboxDisplayState(treeData[0])).toBe(false);
    m.unmount();
  });

  test('checkStrictly=true does not cascade or derive parent state', async () => {
    const m = mountApp(RelationTree as any, {
      props: {
        data: treeData,
        checkStrictly: true,
        defaultExpandAll: true,
        showCheckbox: true,
      },
    });
    await nextTick();
    const ss = m.setupState() as any;
    ss.onCheckboxChange({ key: 'root', data: treeData[0] }, true);
    expect(ss.getCheckedKeys()).toEqual(['root']);
    ss.setCheckedKeys(['a', 'b']);
    expect(ss.checkboxDisplayState(treeData[0])).toBe(false);
    expect(ss.getHalfCheckedKeys()).toEqual([]);
    m.unmount();
  });
});
