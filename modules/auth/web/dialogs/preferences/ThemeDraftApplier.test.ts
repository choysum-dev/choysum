// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, h, nextTick, ref } from 'vue';
import { ThemeDraftApplier } from './ThemeDraftApplier';

async function flush(): Promise<void> {
  await nextTick();
  await nextTick();
}

test('ThemeDraftApplier emits apply when theme draft changes', async () => {
  const theme = ref<unknown>('light');
  const density = ref<unknown>('comfortable');
  const modes: string[] = [];
  const densities: unknown[] = [];
  const host = document.createElement('div');
  const app = createApp({
    setup() {
      return () =>
        h(ThemeDraftApplier, {
          theme: theme.value,
          density: density.value,
          active: true,
          onApply: (mode: string, nextDensity?: unknown) => {
            modes.push(mode);
            densities.push(nextDensity);
          },
        });
    },
  });
  app.mount(host);
  await flush();
  expect(modes[0]).toBe('light');
  expect(densities[0]).toBe('comfortable');

  theme.value = 'dark';
  await flush();
  expect(modes[modes.length - 1]).toBe('dark');

  density.value = 'compact';
  await flush();
  expect(densities[densities.length - 1]).toBe('compact');

  app.unmount();
});

test('ThemeDraftApplier stays quiet when inactive', async () => {
  const theme = ref<unknown>('light');
  const modes: string[] = [];
  const host = document.createElement('div');
  const app = createApp({
    setup() {
      return () =>
        h(ThemeDraftApplier, {
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
