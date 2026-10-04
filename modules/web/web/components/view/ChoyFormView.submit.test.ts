// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { defineComponent, h, inject, onMounted } from 'vue';
import type { App } from 'vue';
import * as VueRouter from 'vue-router';

import { flushPromises, fnRecorder, mountApp } from '@/web/web/__tests__/mountApp';
import { FIELD_CLIENT_VALIDATORS_KEY } from '@/web/web/composables/fieldClientValidation';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { createLocalFormStore } from '@/web/web/stores/localFormStore';
import FormView from './ChoyFormView.vue';

const createFeStubRouter = (VueRouter as any).createFeStubRouter;

const NoopLoading = {
  install(app: App) {
    app.directive('loading', {
      mounted() {},
      updated() {},
      unmounted() {},
    });
  },
};

function loginStore() {
  return createLocalFormStore({
    fields: [{ name: 'Username', label: 'Username', type: 'varchar' }],
    initialValues: { Username: 'admin' },
  });
}

function SlotSubmit(opts?: { failValidate?: boolean }) {
  return defineComponent({
    setup() {
      const validators = inject(FIELD_CLIENT_VALIDATORS_KEY, null);
      onMounted(() => {
        if (opts?.failValidate) {
          validators?.set('Username', async () => 'required');
        }
      });
      return () =>
        h('button', { type: 'submit', 'data-test': 'slot-submit' }, 'Go');
    },
  });
}

