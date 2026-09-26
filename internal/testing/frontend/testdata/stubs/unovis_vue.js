// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@unovis/vue` chart components used by ChoyChartView. */
import { defineComponent, h } from 'vue';

function stub(name) {
  return defineComponent({
    name,
    setup(_, { slots, attrs }) {
      return () => h('div', { 'data-stub': name, ...attrs }, slots.default?.());
    },
  });
}

export const VisXYContainer = stub('VisXYContainer');
export const VisSingleContainer = stub('VisSingleContainer');
export const VisGroupedBar = stub('VisGroupedBar');
export const VisStackedBar = stub('VisStackedBar');
export const VisLine = stub('VisLine');
export const VisArea = stub('VisArea');
export const VisAxis = stub('VisAxis');
export const VisDonut = stub('VisDonut');
export const VisTooltip = stub('VisTooltip');
export default {
  VisXYContainer,
  VisSingleContainer,
  VisGroupedBar,
  VisStackedBar,
  VisLine,
  VisArea,
  VisAxis,
  VisDonut,
  VisTooltip,
};
