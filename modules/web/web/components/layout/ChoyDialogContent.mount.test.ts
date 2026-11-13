// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises } from '@choysum/test-utils';
import { mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyDialogContent from './ChoyDialogContent.vue';
import VendorDialogContent from '../vendor/ui/dialog/DialogContent.vue';

describe('ChoyDialogContent', () => {
  afterEach(() => {
    restoreSfc(VendorDialogContent as any);
  });

  test('forwards dismissible=false into escape/outside lock handlers', async () => {
    const events: string[] = [];
    stubSfc(VendorDialogContent as any, {
      name: 'VendorDialogContent',
      inheritAttrs: false,
      props: {
        showCloseButton: { type: Boolean, default: true },
        class: { type: [String, Object, Array], default: undefined },
      },
      emits: ['escape-key-down', 'pointer-down-outside', 'focus-outside', 'interact-outside'],
      setup(_p: any, { slots, emit, attrs }: any) {
        return () =>
          h(
            'div',
            {
              'data-testid': 'vendor-dialog-content',
              ...attrs,
              onClick: () => {
                const ev = {
                  defaultPrevented: false,
                  preventDefault() {
                    this.defaultPrevented = true;
                  },
                };
                emit('escape-key-down', ev);
                events.push(ev.defaultPrevented ? 'blocked' : 'allowed');
              },
            },
            slots.default?.(),
          );
      },
    });

    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyDialogContent as any, { dismissible: false }, { default: () => 'locked-body' });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    expect(mounted.qa('[data-testid="vendor-dialog-content"]').length).toBe(1);
    (mounted.qa('[data-testid="vendor-dialog-content"]')[0] as HTMLElement).click();
    expect(events).toEqual(['blocked']);
    mounted.unmount();
  });

  test('dismissible true does not preventDefault on escape attempt', async () => {
    const events: string[] = [];
    stubSfc(VendorDialogContent as any, {
      name: 'VendorDialogContent',
      inheritAttrs: false,
      props: {
        showCloseButton: { type: Boolean, default: true },
        class: { type: [String, Object, Array], default: undefined },
      },
      emits: ['escape-key-down', 'pointer-down-outside', 'focus-outside', 'interact-outside'],
      setup(_p: any, { slots, emit }: any) {
        return () =>
          h(
            'div',
            {
              'data-testid': 'vendor-dialog-content-ok',
              onClick: () => {
                const ev = {
                  defaultPrevented: false,
                  preventDefault() {
                    this.defaultPrevented = true;
                  },
                };
                emit('escape-key-down', ev);
                events.push(ev.defaultPrevented ? 'blocked' : 'allowed');
              },
            },
            slots.default?.(),
          );
      },
    });

    const Host = defineComponent({
      setup() {
        return () =>
          h(ChoyDialogContent as any, { dismissible: true }, { default: () => 'ok-body' });
      },
    });
    const mounted = mountApp(Host as any);
    await flushPromises();
    (mounted.qa('[data-testid="vendor-dialog-content-ok"]')[0] as HTMLElement).click();
    expect(events).toEqual(['allowed']);
    mounted.unmount();
  });
});
