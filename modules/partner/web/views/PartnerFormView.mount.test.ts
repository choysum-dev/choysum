// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import PartnerFormView from './PartnerFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    currentRoute: { value: { path: string } };
  };
};

test('PartnerFormView openRelatedRecord value-click handlers', async () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const { router } = createFeStubRouter();
  const pushes: unknown[] = [];
  const orig = router.push.bind(router);
  router.push = (to: unknown) => {
    pushes.push(to);
    return orig(to);
  };

  const store = { $id: 'fe-stub-partner-form-store', records: {}, modelName: 'partner.Partner' };
  const mounted = mountApp(PartnerFormView as any, {
    props: { store, viewMode: 'create' },
    plugins: [pinia, router as any],
  });
  await flushPromises();

  const ss = mounted.setupState() as {
    onCompanyValueClick?: (payload: { id?: string }) => void;
    onDefaultContactValueClick?: (payload: { id?: string }) => void;
    onDefaultBillingAddressValueClick?: (payload: { id?: string }) => void;
    onDefaultShippingAddressValueClick?: (payload: { id?: string }) => void;
  };
  ss.onCompanyValueClick?.({ id: '' });
  ss.onCompanyValueClick?.({ id: 'co-1' });
  ss.onDefaultContactValueClick?.({ id: 'ptn-1' });
  ss.onDefaultBillingAddressValueClick?.({ id: 'bill-1' });
  ss.onDefaultShippingAddressValueClick?.({ id: 'ship-1' });
  await flushPromises();

  const blob = JSON.stringify(pushes);
  expect(blob).toContain('co-1');
  expect(blob).toContain('ptn-1');
  expect(blob).toContain('bill-1');
  expect(blob).toContain('ship-1');
  mounted.unmount();
});
