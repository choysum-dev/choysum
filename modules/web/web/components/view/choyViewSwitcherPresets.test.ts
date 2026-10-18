// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import {
  getChoyViewSwitcherPreset,
  listChoyViewSwitcherPresetIds,
  registerChoyViewSwitcherPreset,
  resetChoyViewSwitcherPresetsForTests,
  resolveChoyViewSwitcherOptions,
} from './choyViewSwitcherPresets';

const ExtraIcon = defineComponent({
  name: 'ExtraIcon',
  setup() {
    return () => h('span', { class: 'extra-icon' });
  },
});

describe('choyViewSwitcherPresets', () => {
  afterEach(() => {
    resetChoyViewSwitcherPresetsForTests();
  });

  test('builtins cover list / kanban / board / chart', () => {
    const ids = listChoyViewSwitcherPresetIds();
    expect(ids).toContain('list');
    expect(ids).toContain('kanban');
    expect(ids).toContain('board');
    expect(ids).toContain('chart');
    expect(getChoyViewSwitcherPreset('chart')?.defaultLabel).toBe('Chart');
  });

  test('resolve fills icons from presets and keeps caller order', () => {
    const resolved = resolveChoyViewSwitcherOptions([
      { value: 'list', label: 'List View' },
      { value: 'kanban', label: 'Kanban View' },
      { value: 'chart', label: 'Chart View' },
    ]);
    expect(resolved.map((o) => o.value)).toEqual(['list', 'kanban', 'chart']);
    expect(resolved.every((o) => !!o.icon)).toBe(true);
    expect(resolved[2]?.label).toBe('Chart View');
  });

  test('resolve drops hidden and skips unknown kinds without icon', () => {
    const warn = console.warn;
    const seen: string[] = [];
    console.warn = (msg?: unknown) => {
      seen.push(String(msg ?? ''));
    };
    try {
      const resolved = resolveChoyViewSwitcherOptions([
        { value: 'list', label: 'List View' },
        { value: 'kanban', label: 'Kanban View', hidden: true },
        { value: 'calendar' },
        { value: 'calendar', icon: ExtraIcon, label: 'Calendar' },
      ]);
      expect(resolved.map((o) => o.value)).toEqual(['list', 'calendar']);
      expect(seen.some((m) => m.includes('calendar'))).toBe(true);
    } finally {
      console.warn = warn;
    }
  });

  test('registerChoyViewSwitcherPreset extends builtins', () => {
    registerChoyViewSwitcherPreset('calendar', {
      defaultLabel: 'Calendar',
      icon: ExtraIcon,
    });
    expect(listChoyViewSwitcherPresetIds()).toContain('calendar');
    const resolved = resolveChoyViewSwitcherOptions([{ value: 'calendar' }]);
    expect(resolved).toEqual([
      { value: 'calendar', label: 'Calendar', icon: ExtraIcon, disabled: false },
    ]);
  });
});
