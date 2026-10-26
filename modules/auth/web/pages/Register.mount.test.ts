// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import Register from './Register.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';

async function mountRegister() {
  const replaces: unknown[] = [];
  const wrapper = mount(Register as any, {
    global: buildPageMountGlobal({
      route: { path: '/register', query: {} },
      router: {
        replace: (to: unknown) => {
          replaces.push(to);
          return Promise.resolve(to);
        },
      },
    }),
  });
  await flushPromises();
  return { wrapper, replaces };
}

function fieldInput(wrapper: { find: (sel: string) => any }, selector: string) {
  const root = wrapper.find(selector);
  if (!root.exists()) return root;
  const el = root.element as HTMLElement;
  if (String(el.tagName || '').toLowerCase() === 'input') return root;
  const inner = typeof el.querySelector === 'function' ? el.querySelector('input') : null;
  if (inner) {
    return {
      exists: () => true,
      element: inner,
      trigger: async (type: string) => {
        inner.dispatchEvent(new Event(type, { bubbles: true }));
      },
    };
  }
  return root;
}

async function fillField(wrapper: { find: (sel: string) => any }, selector: string, value: string) {
  const input = fieldInput(wrapper, selector);
  const el = input.element as HTMLInputElement;
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
  el.dispatchEvent(new Event('change', { bubbles: true, cancelable: true }));
  el.dispatchEvent(new Event('blur', { bubbles: true, cancelable: true }));
  await flushPromises();
}

function submitForm(wrapper: { find: (sel: string) => any }) {
  const form = wrapper.find('form');
  (form.element as HTMLFormElement).dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
}

async function afterSubmit() {
  for (let i = 0; i < 8; i++) await flushPromises();
}

function queryIn(wrapper: { find: (sel: string) => any }, rootSelector: string, sel: string): HTMLElement | null {
  const root = wrapper.find(rootSelector);
  if (!root.exists()) return null;
  const el = root.element as HTMLElement;
  return typeof el.querySelector === 'function' ? (el.querySelector(sel) as HTMLElement | null) : null;
}

/**
 * Toggle AgreeTerms via the kit checkbox (Reka CheckboxRoot button, not a native <input>).
 * Minimal DOM only supports simple selectors (#id / .class / tag / [attr=value]).
 */
async function checkTerms(wrapper: { find: (sel: string) => any }) {
  const terms = wrapper.find('[data-testid="register-terms"]');
  expect(terms.exists()).toBe(true);
  const root = terms.element as HTMLElement;
  const box =
    (root.querySelector('#fld-AgreeTerms') as HTMLElement | null) ||
    (root.querySelector('.choy-bool-input') as HTMLElement | null);
  expect(!!box).toBe(true);
  if (typeof (box as { click?: () => void }).click === 'function') {
    (box as { click: () => void }).click();
  } else {
    box!.dispatchEvent(new Event('click', { bubbles: true, cancelable: true }));
  }
  await flushPromises();
}

/** Submit control — not the first <button> (kit checkbox is also a button). */
function findSubmitButton(wrapper: { find: (sel: string) => any }): HTMLButtonElement | null {
  const card = wrapper.find('.register-card');
  if (!card.exists()) return null;
  const root = card.element as HTMLElement;
  const buttons = typeof root.querySelectorAll === 'function' ? root.querySelectorAll('button') : [];
  for (let i = 0; i < buttons.length; i++) {
    const btn = buttons[i] as HTMLButtonElement;
    if (btn.getAttribute('type') === 'submit') return btn;
  }
  return null;
}

