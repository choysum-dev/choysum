// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { flushPromises } from '@choysum/test-utils';
import { mountApp } from '@/web/web/__tests__/mountApp';
import ChoyDialogHost from './ChoyDialogHost.vue';
import {
  closeDialog,
  openDialog,
  resetDialogServiceForTests,
  useDialogStack,
} from '../../composables/dialogService';

const StubDialog = defineComponent({
  name: 'StubDialog',
  props: {
    modelValue: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'stub-dialog',
          onClick: () => emit('update:modelValue', false),
        },
        String(props.modelValue),
      );
  },
});

describe('ChoyDialogHost', () => {
  beforeEach(() => {
    resetDialogServiceForTests();
  });

  test('renders openDialog entries and closes on modelValue false', async () => {
    const mounted = mountApp(ChoyDialogHost as any);
    const key = openDialog({ component: StubDialog, props: { title: 'x' } });
    await flushPromises();
    await nextTick();
    expect(useDialogStack().value).toHaveLength(1);
    expect(mounted.qa('[data-testid="stub-dialog"]').length).toBe(1);

    (mounted.qa('[data-testid="stub-dialog"]')[0] as HTMLElement).click();
    await flushPromises();
    await nextTick();
    expect(useDialogStack().value.find(e => e.key === key)).toBeUndefined();
    mounted.unmount();
  });

  test('closeDialog from outside unmounts the host entry', async () => {
    const mounted = mountApp(ChoyDialogHost as any);
    const key = openDialog({ component: StubDialog });
    await nextTick();
    closeDialog(key);
    await nextTick();
    expect(mounted.qa('[data-testid="stub-dialog"]').length).toBe(0);
    mounted.unmount();
  });
});
