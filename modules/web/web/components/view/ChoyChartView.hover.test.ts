// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { nextTick } from 'vue';
import ChoyChartView from './ChoyChartView.vue';

function makeStore() {
  return {
    storeId: 'web.chart.hover.test',
    state: { queryState: {} },
  } as any;
}

function stubRect(el: Element, rect: Record<string, number>): void {
  const value = () => ({ ...rect, x: rect.left, y: rect.top, toJSON: () => ({}) });
  try {
    Object.defineProperty(el, 'getBoundingClientRect', { value, configurable: true });
  } catch {
    (el as any).getBoundingClientRect = value;
  }
}

function injectPlotSvg(host: Element): void {
  if (host.querySelector('svg')) {
    stubRect(host, { left: 100, top: 50, width: 400, height: 200, right: 500, bottom: 250 });
    return;
  }
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  stubRect(svg, { left: 100, top: 50, width: 400, height: 200, right: 500, bottom: 250 });
  const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  const line = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  line.setAttribute('class', 'line');
  stubRect(g, { left: 140, top: 60, width: 320, height: 160, right: 460, bottom: 220 });
  g.appendChild(line);
  svg.appendChild(g);
  host.appendChild(svg);
  stubRect(host, { left: 100, top: 50, width: 400, height: 200, right: 500, bottom: 250 });
}

const nestedGroupResult = {
  kind: 'group',
  rows: [
    {
      kind: 'group',
      depth: 0,
      key: 'refresh',
      label: 'refresh',
      children: [
        { kind: 'group', depth: 1, key: 'open', label: 'Open', count: 10 },
        { kind: 'group', depth: 1, key: 'done', label: 'Done', count: 20 },
      ],
    },
    {
      kind: 'group',
      depth: 0,
      key: 'access',
      label: 'access',
      children: [
        { kind: 'group', depth: 1, key: 'open', label: 'Open', count: 5 },
        { kind: 'group', depth: 1, key: 'done', label: 'Done', count: 25 },
      ],
    },
  ],
};

type ChartHoverExpose = {
  controller: { vm: { result: unknown } };
  rebuildFromSnapshot: () => void;
  onXyPointerMove: (event: { clientX: number; clientY: number }) => void;
  clearLinePlotHover: () => void;
  updateLinePlotHover: (event: { clientX: number; clientY: number }) => void;
  placeLineHoverTip: (html: string, x: number, y: number) => void;
  onXyPointerLeave: () => void;
  onViewportDismissLineHover: () => void;
  findScatterPointEl: (root: Element | null, c: number, s: number) => Element | null;
  highlightScatterPointForTooltip: (
    c: number | null,
    s: number | undefined,
    root: Element | null,
  ) => void;
  onLineClick: (d: unknown, e: unknown, si?: number) => void;
  selectChartType: (t: string) => void;
  linePlotYDomain: () => [number, number];
};

function tipVisible(): boolean {
  return !!document.querySelector('[data-testid=choy-chart-line-tooltip]');
}

function bandVisible(): boolean {
  return !!document.querySelector('[data-testid=choy-chart-line-band]');
}

async function mountLineChartWithData() {
  const store = makeStore();
  const mounted = mountApp(ChoyChartView as any, {
    props: {
      store,
      autoBootstrap: false,
      chartType: 'line',
      stacked: false,
      metrics: [{ alias: 'count', label: 'Count' }],
    },
  });
  await flushPromises();
  const exposed = mounted.root as ChartHoverExpose;
  exposed.controller.vm.result = nestedGroupResult;
  exposed.rebuildFromSnapshot();
  await nextTick();
  await flushPromises();
  const host = mounted.q('[data-region=chart-xy-plot]') as HTMLElement | null;
  expect(host).not.toBeNull();
  injectPlotSvg(host!);
  return { mounted, exposed };
}

