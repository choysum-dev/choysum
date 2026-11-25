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

  test('ActionMenuItem download is disabled while unreachable; onDownloadPo still guards busy', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    useConnectivityStore().markUnreachable();

    const mounted = mountApp(TerminologyExportMenu as any, {
      props: { application: 'web' },
      plugins: [pinia],
    });
    await flushPromises();

    const ss = mounted.setupState() as {
      canDownloadBase: boolean;
      selectedModule: string;
      onDownloadPo: () => Promise<void>;
      downloading: boolean;
    };
    ss.selectedModule = 'web';
    await flushPromises();
    expect(ss.canDownloadBase).toBe(true);

    const download = mounted.q('[data-testid="terminology-export-download"]') as HTMLElement | null;
    expect(download).not.toBeNull();
    expect(
      download!.getAttribute('data-disabled') != null ||
        download!.hasAttribute('disabled') ||
        (download as HTMLButtonElement).disabled === true,
    ).toBe(true);

    await ss.onDownloadPo();
    expect(ss.downloading).toBe(false);

    mounted.unmount();
  });
});
