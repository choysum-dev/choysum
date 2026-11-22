// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import PartnerIdentifierFormView from './PartnerIdentifierFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string } };
  };
};

test('PartnerIdentifierFormView mount coverage', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter();
  const pushes: unknown[] = [];
  const orig = router.push.bind(router);
  router.push = (to: unknown) => {
    pushes.push(to);
    return orig(to);
  };

  const store = { $id: 'fe-stub-id-store', records: {}, modelName: 'partner.Identifier' };
  const mounted = mountApp(PartnerIdentifierFormView as any, {
    props: { store, viewMode: 'create' },
    plugins: [pinia, router as any],
  });
  await flushPromises();
  expect(mounted.el != null).toBe(true);

  const ss = mounted.setupState() as {
    onCountryValueClick?: (payload: { id?: string }) => void;
  };
  ss.onCountryValueClick?.({ id: '' });
  ss.onCountryValueClick?.({ id: 'country-1' });
  await flushPromises();
  expect(pushes.some(p => JSON.stringify(p).includes('country-1'))).toBe(true);

  mounted.unmount();
});
