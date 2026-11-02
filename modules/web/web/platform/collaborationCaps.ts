// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, inject, type ComputedRef, type InjectionKey } from 'vue';
import {
  getStoreFactoryRegistryVersion,
  listRegisteredModelNames,
} from '@/web/web/stores/registry';

/** Runtime availability of optional collaboration / attachment models. */
export type CollaborationCaps = {
  message: boolean;
  audit: boolean;
  document: boolean;
};

/**
 * Optional test/harness override for {@link useCollaborationCaps} /
 * {@link useReactiveCollaborationCaps}. Product code resolves from the store registry.
 */
export const CollaborationCapsKey: InjectionKey<CollaborationCaps> = Symbol('choyCollaborationCaps');

const MESSAGE_MODELS = ['message.Message', 'message.Notification', 'message.Follower'] as const;

/**
 * Resolve collaboration caps from registered store factories.
 * Missing models yield `false` — callers must not dial those modules.
 *
 * Generated webapistore modules call `registerStoreFactory` at import time, so
 * caps reflect installed modules once those artifacts are loaded.
 */
export function resolveCollaborationCaps(
  registered: readonly string[] = listRegisteredModelNames()
): CollaborationCaps {
  const set = new Set(registered);
  return {
    message: MESSAGE_MODELS.every(name => set.has(name)),
    audit: set.has('audit.FieldChange'),
    document: set.has('document.AttachmentBinding'),
  };
}

/**
 * Snapshot caps: inject override if provided, else resolve from the current registry.
 */
export function useCollaborationCaps(): CollaborationCaps {
  return inject(CollaborationCapsKey, null) ?? resolveCollaborationCaps();
}

/**
 * Reactive caps that re-resolve when store factories are registered or removed.
 */
export function useReactiveCollaborationCaps(): ComputedRef<CollaborationCaps> {
  const override = inject(CollaborationCapsKey, null);
  const version = getStoreFactoryRegistryVersion();
  return computed(() => {
    if (override) return override;
    // Touch registry version so late registerStoreFactory re-resolves caps.
    void (version as { value?: unknown })?.value;
    return resolveCollaborationCaps();
  });
}
