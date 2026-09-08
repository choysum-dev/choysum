// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { defineComponent, inject, h, resolveComponent } from 'vue';
import { mount, flushPromises } from '@choysum/test-utils';

const feHostSymbol = Symbol('feHostSymbol');

const Probe = defineComponent({
  name: 'GlobalPluginsProbe',
  setup() {
    const token = inject('feHostToken', 'missing');
    const sym = inject(feHostSymbol, 'missing-symbol');
    return () => {
      const Extra = resolveComponent('Extra');
      return h('div', { 'data-testid': 'global-plugins-probe' }, [
        String(token),
        '|',
        String(sym),
        h(Extra),
      ]);
    };
  },
});

const plugin = {
  install(app: any, a?: string, b?: string) {
    app.provide('feHostToken', 'from-plugin');
    app.provide('fePluginOpts', String(a || '') + ':' + String(b || ''));
  },
};

async function main() {
  const wrapper = mount(Probe as any, {
    global: {
      plugins: [[plugin, 'opt-a', 'opt-b']],
      provide: {
        feHostToken: 'from-provide-override',
        [feHostSymbol]: 'from-symbol-provide',
      },
      components: {
        Extra: defineComponent({
          name: 'Extra',
          setup: () => () => h('span', { 'data-testid': 'global-extra' }, 'extra'),
        }),
      },
    },
  });
  await flushPromises();
  // provide on options applied after plugins — override wins.
  const text = wrapper.find('[data-testid="global-plugins-probe"]').text();
  const extra = wrapper.find('[data-testid="global-extra"]').exists();
  globalThis.__hostResult = { ready: true, text, extra };
  wrapper.unmount();
}

main().catch((err) => {
  globalThis.__hostResult = { ready: true, error: String(err && err.message ? err.message : err) };
});
