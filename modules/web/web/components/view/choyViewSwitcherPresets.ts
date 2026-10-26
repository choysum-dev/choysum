// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Component } from 'vue';
import { BarChart3, LayoutGrid, List } from 'lucide-vue-next';

/**
 * Built-in switcher kinds used by action-bar view toggles.
 * Values are UI/route keys (lowercase), not ViewKindId (`List` / `Kanban`).
 * Register more via registerChoyViewSwitcherPreset when a new chrome view ships.
 */
export type ChoyViewSwitcherKind = 'list' | 'kanban' | 'board' | 'chart' | (string & {});

export type ChoyViewSwitcherPreset = {
  /** Default English tip/aria label when a caller omits `label`. */
  defaultLabel: string;
  icon: Component;
};

export type ChoyViewSwitcherOption = {
  /** Stable kind / route key (e.g. list, kanban, chart). */
  value: ChoyViewSwitcherKind;
  /** Hover tip + button aria-label (prefer `_t(...)` at the call site). */
  label?: string;
  /** Override preset icon; required when `value` has no preset. */
  icon?: Component;
  disabled?: boolean;
  /** Dropped from the rendered group (permission / feature flags). */
  hidden?: boolean;
};

/** Resolved option ready for the button group (icon + label required). */
export type ChoyViewSwitcherResolvedOption = {
  value: string;
  label: string;
  icon: Component;
  disabled: boolean;
};

const presets = new Map<string, ChoyViewSwitcherPreset>([
  ['list', { defaultLabel: 'List', icon: List }],
  // Card/lane boards share the grid glyph; meta calls this "board".
  ['kanban', { defaultLabel: 'Kanban', icon: LayoutGrid }],
  ['board', { defaultLabel: 'Board', icon: LayoutGrid }],
  ['chart', { defaultLabel: 'Chart', icon: BarChart3 }],
]);

/** Built-in preset ids (list / kanban / board / chart, plus any registered). */
export function listChoyViewSwitcherPresetIds(): string[] {
  return [...presets.keys()];
}

/** Register or replace a switcher preset for a new view kind. */
export function registerChoyViewSwitcherPreset(id: string, preset: ChoyViewSwitcherPreset): void {
  const key = String(id || '').trim();
  if (!key) {
    console.warn('[choyViewSwitcher] registerChoyViewSwitcherPreset ignored empty id');
    return;
  }
  presets.set(key, preset);
}

/** Look up a preset by kind id (case-sensitive). */
export function getChoyViewSwitcherPreset(id: string): ChoyViewSwitcherPreset | undefined {
  return presets.get(String(id || '').trim());
}

/**
 * Fill icons/labels from presets, drop `hidden`, and keep caller order.
 * Options without a preset must supply `icon` (and usually `label`).
 */
export function resolveChoyViewSwitcherOptions(
  options: readonly ChoyViewSwitcherOption[],
): ChoyViewSwitcherResolvedOption[] {
  const out: ChoyViewSwitcherResolvedOption[] = [];
  for (const opt of options) {
    if (opt.hidden) continue;
    const value = String(opt.value ?? '').trim();
    if (!value) continue;
    const preset = presets.get(value);
    const icon = opt.icon ?? preset?.icon;
    if (!icon) {
      console.warn(
        `[choyViewSwitcher] option "${value}" has no icon; pass icon or registerChoyViewSwitcherPreset`,
      );
      continue;
    }
    const label = String(opt.label ?? preset?.defaultLabel ?? value).trim() || value;
    out.push({
      value,
      label,
      icon,
      disabled: opt.disabled === true,
    });
  }
  return out;
}

/** Test helper: restore builtin presets only. */
export function resetChoyViewSwitcherPresetsForTests(): void {
  presets.clear();
  presets.set('list', { defaultLabel: 'List', icon: List });
  presets.set('kanban', { defaultLabel: 'Kanban', icon: LayoutGrid });
  presets.set('board', { defaultLabel: 'Board', icon: LayoutGrid });
  presets.set('chart', { defaultLabel: 'Chart', icon: BarChart3 });
}
