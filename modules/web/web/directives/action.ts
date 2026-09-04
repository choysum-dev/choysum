// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Directive, DirectiveBinding } from 'vue';
import { normalizeOptionalString } from '@/core/service/utils/normalization';

type BoolOperator = 'or' | 'and';
type ActionMode = 'hide' | 'disable';

/**
 * Checks whether a resource action is allowed.
 */
type ActionChecker = (resourceId: string | undefined) => boolean;

let globalActionChecker: ActionChecker | undefined;

/**
 * Installs the default permission checker used by the action directive.
 */
export function setGlobalActionChecker(checker: ActionChecker | undefined): void {
  globalActionChecker = checker;
}

/**
 * Object syntax supported by the action directive.
 */
export interface ActionBindingObject {
  ids?: string | string[];
  hasAction?: ActionChecker;
  mode?: ActionMode;
  operator?: BoolOperator;
}

/**
 * Accepted directive binding values for action permission control.
 */
export type ActionBindingValue = string | string[] | ActionBindingObject;

interface NormalizedBinding {
  ids: string[];
  hasAction?: ActionChecker;
  mode: ActionMode;
  operator: BoolOperator;
}

const PREV_DISPLAY_KEY = '__choysum_v_action_prev_display__';

/**
 * Normalizes a single id or id list into trimmed action ids.
 */
function normalizeIds(ids: string | string[] | undefined): string[] {
  if (!ids) return [];
  const list = Array.isArray(ids) ? ids : [ids];
  return list.map(v => normalizeOptionalString(v)).filter((v): v is string => Boolean(v));
}

/**
 * Normalizes a directive binding into the internal evaluation shape.
 */
function normalizeBinding(binding: DirectiveBinding<ActionBindingValue>): NormalizedBinding {
  const isObjectValue = typeof binding.value === 'object' && binding.value !== null && !Array.isArray(binding.value);
  const obj = (isObjectValue ? binding.value : null) as ActionBindingObject | null;

  const ids = normalizeIds(obj?.ids ?? (Array.isArray(binding.value) || typeof binding.value === 'string' ? binding.value : undefined));
  const hasAction = obj?.hasAction ?? globalActionChecker;

  const mode: ActionMode = obj?.mode ?? (binding.modifiers.disable ? 'disable' : 'hide');
  const operator: BoolOperator = obj?.operator ?? (binding.modifiers.and ? 'and' : 'or');

  return { ids, hasAction, mode, operator };
}

/**
 * Evaluates whether the binding grants access to the current element.
 */
function evaluatePermission(normalized: NormalizedBinding): boolean {
  const { ids, hasAction, operator } = normalized;

  if (ids.length === 0) {
    return true;
  }
  if (typeof hasAction !== 'function') {
    return false;
  }

  if (operator === 'and') {
    return ids.every(id => hasAction(id));
  }
  return ids.some(id => hasAction(id));
}

/**
 * Applies hide-mode visibility to an element.
 */
function applyHide(el: HTMLElement, allowed: boolean): void {
  const anyEl = el as any;
  if (allowed) {
    if (typeof anyEl[PREV_DISPLAY_KEY] === 'string') {
      el.style.display = anyEl[PREV_DISPLAY_KEY];
      delete anyEl[PREV_DISPLAY_KEY];
    } else {
      el.style.removeProperty('display');
    }
    return;
  }

  if (typeof anyEl[PREV_DISPLAY_KEY] !== 'string') {
    anyEl[PREV_DISPLAY_KEY] = el.style.display;
  }
  el.style.display = 'none';
}

/**
 * Applies disable-mode state to an element.
 */
function applyDisable(el: HTMLElement, allowed: boolean): void {
  const disabled = !allowed;

  if (el instanceof HTMLButtonElement || el instanceof HTMLInputElement || el instanceof HTMLSelectElement || el instanceof HTMLTextAreaElement) {
    el.disabled = disabled;
  }

  if (disabled) {
    el.setAttribute('aria-disabled', 'true');
    el.style.pointerEvents = 'none';
  } else {
    el.removeAttribute('aria-disabled');
    el.style.removeProperty('pointer-events');
  }
}

/**
 * Evaluates and applies the action permission result to an element.
 */
function applyActionPermission(el: HTMLElement, binding: DirectiveBinding<ActionBindingValue>): void {
  const normalized = normalizeBinding(binding);
  const allowed = evaluatePermission(normalized);
  if (normalized.mode === 'disable') {
    applyDisable(el, allowed);
    return;
  }
  applyHide(el, allowed);
}

/**
 * Directive that hides or disables elements based on action permissions.
 */
export const vAction: Directive<HTMLElement, ActionBindingValue> = {
  mounted(el, binding) {
    applyActionPermission(el, binding);
  },
  updated(el, binding) {
    applyActionPermission(el, binding);
  },
};
