// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type CommitStrategy = 'live' | 'blur' | 'idle';

export interface BufferedOptions<T> {
  strategy: CommitStrategy;
  idleDelay?: number; // Optional with a default fallback
  normalize?: (v: T | null) => T | null;
  equals?: (a: T | null, b: T | null) => boolean;
  onCommit?: (changed: boolean, value: T | null) => void;
  commitOnBlur?: boolean; // Whether to commit on blur, defaults to true
}

import { ref, watch, onBeforeUnmount } from 'vue';

export function useBufferedCommit<T>(modelGetter: () => T | null, modelSetter: (v: T | null) => void, opts: BufferedOptions<T>) {
  const editingValue = ref<T | null>(modelGetter());
  const dirty = ref(false);
  let timer: number | null = null;

  const strategy = opts.strategy;
  const idleDelay = opts.idleDelay ?? 360;
  const commitOnBlur = opts.commitOnBlur !== false; // Defaults to true
  const eq = opts.equals || ((a, b) => a === b);
  const norm = opts.normalize || ((v: T | null) => v);

  function scheduleIdle() {
    if (strategy !== 'idle') return; // Guard against non-idle strategies
    if (timer) clearTimeout(timer);
    timer = window.setTimeout(commit, idleDelay);
  }

  function setEditing(v: T | null) {
    editingValue.value = v;
    dirty.value = true;
    if (strategy === 'live') {
      commit(); // Commit immediately
    } else if (strategy === 'idle') {
      scheduleIdle();
    }
  }

  function commit(): { changed: boolean; value: T | null } {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (!dirty.value) {
      const current = modelGetter();
      return { changed: false, value: current };
    }
    const prev = modelGetter();
    const next = norm(editingValue.value);
    const changed = !eq(prev, next);
    if (changed) {
      modelSetter(next);
    }
    // Keep the editing buffer aligned because normalize may change the value.
    editingValue.value = next;
    dirty.value = false;
    opts.onCommit?.(changed, next);
    return { changed, value: next };
  }

  function onBlur() {
    if (commitOnBlur) commit();
  }

  // Mirror external model updates only when the local buffer is not dirty.
  watch(
    () => modelGetter(),
    v => {
      if (!dirty.value) editingValue.value = v;
    }
  );

  onBeforeUnmount(() => {
    if (timer) clearTimeout(timer);
    if (dirty.value) commit(); // Flush pending edits before unmount
  });

  return {
    editingValue,
    dirty,
    setEditing,
    onBlur,
    commit,
  };
}
