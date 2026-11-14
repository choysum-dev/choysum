// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@choysum/test-utils';
import { getActivePinia, setActivePinia, type Pinia } from 'pinia';
import Setup from './Setup.vue';
import { buildPageMountGlobal } from '@choysum/page-mount';
import { useAuthStore, type CompleteInitialSetupInput, type InitialSetupOptions } from '../stores/auth';
import { ChoysumError } from '../error';
import { clearSetupDraft, saveSetupDraft } from './setup_draft';
import { ChoyMessage } from '@/web/web/composables/useChoyMessage';
import { setSetupLoadDemoRequesterForTests } from './setup_form';
import {
  getLoadingRegistryDebugState,
  resetLoadingRegistryForTests,
} from '@/core/web/client/loadingRegistry';

function setupOptions(): InitialSetupOptions {
  return {
    Countries: [
      { Code: 'CN', Name: 'China', DefaultCurrencyCode: 'CNY' },
      { Code: 'US', Name: 'United States', DefaultCurrencyCode: 'USD' },
    ],
    Currencies: [
      { Code: 'CNY', Name: 'Chinese Yuan' },
      { Code: 'USD', Name: 'US Dollar' },
    ],
    Timezones: [
      { Value: 'Asia/Shanghai', Label: 'Asia/Shanghai' },
      { Value: 'UTC', Label: 'UTC' },
    ],
    Languages: [
      { Code: 'en_US', Name: 'English' },
      { Code: 'zh_CN', Name: '简体中文' },
    ],
    Defaults: {
      OrganizationName: 'My Company',
      CountryCode: 'CN',
      CurrencyCode: 'CNY',
      Timezone: 'Asia/Shanghai',
    },
  };
}

type SetupStoreStub = {
  completeInitialSetup: (req: CompleteInitialSetupInput) => Promise<unknown>;
  completeInitialSetupCore: (req: CompleteInitialSetupInput) => Promise<unknown>;
  login: (username: string, password: string) => Promise<unknown>;
  loginCore: (username: string, password: string) => Promise<unknown>;
  ensureSetupStateReady: () => Promise<boolean>;
  getInitialSetupOptions: () => Promise<InitialSetupOptions>;
};

let optionsCalls = 0;

async function mountSetup(opts?: {
  query?: Record<string, string>;
  failSetup?: ChoysumError;
  failLogin?: string;
  /** When true, router.replace rejects (covers login-fallback recovery messaging). */
  failReplace?: boolean;
  failOptions?: ChoysumError;
  setupReady?: boolean;
  draft?: Record<string, unknown>;
  /** Terminology language reported by the FE-unit i18n stub store. */
  uiLang?: string;
}) {
  clearSetupDraft();
  if (opts?.draft) saveSetupDraft(opts.draft);
  const replaces: unknown[] = [];
  const requests: CompleteInitialSetupInput[] = [];
  const logins: Array<[string, string]> = [];
  const global = buildPageMountGlobal({
    route: { path: '/setup', query: opts?.query || {} },
    router: {
      replace: (to: unknown) => {
        replaces.push(to);
        if (opts?.failReplace) return Promise.reject(new Error('nav boom'));
        return Promise.resolve(to);
      },
    },
  });
  const pinia = global.plugins[0] as Pinia;
  setActivePinia(pinia);
  const store = useAuthStore() as unknown as SetupStoreStub;
  optionsCalls = 0;
  store.getInitialSetupOptions = async () => {
    optionsCalls += 1;
    if (opts?.failOptions) throw opts.failOptions;
    const resp = setupOptions();
    if (optionsCalls > 1) resp.Currencies[0].Name = '人民币';
    return resp;
  };
  store.ensureSetupStateReady = async () => !!opts?.setupReady;
  const completeSetup = async (req: CompleteInitialSetupInput) => {
    requests.push(req);
    if (opts?.failSetup) throw opts.failSetup;
    return { UserId: 'usr_admin' };
  };
  store.completeInitialSetup = completeSetup;
  store.completeInitialSetupCore = completeSetup;
  const login = async (username: string, password: string) => {
    logins.push([username, password]);
    if (opts?.failLogin) {
      throw new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message: opts.failLogin });
    }
    return true;
  };
  store.login = login;
  store.loginCore = login;
  const wrapper = mount(Setup as any, { global });
  if (opts?.uiLang) setUiLang(opts.uiLang);
  for (let i = 0; i < 4; i++) await flushPromises();
  return { wrapper, replaces, requests, logins };
}

