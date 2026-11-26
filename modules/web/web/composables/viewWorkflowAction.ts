// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, type Ref } from 'vue';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';

export type ViewWorkflowErrorMessage = string | ((err: unknown) => string);

export type RunViewWorkflowActionOpts = {
  busy: Ref<boolean>;
  call: () => void | Promise<void>;
  afterSuccess?: () => void | Promise<void>;
  onSuccess?: () => void | Promise<void>;
  errorMessage?: ViewWorkflowErrorMessage;
  fallbackErrorMessage?: string;
};

function resolveErrorMessage(
  err: unknown,
  errorMessage: ViewWorkflowErrorMessage | undefined,
  fallback: string,
): string {
  if (typeof errorMessage === 'function') {
    try {
      const msg = String(errorMessage(err) || '').trim();
      if (msg) return msg;
    } catch {
      // Fall through to default formatting.
    }
  } else if (typeof errorMessage === 'string' && errorMessage.trim()) {
    return errorMessage.trim();
  }
  if (err instanceof Error && err.message.trim()) return err.message;
  const raw = String((err as { message?: unknown })?.message ?? err ?? '').trim();
  return raw || fallback;
}

function toastError(message: string): void {
  try {
    ChoyMessage.error(message);
  } catch {
    // Toast host may be absent in FE unit tests.
  }
}

/**
 * Shared busy + call + optional surface sync + error toast for Form/List workflow CTAs.
 * Kit Action buttons stay presentation-only; hosts pass `run(() => store.…)` here.
 */
export async function runViewWorkflowAction(opts: RunViewWorkflowActionOpts): Promise<void> {
  if (opts.busy.value) return;
  opts.busy.value = true;
  try {
    await opts.call();
    if (opts.afterSuccess) await opts.afterSuccess();
    if (opts.onSuccess) await opts.onSuccess();
  } catch (err) {
    toastError(
      resolveErrorMessage(err, opts.errorMessage, opts.fallbackErrorMessage || 'Action failed'),
    );
  } finally {
    opts.busy.value = false;
  }
}

export type FormWorkflowReload = 'display' | 'none';

export type FormWorkflowFormApi = {
  reloadDisplay?: () => void | Promise<void>;
  refresh?: () => void | Promise<void>;
  getViewMode?: () => string | null | undefined;
};

export type UseFormWorkflowActionOpts = {
  formRef: Ref<FormWorkflowFormApi | null | undefined>;
  /** display → reloadDisplay (edit/create no-op); none → caller owns sync. Default display. */
  reload?: FormWorkflowReload;
  onSuccess?: () => void | Promise<void>;
  errorMessage?: ViewWorkflowErrorMessage;
};

/**
 * Form domain workflow (statusbar): RPC then optional silent display reload.
 * Does not replace platform Overflow Refresh (dirty gate + toast).
 */
export function useFormWorkflowAction(opts: UseFormWorkflowActionOpts): {
  busy: Ref<boolean>;
  run: (call: () => void | Promise<void>) => Promise<void>;
} {
  const busy = ref(false);
  const reload = opts.reload ?? 'display';

  async function afterSuccess(): Promise<void> {
    if (reload === 'none') return;
    const form = opts.formRef.value;
    const mode = form?.getViewMode?.();
    if (mode === 'edit' || mode === 'create') return;
    if (typeof form?.reloadDisplay === 'function') {
      await form.reloadDisplay();
      return;
    }
    if (typeof form?.refresh === 'function') {
      await form.refresh();
    }
  }

  async function run(call: () => void | Promise<void>): Promise<void> {
    await runViewWorkflowAction({
      busy,
      call,
      afterSuccess,
      onSuccess: opts.onSuccess,
      errorMessage: opts.errorMessage,
      fallbackErrorMessage: 'Action failed',
    });
  }

  return { busy, run };
}

export type ListWorkflowReload = 'apply' | 'none';

export type ListWorkflowListApi = {
  softRefresh?: () => void | Promise<void>;
  clearSelection?: () => void;
  /** Fallback when softRefresh is missing (clears selection via load). */
  load?: () => void | Promise<void>;
  refresh?: () => void | Promise<void>;
};

export type UseListWorkflowActionOpts = {
  listRef: Ref<ListWorkflowListApi | null | undefined>;
  /** apply → softRefresh; none → caller owns sync. Default apply. */
  reload?: ListWorkflowReload;
  /** After successful apply, clear row selection (bulk demos). Default false. */
  clearSelection?: boolean;
  onSuccess?: () => void | Promise<void>;
  errorMessage?: ViewWorkflowErrorMessage;
};

/**
 * List domain workflow (#contextual-actions): RPC then optional silent softRefresh.
 * Platform Delete/Overflow Refresh keep their own loadData / toast paths.
 */
export function useListWorkflowAction(opts: UseListWorkflowActionOpts): {
  busy: Ref<boolean>;
  run: (call: () => void | Promise<void>) => Promise<void>;
} {
  const busy = ref(false);
  const reload = opts.reload ?? 'apply';

  async function afterSuccess(): Promise<void> {
    if (reload === 'none') return;
    const list = opts.listRef.value;
    if (typeof list?.softRefresh === 'function') {
      await list.softRefresh();
    } else if (typeof list?.load === 'function') {
      await list.load();
    } else if (typeof list?.refresh === 'function') {
      await list.refresh();
    }
    if (opts.clearSelection && typeof list?.clearSelection === 'function') {
      list.clearSelection();
    }
  }

  async function run(call: () => void | Promise<void>): Promise<void> {
    await runViewWorkflowAction({
      busy,
      call,
      afterSuccess,
      onSuccess: opts.onSuccess,
      errorMessage: opts.errorMessage,
      fallbackErrorMessage: 'Action failed',
    });
  }

  return { busy, run };
}
