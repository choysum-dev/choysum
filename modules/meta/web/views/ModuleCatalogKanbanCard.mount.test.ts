// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import ModuleCatalogKanbanCard from './ModuleCatalogKanbanCard.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

function makeCard(payload: Record<string, unknown>) {
  return {
    id: String(payload.Id || payload.ModuleName || 'card'),
    title: String(payload.ModuleName || 'card'),
    laneKey: 'all',
    payload,
  };
}

const global = buildPageMountGlobal({
  route: { path: '/meta/modules', fullPath: '/meta/modules', name: 'MetaModuleCatalog' },
});

test('ModuleCatalogKanbanCard.vue shows author, license, summary, and install CTA', async () => {
  const wrapper = mount(ModuleCatalogKanbanCard as any, {
    props: {
      card: makeCard({
        ModuleName: 'partner',
        Author: 'Acme Corp',
        License: 'MIT',
        Summary: 'Partner integrations pack',
        Application: 'partner',
        Installed: false,
        Available: true,
      }),
      statusLabel: 'Not Installed',
      statusVariant: 'secondary',
      primaryAction: 'install',
      primaryLabel: 'Install',
      showUninstall: false,
    },
    global,
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-card-author"]').text()).toBe('Acme Corp');
  expect(wrapper.find('[data-testid="module-catalog-card-license"]').text()).toBe('MIT');
  expect(wrapper.find('[data-testid="module-catalog-card-application"]').text()).toBe('partner');
  expect(wrapper.text()).toContain('Partner integrations pack');
  expect(wrapper.find('[data-testid="module-catalog-card-primary-install"]').exists()).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogKanbanCard.vue license-only uses ml-auto; progress and uninstall', async () => {
  const wrapper = mount(ModuleCatalogKanbanCard as any, {
    props: {
      card: makeCard({ ModuleName: 'core', License: 'Apache-2.0' }),
      statusLabel: 'Installed',
      statusVariant: 'outline',
      primaryAction: 'progress',
      primaryLabel: 'View Progress',
      showUninstall: true,
    },
    global,
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-card-author"]').exists()).toBe(false);
  const license = wrapper.find('[data-testid="module-catalog-card-license"]');
  expect(license.exists()).toBe(true);
  expect(String(license.element?.className || '')).toContain('ml-auto');
  expect(wrapper.find('[data-testid="module-catalog-card-primary-progress"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-catalog-card-uninstall"]').exists()).toBe(true);
  wrapper.unmount();
});

test('ModuleCatalogKanbanCard.vue upgrade primary when catalog row has update', async () => {
  const wrapper = mount(ModuleCatalogKanbanCard as any, {
    props: {
      card: makeCard({
        ModuleName: 'core',
        Installed: true,
        UpdateAvailable: true,
        InstalledVersion: '1.0',
        LocalVersion: '1.1',
      }),
      statusLabel: 'Update available',
      statusVariant: 'secondary',
      primaryAction: 'upgrade',
      primaryLabel: 'Upgrade',
      showUninstall: false,
    },
    global,
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-card-primary-upgrade"]').exists()).toBe(true);
  expect(wrapper.text()).toContain('Upgrade');
  wrapper.unmount();
});

test('ModuleCatalogKanbanCard.vue author-only attribution without license row', async () => {
  const wrapper = mount(ModuleCatalogKanbanCard as any, {
    props: {
      card: makeCard({ ModuleName: 'meta', Author: 'Choysum Team' }),
      statusLabel: 'Installed',
      statusVariant: 'outline',
      primaryAction: null,
      primaryLabel: '',
      showUninstall: false,
    },
    global,
  });
  await flushPromises();
  expect(wrapper.find('[data-testid="module-catalog-card-attribution"]').exists()).toBe(true);
  expect(wrapper.find('[data-testid="module-catalog-card-author"]').text()).toBe('Choysum Team');
  expect(wrapper.find('[data-testid="module-catalog-card-license"]').exists()).toBe(false);
  wrapper.unmount();
});

test('ModuleCatalogKanbanCard.vue primary and uninstall clicks emit', async () => {
  const events: string[] = [];
  const wrapper = mount(ModuleCatalogKanbanCard as any, {
    props: {
      card: makeCard({ ModuleName: 'auth', License: 'Apache-2.0' }),
      statusLabel: 'Installed',
      statusVariant: 'outline',
      primaryAction: 'install',
      primaryLabel: 'Install',
      showUninstall: true,
      onPrimary: () => events.push('primary'),
      onUninstall: () => events.push('uninstall'),
    },
    global,
  });
  await flushPromises();
  await wrapper.find('[data-testid="module-catalog-card-primary-install"]').trigger('click');
  await wrapper.find('[data-testid="module-catalog-card-uninstall"]').trigger('click');
  expect(events).toEqual(['primary', 'uninstall']);
  wrapper.unmount();
});