/** Product SFCs resolve the FE-unit i18n stub store; drive its terminologyLang directly. */
function setUiLang(lang: string) {
  const pinia = getActivePinia() as unknown as { _s: Map<string, Record<string, unknown>> };
  const i18n = pinia._s.get('i18n-fe-stub');
  expect(!!i18n).toBe(true);
  i18n!.terminologyLang = lang;
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

async function fillAdmin(wrapper: { find: (sel: string) => any }, email = '') {
  await fillField(wrapper, '.setup-username', 'admin');
  await fillField(wrapper, '.setup-password', 'secret1');
  await fillField(wrapper, '.setup-confirm', 'secret1');
  if (email) await fillField(wrapper, '.setup-email', email);
}

async function advanceToReview(wrapper: { find: (sel: string) => any }, email = '') {
  await fillAdmin(wrapper, email);
  submitForm(wrapper);
  await afterSubmit();
  submitForm(wrapper);
  await afterSubmit();
}

/**
 * Toggle WithDemo via the kit checkbox (Reka CheckboxRoot button, not a native <input>).
 * Minimal DOM only supports simple selectors (#id / .class / tag / [attr=value]).
 */
async function checkWithDemo(wrapper: { find: (sel: string) => any }) {
  const demo = wrapper.find('[data-testid="setup-review-demo"]');
  expect(demo.exists()).toBe(true);
  const root = demo.element as HTMLElement;
  const box =
    (root.querySelector('#fld-WithDemo') as HTMLElement | null) ||
    (root.querySelector('.choy-bool-input') as HTMLElement | null);
  expect(!!box).toBe(true);
  if (typeof (box as { click?: () => void }).click === 'function') {
    (box as { click: () => void }).click();
  } else {
    box!.dispatchEvent(new Event('click', { bubbles: true, cancelable: true }));
  }
  await flushPromises();
}

function stepItems(wrapper: { find: (sel: string) => any }): HTMLElement[] {
  const root = wrapper.find('[data-anchor="choy.stepper"]').element as HTMLElement;
  return Array.from(root.querySelectorAll('[data-step]')) as HTMLElement[];
}

function activeStep(wrapper: { find: (sel: string) => any }): string {
  const active = stepItems(wrapper).find(el => el.getAttribute('data-state') === 'active');
  return active?.getAttribute('data-step') || '';
}

test('Setup.vue mounts with a three-step stepper and admin fields', async () => {
  const { wrapper } = await mountSetup();
  expect(wrapper.find('[data-anchor="choy.stepper"]').exists()).toBe(true);
  expect(stepItems(wrapper).length).toBe(3);
  expect(activeStep(wrapper)).toBe('admin');
  expect(fieldInput(wrapper, '.setup-username').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-password').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-confirm').exists()).toBe(true);
  expect(fieldInput(wrapper, '.setup-email').exists()).toBe(true);
  expect(wrapper.text().includes('Initial Setup') || wrapper.text().includes('初始设置')).toBe(true);
  wrapper.unmount();
});

test('Setup.vue: empty Next stays on the admin step and shows field errors', async () => {
  const { wrapper, replaces, requests } = await mountSetup();
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('admin');
  expect(wrapper.text().includes('Enter username') || wrapper.find('.text-destructive').exists()).toBe(true);
  expect(requests).toEqual([]);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Setup.vue: invalid optional email blocks the admin step', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper, 'not-an-email');
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('admin');
  expect(wrapper.text().includes('Enter a valid email address') || wrapper.text().includes('有效')).toBe(true);
  wrapper.unmount();
});

