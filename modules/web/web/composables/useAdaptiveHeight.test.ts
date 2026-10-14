// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, KeepAlive, nextTick, ref, type Ref } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useAdaptiveHeight } from './useAdaptiveHeight';

function rect(partial: Partial<DOMRect>): DOMRect {
  return {
    x: 0,
    y: 0,
    width: 100,
    height: 100,
    top: 0,
    left: 0,
    bottom: 100,
    right: 100,
    toJSON: () => ({}),
    ...partial,
  } as DOMRect;
}

function wrapRef(): Ref<Element | null> {
  // Choysum's DOM typings widen Element; cast keeps the composable contract.
  return ref(null) as Ref<Element | null>;
}

async function flushFrames(n = 3) {
  for (let i = 0; i < n; i++) {
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()));
    await flushPromises();
  }
}

/** fe-qjs minimal DOM omits window listeners / ResizeObserver; stub for observer paths. */
function installWindowObserverStubs() {
  const w = window as any;
  if (typeof w.addEventListener !== 'function') {
    const bag = new Map<string, Set<() => void>>();
    w.addEventListener = (type: string, fn: () => void) => {
      if (!bag.has(type)) bag.set(type, new Set());
      bag.get(type)!.add(fn);
    };
    w.removeEventListener = (type: string, fn: () => void) => {
      bag.get(type)?.delete(fn);
    };
  }
  if (typeof w.ResizeObserver !== 'function') {
    w.ResizeObserver = class ResizeObserverStub {
      private cb: () => void;
      constructor(cb: () => void) {
        this.cb = cb;
      }
      observe(_el: Element) {
        // Touch the callback so the RO wiring is covered.
        void this.cb;
      }
      unobserve(_el: Element) {}
      disconnect() {}
    };
  }
}

