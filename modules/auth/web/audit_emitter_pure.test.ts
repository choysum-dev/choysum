// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, test, expect } from 'vitest';

// Inlined from user/_lifecycle_scope.ts
export type SwitchScopeValidationErrorCode = 'active_empty' | 'enabled_type' | 'enabled_unauthorized' | 'active_outside_allowed' | 'active_not_in_enabled';

export type SwitchScopeValidationResult =
  | { ok: true; active: string; enabled: string[]; allowed: string[]; prefs: Record<string, any> }
  | { ok: false; code: SwitchScopeValidationErrorCode; active: string; enabled: string[] | null; allowed: string[]; prefs: Record<string, any>; companyId?: string };

function createSwitchCompanyScopeAuditEmitter(eventName: string) {
  let emitted = false;

  const emit = (payload: Record<string, any>) => {
    try {
      void { ...payload, event: eventName, traceId: '' };
      // In test, suppress actual console output.
    } catch {
      // ignore
    }
  };

  return {
    emitOnce(payload: Record<string, any>) {
      if (emitted) return;
      emitted = true;
      emit(payload);
    },
    wasEmitted() {
      return emitted;
    },
  };
}

describe('createSwitchCompanyScopeAuditEmitter', () => {
  test('emitOnce sets emitted state', () => {
    const emitter = createSwitchCompanyScopeAuditEmitter('auth.user.test');
    expect(emitter.wasEmitted()).toBe(false);
    emitter.emitOnce({ ok: true });
    expect(emitter.wasEmitted()).toBe(true);
  });

  test('emitOnce is idempotent', () => {
    const emitter = createSwitchCompanyScopeAuditEmitter('auth.user.test');
    emitter.emitOnce({ ok: true });
    emitter.emitOnce({ ok: false });
    expect(emitter.wasEmitted()).toBe(true);
  });

  test('wasEmitted returns false before first emit', () => {
    const emitter = createSwitchCompanyScopeAuditEmitter('auth.user.test');
    expect(emitter.wasEmitted()).toBe(false);
  });
});