test('Setup.vue: Next walks admin → organization → review, Back returns', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('organization');
  expect(fieldInput(wrapper, '.setup-org-name').exists()).toBe(true);
  expect(wrapper.find('.setup-currency').exists()).toBe(true);
  expect(wrapper.find('.setup-timezone').exists()).toBe(true);

  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('review');
  expect(wrapper.find('.setup-review-step').exists()).toBe(true);
  expect(wrapper.text().includes('My Company')).toBe(true);
  expect(wrapper.text().includes('CNY · Chinese Yuan')).toBe(true);
  expect(wrapper.find('[data-testid="setup-review-demo"]').exists()).toBe(true);
  expect(wrapper.text().includes('Optional') || wrapper.text().includes('可选')).toBe(true);
  expect(
    wrapper.text().includes('Load demo data') || wrapper.text().includes('加载演示数据'),
  ).toBe(true);

  const backEl = wrapper.find('.setup-back').element as HTMLElement;
  let backClicks = 0;
  backEl.addEventListener('click', () => backClicks++);
  backEl.dispatchEvent(new Event('click', { bubbles: true, cancelable: true }));
  await afterSubmit();
  expect(backClicks).toBe(1);
  expect(activeStep(wrapper)).toBe('organization');
  wrapper.unmount();
});

test('Setup.vue: review submit sends sectioned request, logs in, and redirects', async () => {
  resetLoadingRegistryForTests();
  const { wrapper, replaces, requests, logins } = await mountSetup({ query: { redirect: '/auth/users' } });
  await advanceToReview(wrapper, 'owner@acme.io');
  submitForm(wrapper);
  await afterSubmit();
  expect(wrapper.find('.setup-error').exists()).toBe(false);
  expect(requests.length).toBe(1);
  const req = requests[0];
  expect(req.Admin.Username).toBe('admin');
  expect(req.Admin.Password).toBe('secret1');
  expect(req.Admin.Email).toBe('owner@acme.io');
  expect(req.Organization).toEqual({
    Name: 'My Company',
    CountryCode: 'CN',
    CurrencyCode: 'CNY',
    Timezone: req.Organization.Timezone,
  });
  // Browser timezone wins when listed; otherwise the company default.
  expect(['Asia/Shanghai', 'UTC'].includes(req.Organization.Timezone)).toBe(true);
  expect(logins).toEqual([['admin', 'secret1']]);
  expect(replaces).toEqual(['/auth/users']);
  // Success hands off a nav chrome ticket before leaving /setup.
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  expect(getLoadingRegistryDebugState().barCount).toBe(1);
  resetLoadingRegistryForTests();
  wrapper.unmount();
});

test('Setup.vue: unchecked WithDemo does not call RequestLoadDemo', async () => {
  const loadDemoCalls: unknown[] = [];
  setSetupLoadDemoRequesterForTests(async () => {
    loadDemoCalls.push({});
    return { Ok: true, Id: 'op-demo' };
  });
  try {
    const { wrapper, replaces, requests, logins } = await mountSetup({
      query: { redirect: '/auth/users' },
    });
    await advanceToReview(wrapper);
    expect(wrapper.find('[data-testid="setup-review-demo"]').exists()).toBe(true);
    submitForm(wrapper);
    await afterSubmit();
    expect(requests.length).toBe(1);
    expect(logins).toEqual([['admin', 'secret1']]);
    expect(loadDemoCalls.length).toBe(0);
    expect(replaces).toEqual(['/auth/users']);
    wrapper.unmount();
  } finally {
    setSetupLoadDemoRequesterForTests(null);
  }
});

test('Setup.vue: checked WithDemo calls RequestLoadDemo after login', async () => {
  const loadDemoCalls: unknown[] = [];
  setSetupLoadDemoRequesterForTests(async () => {
    loadDemoCalls.push({});
    return { Ok: true, Id: 'op-demo' };
  });
  try {
    const { wrapper, replaces, requests, logins } = await mountSetup({
      query: { redirect: '/auth/users' },
    });
    await advanceToReview(wrapper);
    await checkWithDemo(wrapper);
    submitForm(wrapper);
    await afterSubmit();
    expect(requests.length).toBe(1);
    expect(logins).toEqual([['admin', 'secret1']]);
    expect(loadDemoCalls.length).toBe(1);
    expect(replaces).toEqual(['/auth/users']);
    wrapper.unmount();
  } finally {
    setSetupLoadDemoRequesterForTests(null);
  }
});

