// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useConnectivityStore } from '@/web/web/stores/connectivityStore';
import TerminologyExportMenu from './TerminologyExportMenu.vue';

describe('TerminologyExportMenu offline download gate', () => {
  afterEach(() => {
    try {
      useConnectivityStore().resetForTests();
    } catch {
      /* no pinia */
    }
  });

  test('canDownload is false and onDownloadPo no-ops while unreachable', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();

    const mounted = mountApp(TerminologyExportMenu as any, {
      props: { application: 'web' },
      plugins: [pinia],
    });
    await flushPromises();

    const ss = mounted.setupState() as {
      canDownload: boolean;
      selectedModule: string;
      onDownloadPo: () => Promise<void>;
      downloading: boolean;
    };
    ss.selectedModule = 'web';
    await flushPromises();
    expect(ss.canDownload).toBe(false);
    await ss.onDownloadPo();
    expect(ss.downloading).toBe(false);

    mounted.unmount();
  });
});
