// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, type Ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import type { OnchangeFlushPayload } from '@/web/web/composables/useOnchange';

/**
 * useOnchangeAggregation
 * Encapsulates merging condition/selection from onchange responses,
 * field-level error tracking with alias expansion, and global messaging.
 */
export function useOnchangeAggregation(options?: { showMessages?: boolean }) {
  const showMessages = options?.showMessages ?? true;

  const lastOnchangeResult: Ref<any | null> = ref(null);
  const fieldErrors: Ref<Map<string, string>> = ref(new Map());

  // Generate last-level aliases for nested selectors to help error mapping
  function expandLastLevelAliases(field: string): string[] {
    const ids: string[] = [];
    const strippedIds = field.replace(/\(id=([^)]+)\)/g, (_m, g1) => {
      ids.push(String(g1));
      return '';
    });
    const idxes: number[] = [];
    const noSel = strippedIds.replace(/\[(\d+)\]/g, (_m, g1) => {
      idxes.push(Number(g1));
      return '';
    });

    const base = noSel.replace(/\.+/g, '.').replace(/\.$/, '');
    const segs = base.split('.').filter(Boolean);
    if (segs.length < 2) return [];

    const lastCollectionIdx = segs.length - 2;
    const lastCollection = segs[lastCollectionIdx];

    const out: string[] = [];

    const lastId = ids.length ? ids[ids.length - 1] : null;
    if (lastId) {
      const withLastId = [...segs.slice(0, lastCollectionIdx), `${lastCollection}(id=${lastId})`, ...segs.slice(lastCollectionIdx + 1)].join('.');
      out.push(withLastId);
    }

    const lastIdx = idxes.length ? idxes[idxes.length - 1] : null;
    if (lastIdx != null) {
      const withLastIdx = [...segs.slice(0, lastCollectionIdx), `${lastCollection}[${lastIdx}]`, ...segs.slice(lastCollectionIdx + 1)].join('.');
      out.push(withLastIdx);
    }

    const hadSelector = /\(id=|\[\d+\]/.test(field);
    if (!hadSelector) out.push(base);

    return out;
  }

  const afterFlushHandler = async (p: OnchangeFlushPayload) => {
    const res = (p as any)?.result ?? {};
    const changed: string[] = Array.isArray((p as any)?.changed) ? (p as any).changed : [];

    const msgs = Array.isArray(res?.messages) ? res.messages : [];

    // 1) Merge conditions
    const newConditions: Array<{ field: string; condition: any }> = Array.isArray(res?.condition) ? res.condition : [];
    const oldConditions: Array<{ field: string; condition: any }> = Array.isArray(lastOnchangeResult.value?.condition)
      ? lastOnchangeResult.value.condition
      : [];

    const fieldCondMap = new Map<string, any>();
    for (const old of oldConditions) {
      if (old && typeof old.field === 'string') fieldCondMap.set(old.field, old.condition);
    }
    for (const c of newConditions) {
      if (!c || typeof c.field !== 'string') continue;
      if (c.condition === null || c.condition === undefined) fieldCondMap.delete(c.field);
      else fieldCondMap.set(c.field, c.condition);
    }
    const mergedConditions: Array<{ field: string; condition: any }> = [];
    for (const [field, condition] of fieldCondMap.entries()) mergedConditions.push({ field, condition });

    // 2) Merge selections (accumulative)
    const newSelections: Array<{ field: string; selection: string[]; disabled?: string[] }> = Array.isArray(res?.selection) ? res.selection : [];
    const oldSelections: Array<{ field: string; selection: string[]; disabled?: string[] }> = Array.isArray(lastOnchangeResult.value?.selection)
      ? lastOnchangeResult.value.selection
      : [];

    const fieldSelMap = new Map<string, { selection: string[]; disabled?: string[] }>();
    for (const old of oldSelections) {
      if (old && typeof old.field === 'string') fieldSelMap.set(old.field, { selection: old.selection || [], disabled: old.disabled || undefined });
    }
    for (const s of newSelections) {
      if (!s || typeof s.field !== 'string') continue;
      if (!s.selection || (Array.isArray(s.selection) && s.selection.length === 0)) fieldSelMap.delete(s.field);
      else fieldSelMap.set(s.field, { selection: s.selection, disabled: s.disabled || undefined });
    }
    const mergedSelections: Array<{ field: string; selection: string[]; disabled?: string[] }> = [];
    for (const [field, { selection, disabled }] of fieldSelMap.entries()) mergedSelections.push({ field, selection, disabled });

    // 3) Persist accumulated result (preserve other fields in res)
    lastOnchangeResult.value = {
      ...(lastOnchangeResult.value || {}),
      ...(res || {}),
      condition: mergedConditions,
      selection: mergedSelections,
    };

    // 4) Field-level errors with aliasing
    const newPairs: Array<{ key: string; msg: string }> = [];
    for (const m of msgs) {
      if (m?.level === 'error' && m?.field) {
        const field = String(m.field);
        const message = String(m.message ?? '');
        newPairs.push({ key: field, msg: message });
        for (const alias of expandLastLevelAliases(field)) newPairs.push({ key: alias, msg: message });
      }
    }
    const newKeys = new Set(newPairs.map(x => x.key));

    const clearSet = new Set<string>();
    for (const c of changed) {
      clearSet.add(String(c));
      for (const alias of expandLastLevelAliases(String(c))) clearSet.add(alias);
    }
    if (fieldErrors.value.size && clearSet.size) {
      for (const k of Array.from(fieldErrors.value.keys())) {
        if (clearSet.has(k) && !newKeys.has(k)) fieldErrors.value.delete(k);
      }
    }
    if (newPairs.length) {
      for (const { key, msg } of newPairs) fieldErrors.value.set(key, msg);
    }

    // 5) Global messages
    if (!msgs.length || !showMessages) return;

    const blockingMsg = msgs.find((m: any) => m.blocking);
    if (blockingMsg) {
      try {
        await ElMessageBox.confirm(blockingMsg.message, blockingMsg.title || '提示', {
          confirmButtonText: '继续',
          cancelButtonText: '取消',
          type: blockingMsg.level === 'error' ? 'error' : 'warning',
          distinguishCancelAndClose: true,
          closeOnClickModal: false,
        });
      } catch (action) {
        if (action === 'cancel') ElMessage.info('已取消操作');
      }
      return;
    }

    const firstMsg =
      msgs.find((m: any) => !m.blocking && m.level === 'error' && !m.field) ||
      msgs.find((m: any) => !m.blocking && m.level === 'warn') ||
      msgs.find((m: any) => !m.blocking && m.level === 'info');

    if (firstMsg) {
      const text = firstMsg.field ? `${firstMsg.field}: ${firstMsg.message}` : firstMsg.message;
      const typeMap = {
        error: () => ElMessage.error({ message: text, duration: 5000, showClose: true, grouping: true }),
        warn: () => ElMessage.warning({ message: text, duration: 3000, showClose: true, grouping: true }),
        info: () => ElMessage.info({ message: text, duration: 2000, showClose: true, grouping: true }),
      } as const;
      // @ts-expect-error index signature constrained to known keys
      typeMap[firstMsg.level]?.();
    }
  };

  function reset() {
    lastOnchangeResult.value = null;
    fieldErrors.value.clear();
  }

  return { lastOnchangeResult, fieldErrors, afterFlushHandler, reset };
}
