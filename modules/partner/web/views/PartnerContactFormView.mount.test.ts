// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import PartnerContactFormView from './PartnerContactFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string } };
  };
};

test('PartnerContactFormView openRelatedRecord on address value click', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter();
  const pushes: unknown[] = [];
  const orig = router.push.bind(router);
  router.push = (to: unknown) => {
    pushes.push(to);
    return orig(to);
  };

  const store = { $id: 'fe-stub-contact-store', records: {}, modelName: 'partner.Contact' };
  const mounted = mountApp(PartnerContactFormView as any, {
    props: { store, viewMode: 'create' },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const ss = mounted.setupState() as {
    onAddressValueClick?: (payload: { id?: string }) => void;
  };
  ss.onAddressValueClick?.({ id: '' });
  ss.onAddressValueClick?.({ id: 'addr-1' });
  await flushPromises();
  expect(pushes.some(p => JSON.stringify(p).includes('addr-1'))).toBe(true);
  mounted.unmount();
});