test('Setup.vue: RequestLoadDemo failure still redirects and warns', async () => {
  const warnings: string[] = [];
  const origWarning = ChoyMessage.warning;
  ChoyMessage.warning = ((m: string) => {
    warnings.push(m);
  }) as unknown as typeof ChoyMessage.warning;
  setSetupLoadDemoRequesterForTests(async () => ({ Ok: false, ErrorMessage: 'demo busy' }));
  try {
    const { wrapper, replaces, requests, logins } = await mountSetup({
      query: { redirect: '/auth/users' },
    });
    await advanceToReview(wrapper);
    await checkWithDemo(wrapper);
    submitForm(wrapper);
    await afterSubmit();
    expect(requests.length).toBe(1);
    expect(logins).toEqual([['admin', 'secret1']]);
    expect(warnings).toEqual(['demo busy']);
    expect(replaces).toEqual(['/auth/users']);
    wrapper.unmount();
  } finally {
    ChoyMessage.warning = origWarning;
    setSetupLoadDemoRequesterForTests(null);
  }
});

test('Setup.vue: server field error jumps to its step without a header error', async () => {
  const failSetup = new ChoysumError({
    domain: 'auth',
    code: 'VALIDATION_FAILED',
    message: 'Username is already taken',
  }).withMetadata({ Field: 'Admin.Username' });
  const { wrapper, replaces } = await mountSetup({ failSetup });
  await advanceToReview(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('admin');
  expect(wrapper.find('.setup-error').exists()).toBe(false);
  expect(wrapper.text().includes('Username is already taken')).toBe(true);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Setup.vue: setup RPC failure without a field shows the header error', async () => {
  const failSetup = new ChoysumError({ domain: 'auth', code: 'UNKNOWN', message: 'already done' });
  const { wrapper, replaces } = await mountSetup({ failSetup });
  await advanceToReview(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  const alert = wrapper.find('.setup-error');
  expect(alert.exists()).toBe(true);
  expect(wrapper.text().includes('already done')).toBe(true);
  expect(replaces).toEqual([]);
  const close = (alert.element as HTMLElement).querySelector('button');
  close!.dispatchEvent(new Event('click', { bubbles: true, cancelable: true }));
  await flushPromises();
  expect(wrapper.find('.setup-error').exists()).toBe(false);
  wrapper.unmount();
});

test('Setup.vue: setup already complete elsewhere navigates to login', async () => {
  const failSetup = new ChoysumError({
    domain: 'auth',
    code: 'INITIAL_SETUP_ALREADY_COMPLETE',
    message: 'already done',
  });
  const { wrapper, replaces } = await mountSetup({
    query: { redirect: '/auth/users' },
    failSetup,
    setupReady: true,
  });
  await advanceToReview(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(replaces[replaces.length - 1]).toEqual({ name: 'login', query: { redirect: '/auth/users' } });
  wrapper.unmount();
});

test('Setup.vue: login failure after setup goes to login with setup=done', async () => {
  const { wrapper, replaces } = await mountSetup({ query: { redirect: '/auth/users' }, failLogin: 'bad login' });
  await advanceToReview(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(replaces[replaces.length - 1]).toEqual({
    name: 'login',
    query: { redirect: '/auth/users', setup: 'done' },
  });
  wrapper.unmount();
});

test('Setup.vue: login-fallback navigation failure shows recovery message', async () => {
  const { wrapper, replaces } = await mountSetup({
    query: { redirect: '/auth/users' },
    failLogin: 'bad login',
    failReplace: true,
  });
  await advanceToReview(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(replaces.length).toBeGreaterThan(0);
  expect(wrapper.text()).toContain('Setup is complete, but the login page could not open');
  wrapper.unmount();
});

test('Setup.vue: success-path navigation failure shows recovery message', async () => {
  const { wrapper, replaces } = await mountSetup({
    query: { redirect: '/auth/users' },
    failReplace: true,
  });
  await advanceToReview(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(replaces.length).toBeGreaterThan(0);
  expect(wrapper.text()).toContain('Setup is complete, but the page could not open');
  wrapper.unmount();
});

test('Setup.vue: options failure shows a retry prompt', async () => {
  const failOptions = new ChoysumError({ domain: 'base', code: 'UNKNOWN', message: 'options down' });
  const { wrapper, replaces } = await mountSetup({ failOptions });
  expect(wrapper.find('.setup-options-error').exists()).toBe(true);
  expect(wrapper.text().includes('options down')).toBe(true);
  expect(wrapper.find('.setup-options-retry').exists()).toBe(true);
  expect(wrapper.find('form').exists()).toBe(false);
  expect(replaces).toEqual([]);
  wrapper.unmount();
});

test('Setup.vue: options rejected after setup completed navigates to login', async () => {
  const failOptions = new ChoysumError({
    domain: 'setupgate',
    code: 'FAILED_PRECONDITION',
    message: 'initial setup is not pending',
  });
  const { wrapper, replaces } = await mountSetup({ failOptions, setupReady: true });
  expect(replaces[replaces.length - 1]).toEqual({ name: 'login', query: {} });
  wrapper.unmount();
});

test('Setup.vue: restores non-secret draft values on mount', async () => {
  const { wrapper } = await mountSetup({ draft: { Username: 'owner', Password: 'secret1', OrgName: 'Acme' } });
  const username = fieldInput(wrapper, '.setup-username').element as HTMLInputElement;
  const password = fieldInput(wrapper, '.setup-password').element as HTMLInputElement;
  if (typeof sessionStorage !== 'undefined') {
    expect(username.value).toBe('owner');
  }
  expect(password.value).toBe('');
  expect(activeStep(wrapper)).toBe('admin');
  wrapper.unmount();
  clearSetupDraft();
});

test('Setup.vue: UI language change refetches localized option labels', async () => {
  const { wrapper } = await mountSetup();
  await advanceToReview(wrapper);
  expect(optionsCalls).toBe(1);
  expect(wrapper.text().includes('CNY · Chinese Yuan')).toBe(true);
  setUiLang('zh_CN');
  for (let i = 0; i < 8; i++) await flushPromises();
  expect(optionsCalls).toBe(2);
  expect(wrapper.text().includes('CNY · 人民币')).toBe(true);
  expect(activeStep(wrapper)).toBe('review');
  wrapper.unmount();
});

test('Setup.vue: admin language follows the current UI language', async () => {
  const { wrapper, requests } = await mountSetup({ uiLang: 'en_US' });
  setUiLang('zh_CN');
  for (let i = 0; i < 8; i++) await flushPromises();
  await advanceToReview(wrapper);
  expect(wrapper.find('[data-field="OrgLanguageCode"]').exists()).toBe(false);
  submitForm(wrapper);
  await afterSubmit();
  expect(requests.length).toBe(1);
  expect(requests[0].Admin.LanguageCode).toBe('zh_CN');
  expect('LanguageCode' in requests[0].Organization).toBe(false);
  wrapper.unmount();
});

function pressEnter(el: Element, init?: { isComposing?: boolean }) {
  const ev = new Event('keydown', { bubbles: true, cancelable: true });
  Object.defineProperty(ev, 'key', { value: 'Enter' });
  Object.defineProperty(ev, 'keyCode', { value: init?.isComposing ? 229 : 13 });
  Object.defineProperty(ev, 'isComposing', { value: !!init?.isComposing });
  el.dispatchEvent(ev);
  return ev;
}

test('Setup.vue: Enter in a text input advances the step, but not during IME composition', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  const confirm = fieldInput(wrapper, '.setup-confirm').element as HTMLInputElement;
  const composing = pressEnter(confirm, { isComposing: true });
  await afterSubmit();
  expect(composing.defaultPrevented).toBe(false);
  expect(activeStep(wrapper)).toBe('admin');
  const enter = pressEnter(confirm);
  await afterSubmit();
  expect(enter.defaultPrevented).toBe(true);
  expect(activeStep(wrapper)).toBe('organization');
  wrapper.unmount();
});

test('Setup.vue: Enter on a closed organization Selection trigger advances to review', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('organization');
  const tzRoot = wrapper.find('.setup-timezone').element as HTMLElement;
  const trigger = tzRoot.querySelector('[data-slot="combobox-trigger"]') as HTMLElement | null;
  expect(!!trigger).toBe(true);
  trigger!.setAttribute('aria-expanded', 'false');
  trigger!.setAttribute('data-state', 'closed');
  const enter = pressEnter(trigger!);
  await afterSubmit();
  expect(enter.defaultPrevented).toBe(true);
  expect(activeStep(wrapper)).toBe('review');
  wrapper.unmount();
});

test('Setup.vue: Enter on an open organization Selection trigger does not advance', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('organization');
  const tzRoot = wrapper.find('.setup-timezone').element as HTMLElement;
  const trigger = tzRoot.querySelector('[data-slot="combobox-trigger"]') as HTMLElement | null;
  expect(!!trigger).toBe(true);
  trigger!.setAttribute('aria-expanded', 'true');
  trigger!.setAttribute('data-state', 'open');
  const enter = pressEnter(trigger!);
  await afterSubmit();
  expect(enter.defaultPrevented).toBe(false);
  expect(activeStep(wrapper)).toBe('organization');
  wrapper.unmount();
});

test('Setup.vue: Enter on an open combobox search input does not advance', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('organization');
  const form = wrapper.find('form').element as HTMLFormElement;
  const search = document.createElement('input');
  search.setAttribute('role', 'combobox');
  search.setAttribute('aria-expanded', 'true');
  form.appendChild(search);
  const enter = pressEnter(search);
  await afterSubmit();
  expect(enter.defaultPrevented).toBe(false);
  expect(activeStep(wrapper)).toBe('organization');
  search.remove();
  wrapper.unmount();
});

test('Setup.vue: after Next, Enter advances without clicking a field first', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  submitForm(wrapper);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('organization');
  const orgInput = fieldInput(wrapper, '.setup-org-name').element as HTMLElement;
  expect(document.activeElement).toBe(orgInput);
  pressEnter(orgInput);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('review');
  wrapper.unmount();
});

test('Setup.vue: Enter with focus on body still advances when admin is filled', async () => {
  const { wrapper } = await mountSetup();
  await fillAdmin(wrapper);
  const active = document.activeElement as HTMLElement | null;
  if (active && typeof active.blur === 'function') active.blur();
  pressEnter(document.body);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('organization');
  wrapper.unmount();
});

test('Setup.vue: Enter outside the setup panel does not submit the wizard', async () => {
  const { wrapper, requests } = await mountSetup();
  await advanceToReview(wrapper);
  const outsider = document.createElement('div');
  outsider.setAttribute('data-testid', 'outside-setup');
  document.body.appendChild(outsider);
  pressEnter(outsider);
  await afterSubmit();
  expect(requests).toEqual([]);
  expect(activeStep(wrapper)).toBe('review');
  outsider.remove();
  wrapper.unmount();
});

test('Setup.vue: Enter on review demo checkbox completes setup', async () => {
  const { wrapper, requests } = await mountSetup();
  await advanceToReview(wrapper);
  const demo = wrapper.find('.setup-review-demo').element as HTMLElement;
  const checkbox = demo.querySelector('[data-slot="checkbox"]') as HTMLElement | null;
  expect(!!checkbox).toBe(true);
  pressEnter(checkbox!);
  await afterSubmit();
  expect(requests.length).toBe(1);
  wrapper.unmount();
});

test('Setup.vue: Enter on an empty admin step still validates and stays put', async () => {
  const { wrapper, requests } = await mountSetup();
  const username = fieldInput(wrapper, '.setup-username').element as HTMLInputElement;
  pressEnter(username);
  await afterSubmit();
  expect(activeStep(wrapper)).toBe('admin');
  expect(requests).toEqual([]);
  wrapper.unmount();
});
