// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, inject, watch, type PropType } from 'vue';
import { useField } from '@/web/web/composables/useField';
import { mergePreferencesBag, type PreferencesBagDraft } from './preferences_bag';

type FormRootApi = {
  getField: (path: string) => unknown;
  setField: (path: string, value: unknown) => void;
};

/**
 * Registers User.Preferences on the FormView field set and mirrors bagDraft
 * into the draft so Write includes density / notifications / display.
 */
export const PreferenceBagBinder = defineComponent({
  name: 'PreferenceBagBinder',
  props: {
    store: { type: Object as PropType<Record<string, unknown>>, required: true },
    /** Form finished Browse and bagDraft was hydrated from Preferences. */
    active: { type: Boolean, default: false },
    bagDraft: { type: Object as PropType<PreferencesBagDraft>, required: true },
  },
  setup(props) {
    // Register Preferences so Browse/Write include the JSON bag.
    useField({ store: props.store as any, prop: 'Preferences' });
    const formRoot = inject<FormRootApi | null>('form-root', null);

    watch(
      () => [props.active, props.bagDraft] as const,
      () => {
        if (!formRoot || !props.active || !props.bagDraft) return;
        const current = formRoot.getField('Preferences');
        formRoot.setField('Preferences', mergePreferencesBag(current, props.bagDraft));
      },
      { deep: true, immediate: true },
    );

    return () => null;
  },
});