describe('ChoyChartView line hover tip', () => {
  test('shows tip chrome over plot and clears on dismiss', async () => {
    const { mounted, exposed } = await mountLineChartWithData();
    expect(mounted.text()).toContain('Open');

    exposed.onXyPointerMove({ clientX: 430, clientY: 80 });
    await nextTick();
    await flushPromises();

    expect(bandVisible()).toBe(true);
    expect(tipVisible()).toBe(true);

    exposed.clearLinePlotHover();
    await nextTick();
    expect(tipVisible()).toBe(false);
    expect(bandVisible()).toBe(false);

    mounted.unmount();
  });

  test('rebuildFromSnapshot clears a stale tip', async () => {
    const { mounted, exposed } = await mountLineChartWithData();

    exposed.onXyPointerMove({ clientX: 430, clientY: 80 });
    await nextTick();
    expect(tipVisible()).toBe(true);

    exposed.rebuildFromSnapshot();
    // Re-inject after rebuild in case the plot host re-rendered.
    const host = mounted.q('[data-region=chart-xy-plot]') as HTMLElement | null;
    if (host && !host.querySelector('svg')) injectPlotSvg(host);
    await nextTick();
    expect(tipVisible()).toBe(false);
    expect(bandVisible()).toBe(false);

    mounted.unmount();
  });

  test('viewport scroll listener dismisses the floating tip', async () => {
    const { mounted, exposed } = await mountLineChartWithData();
    exposed.onXyPointerMove({ clientX: 430, clientY: 80 });
    await nextTick();
    expect(tipVisible()).toBe(true);

    if (typeof window !== 'undefined' && typeof window.dispatchEvent === 'function') {
      window.dispatchEvent(new Event('scroll'));
    } else {
      exposed.clearLinePlotHover();
    }
    await nextTick();
    expect(tipVisible()).toBe(false);

    mounted.unmount();
  });

  test('hover helpers cover outside-plot, leave, and scatter lookup', async () => {
    const { mounted, exposed } = await mountLineChartWithData();
    const host = mounted.q('[data-region=chart-xy-plot]') as HTMLElement;
    const setup = mounted.setupState() as { xyPlotHostRef?: { value: HTMLElement | null } };
    if (setup.xyPlotHostRef) setup.xyPlotHostRef.value = host;

    expect(exposed.linePlotYDomain()[1]).toBeGreaterThan(0);
    exposed.updateLinePlotHover({ clientX: 430, clientY: 80 });
    await nextTick();
    expect(tipVisible()).toBe(true);
    exposed.placeLineHoverTip('<div>x</div>', 430, 80);
    await nextTick();
    exposed.updateLinePlotHover({ clientX: 10, clientY: 10 });
    await nextTick();
    expect(tipVisible()).toBe(false);

    const svg = host.querySelector('svg')!;
    for (let i = 0; i < 8; i++) {
      const n = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      n.setAttribute('class', 'point');
      svg.appendChild(n);
    }
    // Enough bare points (no __data__) for series-major layout fallback.
    while (svg.querySelectorAll('circle').length < 8) {
      const n = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      n.setAttribute('class', 'point');
      svg.appendChild(n);
    }
    const prevEscape = (globalThis as any).CSS;
    (globalThis as any).CSS = {
      escape() {
        throw new Error('escape fail');
      },
    };
    try {
      exposed.findScatterPointEl(svg, 0, 1);
      exposed.findScatterPointEl(svg, 1, 0);
    } finally {
      (globalThis as any).CSS = prevEscape;
    }
    const pts = svg.querySelectorAll('circle');
    if (pts[0]) {
      pts[0].classList.add('choy-chart-mark--hover');
      (pts[0] as Element & { __data__?: unknown }).__data__ = {
        index: 0,
        _point: { pointIndex: 0, groupIndex: 0 },
      };
    }
    if (pts[1]) {
      (pts[1] as Element & { __data__?: unknown }).__data__ = { index: '1' };
    }
    exposed.findScatterPointEl(svg, 0, 0);
    exposed.findScatterPointEl(svg, 1, 0);
    exposed.highlightScatterPointForTooltip(null, undefined, svg);
    exposed.highlightScatterPointForTooltip(0, 0, svg);
    exposed.onLineClick(null, { clientX: 430, currentTarget: svg, target: svg }, 0);
    const setupClear = mounted.setupState() as {
      clearChartHover?: () => void;
      applyHoverMarkEl?: (el: Element, kind?: string) => void;
    };
    try {
      if (pts[0]) setupClear.applyHoverMarkEl?.(pts[0], 'fill');
      setupClear.clearChartHover?.();
    } catch {
      /* ignore */
    }
    exposed.onXyPointerLeave();
    exposed.onViewportDismissLineHover();

    // Early-exit paths: zero-size plot, missing svg, empty data tooltip.
    const zero = { left: 0, top: 0, width: 0, height: 0, right: 0, bottom: 0 };
    const liveSvg = host.querySelector('svg');
    const liveG = host.querySelector('g');
    if (liveSvg) stubRect(liveSvg, zero);
    if (liveG) stubRect(liveG, zero);
    exposed.updateLinePlotHover({ clientX: 430, clientY: 80 });
    if (liveSvg && liveSvg.parentNode) liveSvg.parentNode.removeChild(liveSvg);
    exposed.updateLinePlotHover({ clientX: 430, clientY: 80 });
    injectPlotSvg(host);
    if (setup.xyPlotHostRef) setup.xyPlotHostRef.value = host;
    exposed.controller.vm.result = { kind: 'group', rows: [] };
    exposed.rebuildFromSnapshot();
    await nextTick();
    exposed.updateLinePlotHover({ clientX: 430, clientY: 80 });

    mounted.unmount();
  });
});