test('Register.vue mounts under choysumMount', async () => {
  const { wrapper } = await mountRegister();
  expect(fieldInput(wrapper, '.register-username').exists()).toBe(true);
  expect(wrapper.find('[data-testid="register-terms"]').exists()).toBe(true);
  const root = wrapper.element as HTMLElement;
  const labels = Array.from(root.querySelectorAll('label'));
  for (const id of ['register-username', 'register-email', 'register-password', 'register-confirm']) {
    expect(labels.some(el => el.getAttribute('for') === id)).toBe(true);
    const control = root.querySelector(`#${id}`);
    expect(control).not.toBeNull();
    expect(['INPUT', 'TEXTAREA', 'SELECT'].includes(String(control?.tagName || ''))).toBe(true);
  }
  expect(labels.some(el => el.getAttribute('for') === 'fld-AgreeTerms')).toBe(true);
  expect(root.querySelector('#fld-AgreeTerms')).not.toBeNull();
  wrapper.unmount();
});

test('Register.vue: terms side copy does not inherit the field w-full class', async () => {
  const { wrapper } = await mountRegister();
  const terms = wrapper.find('[data-testid="register-terms"]');
  expect(terms.exists()).toBe(true);
  const root = terms.element as HTMLElement;
  const editor = () => root.querySelector('.choy-bool-editor') as HTMLElement | null;
  const wrap = () => root.querySelector('.choy-field-base__inline-wrap') as HTMLElement | null;
  const side = () => root.querySelector('.flex-1') as HTMLElement | null;
  expect(!!editor()).toBe(true);
  expect(String(editor()!.className).includes('w-full')).toBe(false);
  expect(!!wrap()).toBe(true);
  expect(String(wrap()!.className).includes('w-full')).toBe(true);
  expect(String(wrap()!.className).includes('min-w-0')).toBe(true);
  expect(!!side()).toBe(true);
  expect(String(side()!.textContent || '').includes('Terms of Service')).toBe(true);

  await fillField(wrapper, '.register-username', 'alice');
  await fillField(wrapper, '.register-email', 'alice@example.com');
  await fillField(wrapper, '.register-password', 'secret1');
  await fillField(wrapper, '.register-confirm', 'secret1');
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.text().includes('You must agree to the Terms of Service and Privacy Policy')).toBe(true);
  expect(String(editor()!.className).includes('w-full')).toBe(false);
  expect(String(wrap()!.className).includes('w-full')).toBe(true);
  expect(String(wrap()!.className).includes('min-w-0')).toBe(true);
  expect(String(side()!.textContent || '').includes('Terms of Service')).toBe(true);
  wrapper.unmount();
});

test('Register.vue: empty submit shows inline errors and does not register', async () => {
  const { wrapper, replaces } = await mountRegister();
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.text().includes('Enter username')).toBe(true);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Register.vue: confirm mismatch does not register', async () => {
  const { wrapper, replaces } = await mountRegister();
  await fillField(wrapper, '.register-username', 'alice');
  await fillField(wrapper, '.register-email', 'alice@example.com');
  await fillField(wrapper, '.register-password', 'secret1');
  await fillField(wrapper, '.register-confirm', 'other99');
  await checkTerms(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.text().includes('Passwords do not match')).toBe(true);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Register.vue: unchecked terms keeps submit disabled and does not register', async () => {
  const { wrapper, replaces } = await mountRegister();
  await fillField(wrapper, '.register-username', 'alice');
  await fillField(wrapper, '.register-email', 'alice@example.com');
  await fillField(wrapper, '.register-password', 'secret1');
  await fillField(wrapper, '.register-confirm', 'secret1');
  const submit = findSubmitButton(wrapper);
  expect(!!submit).toBe(true);
  expect(submit!.disabled).toBe(true);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Register.vue: valid submit registers and redirects', async () => {
  const { wrapper, replaces } = await mountRegister();
  await fillField(wrapper, '.register-username', 'alice');
  await fillField(wrapper, '.register-email', 'alice@example.com');
  await fillField(wrapper, '.register-password', 'secret1');
  await fillField(wrapper, '.register-confirm', 'secret1');
  await checkTerms(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.text().includes('Passwords do not match')).toBe(false);
  expect(replaces).toEqual(['/']);
  wrapper.unmount();
});
