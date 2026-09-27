// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import WelcomeView from './WelcomeView.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

describe('WelcomeView', () => {
  beforeEach(() => {
    stubSfc(ChoyPage as any, {
      props: { title: String, padding: Boolean, width: String },
      setup: ((_props: any, { slots }: any) => {
        return () => h('div', { 'data-test': 'page' }, slots.default?.());
      }) as any,
    } as any);
    stubSfc(ChoyCard as any, {
      props: { title: String },
      setup: ((_props: any, { slots }: any) => {
        return () => h('section', { 'data-test': 'card' }, slots.default?.());
      }) as any,
    } as any);
    stubSfc(ChoyButton as any, {
      props: { type: String, variant: String },
      setup: ((_props: any, { slots, attrs }: any) => {
        return () =>
          h('button', { 'data-test': 'action', type: 'button', ...attrs }, slots.default?.());
      }) as any,
    } as any);
  });

  afterEach(() => {
    restoreSfc(ChoyPage);
    restoreSfc(ChoyCard);
    restoreSfc(ChoyButton);
  });

  test('steps through welcome flow and navigates home on the final action', async () => {
    const stub = createFeStubRouter({
      route: { path: '/welcome', fullPath: '/welcome', params: {}, query: {} },
    });
    const push = fnRecorder((to: unknown) => Promise.resolve(to));
    stub.router.push = push;

    const mounted = mountApp(WelcomeView as any, { plugins: [stub.router] });
    await flushPromises();

    // Step 0: Continue only (no Back).
    let actions = mounted.qa('[data-test=action]');
    expect(actions.length).toBe(1);
    expect(actions[0]!.textContent).toMatch(/Continue/i);
    (actions[0] as HTMLElement).click();
    await flushPromises();

    // Step 1: Back + Continue.
    actions = mounted.qa('[data-test=action]');
    expect(actions.length).toBe(2);
    expect(actions[0]!.textContent).toMatch(/Back/i);
    expect(actions[1]!.textContent).toMatch(/Continue/i);
    (actions[1] as HTMLElement).click();
    await flushPromises();

    // Step 2 (last): Back + Go to home.
    actions = mounted.qa('[data-test=action]');
    expect(actions.length).toBe(2);
    expect(actions[1]!.textContent).toMatch(/home/i);
    (actions[1] as HTMLElement).click();
    expect(push.calls.map(c => c[0])).toEqual(['/home']);

    // Back from last step returns to Continue (not past the start).
    (actions[0] as HTMLElement).click();
    await flushPromises();
    actions = mounted.qa('[data-test=action]');
    expect(actions.some(a => /Continue/i.test(a.textContent || ''))).toBe(true);
    (actions[0] as HTMLElement).click();
    await flushPromises();
    actions = mounted.qa('[data-test=action]');
    expect(actions.length).toBe(1);
    expect(actions[0]!.textContent).toMatch(/Continue/i);

    mounted.unmount();
  });
});