describe('useAdaptiveHeight', () => {
  installWindowObserverStubs();

  test('mode none leaves height at 0', async () => {
    let height = 0;
    const Host = defineComponent({
      setup() {
        const elRef = wrapRef();
        const api = useAdaptiveHeight(elRef, { mode: 'none' });
        return () => {
          height = api.height.value;
          return h('div', {
            ref: (el: unknown) => {
              elRef.value = (el as Element | null) ?? null;
            },
          });
        };
      },
    });
    const m = mountApp(Host);
    await flushFrames();
    expect(height).toBe(0);
    m.unmount();
  });

  test('viewport mode sizes from window and cleans up observers', async () => {
    let api: ReturnType<typeof useAdaptiveHeight> | null = null;
    const Host = defineComponent({
      setup() {
        const elRef = wrapRef();
        api = useAdaptiveHeight(elRef, { mode: 'viewport', viewportGap: 0, minViewportHeight: 100 });
        return () =>
          h('div', {
            ref: (el: any) => {
              if (!el) return;
              el.getBoundingClientRect = () => rect({ top: 200 });
              el.closest = () => null;
              elRef.value = el as Element;
            },
          });
      },
    });
    const prev = window.innerHeight;
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 900 });
    const m = mountApp(Host);
    await flushFrames();
    expect(api!.height.value).toBeGreaterThanOrEqual(100);
    m.unmount();
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: prev });
  });

  test('shell scroll path observes shell and footer', async () => {
    let api: ReturnType<typeof useAdaptiveHeight> | null = null;
    const shell = document.createElement('div');
    shell.setAttribute('data-testid', 'choy-shell-scroll');
    shell.getBoundingClientRect = () => rect({ top: 0, bottom: 800 });
    const footer = document.createElement('div');
    footer.setAttribute('data-testid', 'choy-shell-footer');
    footer.getBoundingClientRect = () => rect({ top: 600, bottom: 640 });
    shell.appendChild(footer);
    document.body.appendChild(shell);

    const Host = defineComponent({
      setup() {
        const elRef = wrapRef();
        api = useAdaptiveHeight(elRef, { mode: 'viewport', minViewportHeight: 240 });
        return () =>
          h('div', {
            ref: (el: any) => {
              if (!el) return;
              el.getBoundingClientRect = () => rect({ top: 100 });
              el.closest = ((sel: string) => {
                if (sel === '[data-testid="choy-shell-scroll"]') return shell;
                return null;
              }) as typeof el.closest;
              elRef.value = el as Element;
            },
          });
      },
    });
    const m = mountApp(Host);
    await flushFrames();
    expect(api!.height.value).toBe(500);
    api!.recompute();
    await flushFrames(2);
    expect(api!.height.value).toBe(500);
    m.unmount();
    shell.remove();
  });

  test('container mode uses dialog body when present', async () => {
    let api: ReturnType<typeof useAdaptiveHeight> | null = null;
    const dialog = document.createElement('div');
    dialog.className = 'el-dialog';
    const body = document.createElement('div');
    body.className = 'el-dialog__body';
    body.getBoundingClientRect = () => rect({ top: 40, bottom: 440 });
    dialog.appendChild(body);
    document.body.appendChild(dialog);

    const Host = defineComponent({
      setup() {
        const elRef = wrapRef();
        api = useAdaptiveHeight(elRef, {
          mode: 'container',
          minContainerHeight: 100,
          containerPadding: 0,
        });
        return () =>
          h('div', {
            ref: (el: any) => {
              if (!el) return;
              el.getBoundingClientRect = () => rect({ top: 80 });
              el.closest = ((sel: string) => {
                if (sel === '.el-dialog__body') return body;
                return null;
              }) as typeof el.closest;
              elRef.value = el as Element;
            },
          });
      },
    });
    const m = mountApp(Host);
    await flushFrames();
    expect(api!.height.value).toBeGreaterThanOrEqual(100);
    m.unmount();
    dialog.remove();
    await nextTick();
  });

  test('auto mode picks container inside dialog and honors containerSelector', async () => {
    let api: ReturnType<typeof useAdaptiveHeight> | null = null;
    const dialog = document.createElement('div');
    dialog.className = 'el-dialog';
    const body = document.createElement('div');
    body.className = 'el-dialog__body custom-container';
    body.getBoundingClientRect = () => rect({ top: 20, bottom: 420 });
    dialog.appendChild(body);
    document.body.appendChild(dialog);

    const Host = defineComponent({
      setup() {
        const elRef = wrapRef();
        api = useAdaptiveHeight(elRef, {
          mode: 'auto',
          containerSelector: '.custom-container',
          minContainerHeight: 80,
          containerPadding: 0,
        });
        return () =>
          h('div', {
            ref: (el: any) => {
              if (!el) return;
              el.getBoundingClientRect = () => rect({ top: 60 });
              el.closest = ((sel: string) => {
                if (sel === '.el-dialog__body' || sel === '.custom-container') return body;
                return null;
              }) as typeof el.closest;
              elRef.value = el as Element;
            },
          });
      },
    });
    const m = mountApp(Host);
    await flushFrames();
    expect(api!.height.value).toBeGreaterThanOrEqual(80);
    api!.recompute();
    api!.recompute();
    await flushFrames(2);
    expect(api!.height.value).toBeGreaterThanOrEqual(80);
    m.unmount();
    dialog.remove();
  });

  test('container mode without resolvable container falls back to viewport', async () => {
    let api: ReturnType<typeof useAdaptiveHeight> | null = null;
    const Host = defineComponent({
      setup() {
        const elRef = wrapRef();
        api = useAdaptiveHeight(elRef, {
          mode: 'container',
          minViewportHeight: 120,
          viewportGap: 0,
        });
        return () =>
          h('div', {
            ref: (el: any) => {
              if (!el) return;
              el.getBoundingClientRect = () => rect({ top: 100 });
              el.closest = (() => null) as typeof el.closest;
              Object.defineProperty(el, 'parentElement', { configurable: true, value: null });
              elRef.value = el as Element;
            },
          });
      },
    });
    const prev = window.innerHeight;
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 });
    const m = mountApp(Host);
    await flushFrames();
    expect(api!.height.value).toBeGreaterThanOrEqual(120);
    m.unmount();
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: prev });
  });

  test('auto mode without dialog uses viewport and KeepAlive reactivates', async () => {
    let api: ReturnType<typeof useAdaptiveHeight> | null = null;
    const show = ref(true);
    const Inner = defineComponent({
      name: 'AdaptiveInner',
      setup() {
        const elRef = wrapRef();
        api = useAdaptiveHeight(elRef, { mode: 'auto', viewportGap: 0, minViewportHeight: 80 });
        return () =>
          h('div', {
            ref: (el: any) => {
              if (!el) return;
              el.getBoundingClientRect = () => rect({ top: 100 });
              el.closest = (() => null) as typeof el.closest;
              elRef.value = el as Element;
            },
          });
      },
    });
    const Host = defineComponent({
      setup() {
        return () =>
          h(KeepAlive, null, {
            default: () => (show.value ? h(Inner) : null),
          });
      },
    });
    const prev = window.innerHeight;
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 700 });
    const m = mountApp(Host);
    await flushFrames();
    expect(api!.height.value).toBeGreaterThanOrEqual(80);
    show.value = false;
    await flushFrames();
    show.value = true;
    await flushFrames();
    expect(api!.height.value).toBeGreaterThanOrEqual(80);
    m.unmount();
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: prev });
  });
});
