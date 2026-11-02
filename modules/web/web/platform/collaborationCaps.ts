// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { inject, type InjectionKey } from 'vue';
import { listRegisteredModelNames } from '@/web/web/stores/registry';

/** Runtime availability of optional collaboration / attachment models. */
export type CollaborationCaps = {
  message: boolean;
  audit: boolean;
  document: boolean;
};

/** App-level provide key for {@link CollaborationCaps}. */
export const CollaborationCapsKey: InjectionKey<CollaborationCaps> = Symbol('choyCollaborationCaps');

const MESSAGE_MODELS = ['message.Message', 'message.Notification', 'message.Follower'] as const;

/**
 * Resolve collaboration caps from registered store factories.
 * Missing models yield `false` — callers must not dial those modules.
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
 * Inject app-provided caps, or resolve from the current store registry.
 */
export function useCollaborationCaps(): CollaborationCaps {
  return inject(CollaborationCapsKey, null) ?? resolveCollaborationCaps();
}
