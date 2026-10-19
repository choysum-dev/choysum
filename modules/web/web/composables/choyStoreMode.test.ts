// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import {
  hasChoyStoreEngine,
  isChoyStoreFieldBinding,
  splitChoyAttrsListeners,
  useChoyStoreFieldBinding,
} from './choyStoreMode';
import { providePageContext } from './usePageContext';

describe('choyStoreMode', () => {
  test('isChoyStoreFieldBinding requires store+prop or binding', () => {
    expect(isChoyStoreFieldBinding({})).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {} })).toBe(false);
    expect(isChoyStoreFieldBinding({ prop: 'Name' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {}, prop: '' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: {}, prop: 'Name' })).toBe(true);
    expect(isChoyStoreFieldBinding({ binding: {} })).toBe(true);
  });

  test('isChoyStoreFieldBinding accepts page store with prop', () => {
    expect(isChoyStoreFieldBinding({ prop: 'Name' }, { model: 'x' })).toBe(true);
    expect(isChoyStoreFieldBinding({ prop: '' }, { model: 'x' })).toBe(false);
    expect(isChoyStoreFieldBinding({ store: { a: 1 }, prop: 'Name' }, { model: 'x' })).toBe(
      true,
    );
  });

  test('hasChoyStoreEngine accepts prop or page store', () => {
    expect(hasChoyStoreEngine(undefined, undefined)).toBe(false);
    expect(hasChoyStoreEngine(null, null)).toBe(false);
    expect(hasChoyStoreEngine({ model: 'x' }, null)).toBe(true);
    expect(hasChoyStoreEngine(undefined, { model: 'x' })).toBe(true);
  });

  test('useChoyStoreFieldBinding resolves page store into storeBind', async () => {
    const pageStore = { modelName: 'auth.User' };
    let mode = false;
    let bindStore: unknown = null;
    const Probe = defineComponent({
      props: { prop: { type: String, required: true } },
      setup(props) {
        const { storeMode, storeBind } = useChoyStoreFieldBinding(props as any, {});
        mode = storeMode.value;
        bindStore = storeBind.value.store;
        return () => h('div', { 'data-test': 'probe' });
      },
    });
    const Host = defineComponent({
      setup() {
        providePageContext({ store: () => pageStore });
        return () => h(Probe, { prop: 'Name' });
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(mode).toBe(true);
    expect(bindStore).toBe(pageStore);
    w.unmount();
  });

  test('storeBind omits chrome defineModel keys', async () => {
    const pageStore = { modelName: 'auth.User' };
    let bind: Record<string, unknown> = {};
    const Probe = defineComponent({
      props: {
        prop: { type: String, required: true },
        modelValue: { type: Array, default: () => [] },
        modelModifiers: { type: Object, default: () => ({}) },
        readonly: { type: Boolean, default: false },
      },
      setup(props) {
        const { storeBind } = useChoyStoreFieldBinding(props as any, { class: 'x' });
        bind = storeBind.value;
        return () => h('div', { 'data-test': 'probe' });
      },
    });
    const Host = defineComponent({
      setup() {
        providePageContext({ store: () => pageStore });
        return () =>
          h(Probe, {
            prop: 'Lines',
            modelValue: [{ Id: '1' }],
            modelModifiers: { trim: true },
            readonly: true,
          });
      },
    });
    const w = mountApp(Host);
    await flushPromises();
    expect(bind.store).toBe(pageStore);
    expect(bind.prop).toBe('Lines');
    expect(bind.readonly).toBe(true);
    expect(bind.class).toBe('x');
    expect('modelValue' in bind).toBe(false);
    expect('modelModifiers' in bind).toBe(false);
    w.unmount();
  });

  test('splitChoyAttrsListeners strips on prefix for v-on object keys', () => {
    const onFoo = () => undefined;
    const onSelectionChange = () => undefined;
    const onUpdateModelValue = () => undefined;
    const onBare = () => undefined;
    const onchangeSessionId = () => undefined;
    const { bind, listeners } = splitChoyAttrsListeners({
      class: 'x',
      'data-test': 't',
      onFoo,
      onSelectionChange,
      'onUpdate:modelValue': onUpdateModelValue,
      onBar: 'not-a-function',
      on: onBare,
      // CamelCase prop mistaken for a listener if we only use startsWith('on').
      onchangeSessionId,
    });
    expect(bind).toEqual({
      class: 'x',
      'data-test': 't',
      onBar: 'not-a-function',
      on: onBare,
      onchangeSessionId,
    });
    expect(listeners).toEqual({
      foo: onFoo,
      selectionChange: onSelectionChange,
      'update:modelValue': onUpdateModelValue,
    });
  });
});
