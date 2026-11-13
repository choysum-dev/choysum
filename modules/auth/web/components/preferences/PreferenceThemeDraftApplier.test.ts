// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, h, nextTick, ref } from 'vue';
import { PreferenceThemeDraftApplier } from './PreferenceThemeDraftApplier';

async function flush(): Promise<void> {
  await nextTick();
  await nextTick();
}

test('PreferenceThemeDraftApplier emits apply when theme draft changes', async () => {
  const theme = ref<unknown>('light');
  const modes: string[] = [];
  const host = document.createElement('div');
  const app = createApp({
    setup() {
      return () =>
        h(PreferenceThemeDraftApplier, {
          theme: theme.value,
          active: true,
          onApply: (mode: string) => {
            modes.push(mode);
          },
        });
    },
  });
  app.mount(host);
  await flush();
  expect(modes[0]).toBe('light');

  theme.value = 'dark';
  await flush();
  expect(modes[modes.length - 1]).toBe('dark');

  app.unmount();
});

test('PreferenceThemeDraftApplier stays quiet when inactive', async () => {
  const theme = ref<unknown>('light');
  const modes: string[] = [];
  const host = document.createElement('div');
  const app = createApp({
    setup() {
      return () =>
        h(PreferenceThemeDraftApplier, {
          theme: theme.value,
          active: false,
          onApply: (mode: string) => {
            modes.push(mode);
          },
        });
    },
  });
  app.mount(host);
  await flush();
  theme.value = 'dark';
  await flush();
  expect(modes).toEqual([]);
  app.unmount();
});
