// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { defineComponent, inject, h } from 'vue';
import { mount, flushPromises } from '@choysum/test-utils';

const Probe = defineComponent({
  name: 'GlobalPluginsProbe',
  setup() {
    const token = inject('feHostToken', 'missing');
    return () => h('div', { 'data-testid': 'global-plugins-probe' }, String(token));
  },
});

const plugin = {
  install(app: any) {
    app.provide('feHostToken', 'from-plugin');
  },
};

async function main() {
  const wrapper = mount(Probe as any, {
    global: {
      plugins: [plugin],
      provide: { feHostToken: 'from-provide-override' },
      components: {
        Extra: defineComponent({
          name: 'Extra',
          setup: () => () => h('span', 'extra'),
        }),
      },
    },
  });
  await flushPromises();
  // provide on options applied after plugins — override wins.
  const text = wrapper.find('[data-testid="global-plugins-probe"]').text();
  globalThis.__hostResult = { ready: true, text };
  wrapper.unmount();
}

main().catch((err) => {
  globalThis.__hostResult = { ready: true, error: String(err && err.message ? err.message : err) };
});
