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
 * Fill empty LanguageId / Timezone / Theme / Density on the FormView draft from hints.
 * Does not Write until the user submits.
 */
export function seedSelfProfileDraft(
  formRoot: FormRootApi | null | undefined,
  opts: { languageId?: unknown; timezone?: string; theme?: string; density?: string },
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
  const theme = String(opts.theme || '').trim();
  if (isEmptyField(formRoot.getField('Theme')) && (theme === 'light' || theme === 'dark' || theme === 'auto')) {
    formRoot.setField('Theme', theme);
  }
  const density = String(opts.density || '').trim();
  if (
    isEmptyField(formRoot.getField('Density')) &&
    (density === 'comfortable' || density === 'compact')
  ) {
    formRoot.setField('Density', density);
  }
}

/**
 * After FormView beginEdit, seed empty self-profile fields once each seed value arrives.
 * Seeds are tracked separately so a ready form that settles before resolution still
 * receives late-arriving hints.
 */
export const SelfProfileDraftSeeder = defineComponent({
  name: 'SelfProfileDraftSeeder',
  props: {
    ready: { type: Boolean, default: false },
    languageId: { type: String, default: '' },
    timezone: { type: String, default: '' },
    theme: { type: String, default: '' },
    density: { type: String, default: '' },
  },
  setup(props) {
    const formRoot = inject<FormRootApi | null>('form-root', null);
    let languageApplied = false;
    let timezoneApplied = false;
    let themeApplied = false;
    let densityApplied = false;

    function apply(): void {
      if (!formRoot || !props.ready) return;
      if (!languageApplied && props.languageId) {
        seedSelfProfileDraft(formRoot, { languageId: props.languageId });
        languageApplied = true;
      }
      if (!timezoneApplied && props.timezone) {
        seedSelfProfileDraft(formRoot, { timezone: props.timezone });
        timezoneApplied = true;
      }
      if (!themeApplied && props.theme) {
        seedSelfProfileDraft(formRoot, { theme: props.theme });
        themeApplied = true;
      }
      if (!densityApplied && props.density) {
        seedSelfProfileDraft(formRoot, { density: props.density });
        densityApplied = true;
      }
    }

    watch(
      () =>
        [props.ready, props.languageId, props.timezone, props.theme, props.density] as const,
      () => apply(),
      { immediate: true },
    );
    return () => null;
  },
});