describe('FormView native form submit', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  function mountForm(opts?: {
    showMessages?: boolean;
    showHeader?: boolean;
    showActions?: boolean;
    failValidate?: boolean;
    submitHandler?: ReturnType<typeof fnRecorder>;
  }) {
    const submitHandler = opts?.submitHandler ?? fnRecorder(async () => ({ handled: true, skipSuccessMessage: true }));
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const Slot = SlotSubmit({ failValidate: opts?.failValidate });
    const wrapper = mountApp(FormView as any, {
      props: {
        store: loginStore(),
        viewMode: 'create',
        embedded: true,
        showHeader: opts?.showHeader ?? opts?.showActions ?? false,
        showActions: opts?.showActions ?? false,
        showMessages: opts?.showMessages ?? false,
        resolveRecordIdFromRoute: false,
        initialValues: { Username: 'admin' },
        submitHandler,
      },
      plugins: [createPinia(), router, NoopLoading],
      slots: {
        default: () => h(Slot),
      },
    });
    return { wrapper, submitHandler };
  }

  test('showHeader=false omits view-chrome action bar', async () => {
    const { wrapper: hidden } = mountForm({ showHeader: false, showActions: false });
    await flushPromises();
    expect(hidden.q('[data-anchor="choy.form.view-chrome"]')).toBeNull();
    hidden.unmount();

    const { wrapper: shown } = mountForm({ showHeader: true, showActions: true });
    await flushPromises();
    expect(shown.q('[data-anchor="choy.form.view-chrome"]')).toBeTruthy();
    shown.unmount();
  });

  test('slot type=submit button lives in the form and native submit runs submitHandler once', async () => {
    const { wrapper, submitHandler } = mountForm();
    await flushPromises();
    const btn = wrapper.q('[data-test="slot-submit"]') as HTMLButtonElement | null;
    expect(btn).toBeTruthy();
    expect(btn!.getAttribute('type')).toBe('submit');
    expect(btn!.closest('form')).toBeTruthy();
    btn!.closest('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(submitHandler.calls.length).toBe(1);
    wrapper.unmount();
  });

  test('form submit event (Enter) runs submitHandler once', async () => {
    const { wrapper, submitHandler } = mountForm();
    await flushPromises();
    const form = wrapper.q('form') as HTMLFormElement | null;
    expect(form).toBeTruthy();
    form!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(submitHandler.calls.length).toBe(1);
    wrapper.unmount();
  });

  test('a second native submit while the first is pending is ignored', async () => {
    let resolveHandler: ((value: { handled: boolean; skipSuccessMessage: boolean }) => void) | undefined;
    const submitHandler = fnRecorder(
      () =>
        new Promise<{ handled: boolean; skipSuccessMessage: boolean }>(resolve => {
          resolveHandler = resolve;
        })
    );
    const { wrapper } = mountForm({ submitHandler });
    await flushPromises();
    const form = wrapper.q('form') as HTMLFormElement | null;
    expect(form).toBeTruthy();
    form!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    form!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(submitHandler.calls.length).toBe(1);
    resolveHandler?.({ handled: true, skipSuccessMessage: true });
    await flushPromises();
    expect(submitHandler.calls.length).toBe(1);
    wrapper.unmount();
  });

  test('native submit in display mode does not call submitHandler', async () => {
    const { wrapper, submitHandler } = mountForm();
    await flushPromises();
    wrapper.setupState().controller.vm.mode = 'display';
    await flushPromises();
    const form = wrapper.q('form') as HTMLFormElement | null;
    form!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(submitHandler.calls.length).toBe(0);
    const outcome = await wrapper.root.submit();
    expect(outcome.reason).toBe('not-editable');
    expect(submitHandler.calls.length).toBe(0);
    wrapper.unmount();
  });

  test('nested form submit does not save the parent FormView', async () => {
    const submitHandler = fnRecorder(async () => ({ handled: true, skipSuccessMessage: true }));
    const Nested = defineComponent({
      setup() {
        return () =>
          h('form', { 'data-test': 'nested-form' }, [
            h('button', { type: 'submit', 'data-test': 'nested-submit' }, 'Inner'),
          ]);
      },
    });
    const { router } = createFeStubRouter({
      route: { name: 'login', path: '/web/login', fullPath: '/web/login', params: {}, query: {}, meta: {} },
    });
    const wrapper = mountApp(FormView as any, {
      props: {
        store: loginStore(),
        viewMode: 'create',
        embedded: true,
        showHeader: false,
        showActions: false,
        showMessages: false,
        resolveRecordIdFromRoute: false,
        initialValues: { Username: 'admin' },
        submitHandler,
      },
      plugins: [createPinia(), router, NoopLoading],
      slots: {
        default: () => h(Nested),
      },
    });
    await flushPromises();
    const nested = wrapper.q('[data-test="nested-form"]') as HTMLFormElement | null;
    expect(nested).toBeTruthy();
    nested!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();
    expect(submitHandler.calls.length).toBe(0);
    wrapper.unmount();
  });

  test('toolbar Save is type=button and a single submit does not double-call the handler', async () => {
    const { wrapper, submitHandler } = mountForm({ showActions: true });
    await flushPromises();
    const save = Array.from(wrapper.qa('button')).find(btn =>
      (btn.textContent || '').includes('Save')
    ) as HTMLButtonElement | undefined;
    expect(save).toBeTruthy();
    expect(save!.getAttribute('type')).toBe('button');
    expect(typeof wrapper.root.submit).toBe('function');
    await wrapper.root.submit();
    await flushPromises();
    expect(submitHandler.calls.length).toBe(1);
    wrapper.unmount();
  });

  test('validate-failed with showMessages=false skips handler and toast', async () => {
    const errors: string[] = [];
    const origError = ChoyMessage.error;
    ChoyMessage.error = ((msg: string) => {
      errors.push(String(msg));
    }) as typeof ChoyMessage.error;
    try {
      const { wrapper, submitHandler } = mountForm({ failValidate: true, showMessages: false });
      await flushPromises();
      wrapper.click('[data-test="slot-submit"]');
      const form = wrapper.q('form');
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
      await flushPromises();
      expect(submitHandler.calls.length).toBe(0);
      expect(errors.length).toBe(0);
      wrapper.unmount();
    } finally {
      ChoyMessage.error = origError;
    }
  });
});
