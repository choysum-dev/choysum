// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, inject, watch } from 'vue';

type FormRootApi = {
  getField: (path: string) => unknown;
  setField: (path: string, value: unknown) => void;
};

function isEmptyField(value: unknown): boolean {
  if (value == null) return true;
  if (typeof value === 'string') return !value.trim();
  if (typeof value === 'object' && 'Id' in value) {
    return !String((value as { Id?: unknown }).Id ?? '').trim();
  }
  return false;
}

/**
 * Fill empty LanguageId / Timezone on the FormView draft from session / browser hints.
 * Does not Write until the user submits.
 */
export function seedPreferenceDraft(
  formRoot: FormRootApi | null | undefined,
  opts: { languageId?: unknown; timezone?: string },
): void {
  if (!formRoot) return;
  const languageId = opts.languageId;
  if (isEmptyField(formRoot.getField('LanguageId')) && !isEmptyField(languageId)) {
    formRoot.setField('LanguageId', languageId);
  }
  const timezone = String(opts.timezone || '').trim();
  if (isEmptyField(formRoot.getField('Timezone')) && timezone) {
    formRoot.setField('Timezone', timezone);
  }
}

/**
 * After FormView beginEdit, seed empty preference fields once each seed value arrives.
 * Language and timezone are tracked separately so a ready form that settles before
 * seed resolution still receives late-arriving hints.
 */
export const PreferenceDraftSeeder = defineComponent({
  name: 'PreferenceDraftSeeder',
  props: {
    ready: { type: Boolean, default: false },
    languageId: { type: String, default: '' },
    timezone: { type: String, default: '' },
  },
  setup(props) {
    const formRoot = inject<FormRootApi | null>('form-root', null);
    let languageApplied = false;
    let timezoneApplied = false;

    function apply(): void {
      if (!formRoot || !props.ready) return;
      if (!languageApplied && props.languageId) {
        seedPreferenceDraft(formRoot, { languageId: props.languageId });
        languageApplied = true;
      }
      if (!timezoneApplied && props.timezone) {
        seedPreferenceDraft(formRoot, { timezone: props.timezone });
        timezoneApplied = true;
      }
    }

    watch(
      () => [props.ready, props.languageId, props.timezone] as const,
      () => apply(),
      { immediate: true },
    );
    return () => null;
  },
});
