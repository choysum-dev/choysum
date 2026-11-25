// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, type ComputedRef, type Ref } from 'vue';

import { confirmChoyAction } from './confirmChoyAction';
import { getGlobalActionChecker } from '../directives/action';
import { isConnectivityUnreachable } from '../stores/connectivityStore';

export type ChoyActionMode = 'hide' | 'disable';

export type ChoyActionConfirm =
  | string
  | {
      message: string;
      title?: string;
      destructive?: boolean;
      confirmText?: string;
      cancelText?: string;
    };

/** Shared Action-line inputs (button and menu-item skins). */
export type ChoyActionPolicyInput = {
  disabled?: boolean;
  busy?: boolean;
  actionId?: string;
  hasAction?: (id: string | undefined) => boolean;
  actionMode?: ChoyActionMode;
  requiresConnectivity?: boolean;
  confirm?: ChoyActionConfirm;
  run?: () => void | Promise<void>;
};

export type ChoyActionActivateOptions = {
  /**
   * When true, call stopPropagation on the activation event.
   * Button hosts pass true for as=button (or whenever disabled); menu items usually omit.
   */
  stopPropagation?: boolean;
};

/**
 * ACL / connectivity / confirm / busy / run|execute policy shared by
 * ChoyActionButton and ChoyActionMenuItem. Presentation-agnostic.
 */
export function useChoyActionPolicy(
  props: ChoyActionPolicyInput,
  emitExecute: () => void,
): {
  pending: Ref<boolean>;
  aclVisible: ComputedRef<boolean>;
  offlineBlocked: ComputedRef<boolean>;
  effectiveDisabled: ComputedRef<boolean>;
  isActionAllowedNow: () => boolean;
  activate: (event?: Event | Partial<Event>, opts?: ChoyActionActivateOptions) => Promise<void>;
} {
  const pending = ref(false);

  const actionIdTrimmed = computed(() => String(props.actionId ?? '').trim());

  function isActionAllowedNow(): boolean {
    const id = actionIdTrimmed.value;
    if (!id) return true;
    const checker = props.hasAction ?? getGlobalActionChecker();
    if (typeof checker !== 'function') return true;
    return !!checker(id);
  }

  const aclAllowed = computed(() => isActionAllowedNow());

  /** Hide mode + denied → do not mount; disable mode keeps the control. */
  const aclVisible = computed(() => {
    if (!actionIdTrimmed.value) return true;
    if (aclAllowed.value) return true;
    return props.actionMode === 'disable';
  });

  const offlineBlocked = computed(
    () => props.requiresConnectivity !== false && isConnectivityUnreachable(),
  );

  const effectiveDisabled = computed(
    () =>
      !!props.disabled ||
      !!props.busy ||
      pending.value ||
      offlineBlocked.value ||
      (!!actionIdTrimmed.value && !aclAllowed.value && props.actionMode === 'disable'),
  );

  async function activate(event?: Event | Partial<Event>, opts?: ChoyActionActivateOptions) {
    const e = event as Partial<Event> | undefined;
    if (opts?.stopPropagation || effectiveDisabled.value) {
      e?.stopPropagation?.();
    }
    // Live ACL: hide-mode denials leave effectiveDisabled false (child unmounted),
    // but exposed onClick/onSelect must still reject programmatic activate.
    if (effectiveDisabled.value || !isActionAllowedNow()) {
      e?.preventDefault?.();
      return;
    }

    if (props.confirm != null) {
      // Block native submit / menu close side-effects until the user answers.
      e?.preventDefault?.();
      const conf = props.confirm;
      const message = typeof conf === 'string' ? conf : conf.message;
      const title = typeof conf === 'string' ? undefined : conf.title;
      const destructive = typeof conf === 'string' ? undefined : conf.destructive;
      const confirmText = typeof conf === 'string' ? undefined : conf.confirmText;
      const cancelText = typeof conf === 'string' ? undefined : conf.cancelText;
      try {
        await confirmChoyAction(message, title || 'Confirm', {
          destructive,
          confirmText,
          cancelText,
        });
      } catch {
        return;
      }
      // Re-check live (do not trust cached ACL computed; checker may flip during await).
      if (
        !!props.disabled ||
        !!props.busy ||
        offlineBlocked.value ||
        !isActionAllowedNow()
      ) {
        return;
      }
    }

    pending.value = true;
    try {
      if (typeof props.run === 'function') {
        await props.run();
      } else {
        emitExecute();
      }
    } finally {
      pending.value = false;
    }
  }

  return {
    pending,
    aclVisible,
    offlineBlocked,
    effectiveDisabled,
    isActionAllowedNow,
    activate,
  };
}
