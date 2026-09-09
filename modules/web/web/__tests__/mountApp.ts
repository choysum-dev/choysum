// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Vitest + happy-dom mount helper for FE unit tests that cannot use VTU.
 * Registers global/unresolved tag stubs via app.component; for script-setup
 * local imports, use stubSfc() to mutate the shared component export in place.
 */

import {
  createApp,
  defineComponent,
  h,
  provide,
  reactive,
  type App,
  type Component,
  type Plugin,
} from 'vue';

export type CallRecorder = { calls: unknown[][] };

export type FnRecorder<T = undefined, A extends unknown[] = unknown[]> = CallRecorder &
  ((...args: A) => T | Promise<T>) & {
    mockReset: () => void;
    mockClear: () => void;
    mockImplementation: (fn: (...args: A) => T | Promise<T>) => void;
    mockReturnValue: (value: T) => void;
  };

export function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): FnRecorder<T, A> {
  let current = impl;
  const rec = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return current ? current(...args) : (undefined as T);
    },
    {
      calls: [] as unknown[][],
      mockReset() {
        rec.calls = [];
        current = impl;
      },
      mockClear() {
        rec.calls = [];
      },
      mockImplementation(fn: (...args: A) => T | Promise<T>) {
        current = fn;
      },
      mockReturnValue(value: T) {
        current = (() => value) as (...args: A) => T;
      },
    }
  ) as FnRecorder<T, A>;
  return rec;
}

export async function flushPromises(): Promise<void> {
  await Promise.resolve();
  await Promise.resolve();
  await new Promise<void>(resolve => setTimeout(resolve, 0));
}

export function stub(name: string, extra?: Component): Component {
  if (extra) {
    return { name, ...(extra as object) } as Component;
  }
  return {
    name,
    setup(_, { slots }) {
      return () => h('div', { 'data-stub': name }, slots.default?.());
    },
  };
}

type SfcSnapshot = {
  setup?: unknown;
  render?: unknown;
  ssrRender?: unknown;
  props?: unknown;
  emits?: unknown;
};

const sfcSnapshots = new WeakMap<object, SfcSnapshot>();

/**
 * Mutates a compiled SFC export so parents that closed over the same object
 * render the stub instead. Call restoreSfc() in afterEach when needed.
 */
export function stubSfc(Comp: any, replacement: Component): void {
  if (!sfcSnapshots.has(Comp)) {
    sfcSnapshots.set(Comp, {
      setup: Comp.setup,
      render: Comp.render,
      ssrRender: Comp.ssrRender,
      props: Comp.props,
      emits: Comp.emits,
    });
  }
  const r = replacement as any;
  Comp.setup = r.setup;
  // Clear compiled render so setup-returned render wins (Element Plus withInstall SFCs).
  Comp.render = r.render ?? undefined;
  Comp.ssrRender = r.ssrRender ?? undefined;
  if (r.props !== undefined) Comp.props = r.props;
  if (r.emits !== undefined) Comp.emits = r.emits;
  if (!Comp.setup && !Comp.render) {
    Comp.setup = () => () => h('div', { 'data-stub': Comp.__name || Comp.name || 'Stub' });
  }
}

export function restoreSfc(Comp: any): void {
  const snap = sfcSnapshots.get(Comp);
  if (!snap) return;
  Comp.setup = snap.setup;
  Comp.render = snap.render;
  Comp.ssrRender = snap.ssrRender;
  Comp.props = snap.props;
  Comp.emits = snap.emits;
  sfcSnapshots.delete(Comp);
}

export type MountAppOptions = {
  props?: Record<string, unknown>;
  /** Vue listeners as onXxx (e.g. onQueryUpdate). */
  on?: Record<string, (...args: any[]) => void>;
  stubs?: Record<string, Component | true>;
  plugins?: Plugin[];
  provide?: Record<string | symbol, unknown>;
  slots?: Record<string, (...args: any[]) => any>;
  /** When true, props are reactive and returned for setProps-style updates. */
  reactiveProps?: boolean;
};

export type MountAppResult = {
  el: HTMLElement;
  app: App;
  /** Public instance of the mounted Comp (via ref). */
  root: any;
  props: Record<string, unknown>;
  unmount: () => void;
  text: () => string;
  q: (sel: string) => Element | null;
  qa: (sel: string) => Element[];
  setupState: () => any;
  click: (sel: string) => void;
};

export function mountApp(Comp: Component, opts: MountAppOptions = {}): MountAppResult {
  const props = opts.reactiveProps
    ? reactive({ ...(opts.props || {}) })
    : { ...(opts.props || {}) };
  let root: any = null;

  const Host = defineComponent({
    name: 'MountAppHost',
    setup() {
      if (opts.provide) {
        for (const key of Reflect.ownKeys(opts.provide)) {
          provide(key as any, (opts.provide as any)[key]);
        }
      }
      return () =>
        h(
          Comp as any,
          {
            ...props,
            ...(opts.on || {}),
            ref: (r: any) => {
              root = r;
            },
          },
          opts.slots
        );
    },
  });

  const app = createApp(Host);
  if (opts.stubs) {
    for (const [name, def] of Object.entries(opts.stubs)) {
      app.component(name, def === true ? stub(name) : def);
    }
  }
  for (const plugin of opts.plugins || []) {
    app.use(plugin);
  }

  const el = document.createElement('div');
  document.body.appendChild(el);
  app.mount(el);

  return {
    el,
    app,
    get root() {
      return root;
    },
    props: props as Record<string, unknown>,
    unmount: () => {
      app.unmount();
      el.remove();
    },
    text: () => el.textContent ?? '',
    q: (sel: string) => el.querySelector(sel),
    qa: (sel: string) => Array.from(el.querySelectorAll(sel)),
    setupState: () => root?.$?.setupState,
    click: (sel: string) => {
      const node = el.querySelector(sel) as HTMLElement | null;
      if (!node) throw new Error(`click: missing ${sel}`);
      node.click();
    },
  };
}
