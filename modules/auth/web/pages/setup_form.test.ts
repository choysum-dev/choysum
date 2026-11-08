// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '../error';
import type { CompleteInitialSetupInput } from '../stores/auth';
import {
  buildCompleteInitialSetupReq,
  requestMetaModuleLoadDemo,
  requestSetupLoadDemo,
  resolveSetupLoadDemoRequester,
  runSetupLoadDemo,
  runSetupSubmit,
  setSetupLoadDemoRequesterForTests,
  setupConfirmPasswordError,
  setupEmailError,
  setupLocalFieldForServerField,
  setupLoginFallbackTarget,
  setupOrgNameError,
  setupPasswordError,
  setupStepForServerField,
  setupUsernameError,
  setupWantsDemo,
  withSetupServerError,
} from './setup_form';

const t = (s: string) => s;

function req(username = ' admin ', password = 'secret1'): CompleteInitialSetupInput {
  return {
    Admin: { Username: username, Password: password, LanguageCode: 'en_US' },
    Organization: { Name: 'Acme', CurrencyCode: 'CNY', Timezone: 'Asia/Shanghai' },
  };
}

test('setupLoginFallbackTarget preserves query and optionally marks setup done', () => {
  expect(setupLoginFallbackTarget({ redirect: '/auth/users' })).toEqual({
    name: 'login',
    query: { redirect: '/auth/users' },
  });
  expect(setupLoginFallbackTarget(undefined)).toEqual({ name: 'login', query: {} });
  expect(setupLoginFallbackTarget({ redirect: '/x' }, { setupDone: true })).toEqual({
    name: 'login',
    query: { redirect: '/x', setup: 'done' },
  });
});

test('setupUsernameError rejects short or empty names', () => {
  expect(setupUsernameError('', t)).toBe('Enter username');
  expect(setupUsernameError('ab', t)).toBe('Username must be at least 3 characters');
  expect(setupUsernameError('admin', t)).toBe('');
});

test('setupPasswordError requires min length', () => {
  expect(setupPasswordError('', t)).toBe('Enter password');
  expect(setupPasswordError('12345', t)).toBe('Password must be at least 6 characters');
  expect(setupPasswordError('secret1', t)).toBe('');
});

test('setupConfirmPasswordError requires match', () => {
  expect(setupConfirmPasswordError('', 'secret1', t)).toBe('Re-enter password');
  expect(setupConfirmPasswordError('other', 'secret1', t)).toBe('Passwords do not match');
  expect(setupConfirmPasswordError('secret1', 'secret1', t)).toBe('');
});

test('setupEmailError: optional, format and length checked when present', () => {
  expect(setupEmailError('', t)).toBe('');
  expect(setupEmailError('   ', t)).toBe('');
  expect(setupEmailError('owner@acme.io', t)).toBe('');
  expect(setupEmailError('admin@local', t)).toBe('Enter a valid email address');
  expect(setupEmailError(`${'a'.repeat(95)}@acme.io`, t)).toBe('Email is too long');
});

test('setupOrgNameError: required and capped at 100 characters', () => {
  expect(setupOrgNameError('  ', t)).toBe('Enter organization name');
  expect(setupOrgNameError('Acme', t)).toBe('');
  expect(setupOrgNameError('x'.repeat(101), t)).toBe('Organization name must be at most 100 characters');
});

test('server field mapping resolves local field and owning step', () => {
  expect(setupLocalFieldForServerField('Admin.Username')).toBe('Username');
  expect(setupStepForServerField('Admin.Email')).toBe('admin');
  expect(setupLocalFieldForServerField('Organization.Name')).toBe('OrgName');
  expect(setupStepForServerField('Organization.CurrencyCode')).toBe('organization');
  expect(setupStepForServerField('Organization.LanguageCode')).toBe(null);
  expect(setupStepForServerField('Admin.LanguageCode')).toBe(null);
  expect(setupLocalFieldForServerField('Unknown')).toBe('');
});

test('withSetupServerError appends a failing rule only when a message is set', async () => {
  const base = [{ required: true }];
  expect(withSetupServerError(base, '')).toBe(base);
  const rules = withSetupServerError(base, 'taken');
  expect(rules.length).toBe(2);
  const err = await new Promise<Error | undefined>(resolve => rules[1].validator!({}, 'x', resolve));
  expect(err?.message).toBe('taken');
});

test('buildCompleteInitialSetupReq trims values and omits empty optionals', () => {
  expect(
    buildCompleteInitialSetupReq(
      {
        Username: ' admin ',
        Password: ' pw with spaces ',
        Email: '  ',
        OrgName: ' Acme ',
        CountryCode: '',
        CurrencyCode: 'USD',
        Timezone: 'UTC',
      },
      'zh_CN',
    ),
  ).toEqual({
    Admin: { Username: 'admin', Password: ' pw with spaces ', LanguageCode: 'zh_CN' },
    Organization: { Name: 'Acme', CurrencyCode: 'USD', Timezone: 'UTC' },
  });
  const full = buildCompleteInitialSetupReq({ Username: 'a', Password: 'p', Email: 'o@a.io', CountryCode: 'US' }, 'en_US');
  expect(full.Admin.Email).toBe('o@a.io');
  expect(full.Organization.CountryCode).toBe('US');
});

test('setupWantsDemo is true only for an explicit boolean true', () => {
  expect(setupWantsDemo(undefined)).toBe(false);
  expect(setupWantsDemo({})).toBe(false);
  expect(setupWantsDemo({ WithDemo: false })).toBe(false);
  expect(setupWantsDemo({ WithDemo: 'true' })).toBe(false);
  expect(setupWantsDemo({ WithDemo: true })).toBe(true);
});

test('resolveSetupLoadDemoRequester prefers the test override', async () => {
  const fallback = async () => ({ Ok: true, Id: 'fallback' });
  expect(resolveSetupLoadDemoRequester(fallback)).toBe(fallback);
  setSetupLoadDemoRequesterForTests(async () => ({ Ok: false, ErrorMessage: 'override' }));
  try {
    const resolved = resolveSetupLoadDemoRequester(fallback);
    expect(resolved).not.toBe(fallback);
    expect(await resolved()).toEqual({ Ok: false, ErrorMessage: 'override' });
  } finally {
    setSetupLoadDemoRequesterForTests(null);
  }
  expect(resolveSetupLoadDemoRequester(fallback)).toBe(fallback);
});

test('requestMetaModuleLoadDemo soft-oks when meta store is missing or incomplete', async () => {
  expect(
    await requestMetaModuleLoadDemo(() => {
      throw new Error('factory missing');
    }),
  ).toEqual({ Ok: true });
  expect(await requestMetaModuleLoadDemo(() => ({}))).toEqual({ Ok: true });
  expect(await requestMetaModuleLoadDemo(() => null)).toEqual({ Ok: true });
  expect(
    await requestMetaModuleLoadDemo(() => ({
      RequestLoadDemo: async () => ({ Ok: true, Id: 'op-1' }),
    })),
  ).toEqual({ Ok: true, Id: 'op-1' });
  expect(
    await requestMetaModuleLoadDemo(() => ({
      RequestLoadDemo: async () => ({ Ok: false, ErrorMessage: 'busy' }),
    })),
  ).toEqual({ Ok: false, ErrorMessage: 'busy' });
});

test('requestSetupLoadDemo honours the test override and production requester', async () => {
  setSetupLoadDemoRequesterForTests(async () => ({ Ok: false, ErrorMessage: 'from-test' }));
  try {
    expect(await requestSetupLoadDemo()).toEqual({ Ok: false, ErrorMessage: 'from-test' });
  } finally {
    setSetupLoadDemoRequesterForTests(null);
  }
});

test('runSetupLoadDemo soft-succeeds and reports bridge Ok:false without throwing', async () => {
  const failures: string[] = [];
  expect(
    await runSetupLoadDemo({
      requestLoadDemo: async () => ({ Ok: true, Id: 'op-1' }),
      failedMessage: 'demo failed',
      onFailure: m => failures.push(m),
    }),
  ).toBe(true);
  expect(failures).toEqual([]);

  expect(
    await runSetupLoadDemo({
      requestLoadDemo: async () => ({ Ok: false, ErrorMessage: 'boom' }),
      failedMessage: 'demo failed',
      onFailure: m => failures.push(m),
    }),
  ).toBe(false);
  expect(failures).toEqual(['boom']);

  expect(
    await runSetupLoadDemo({
      requestLoadDemo: async () => {
        throw new Error('network');
      },
      failedMessage: 'demo failed',
      onFailure: m => failures.push(m),
    }),
  ).toBe(false);
  expect(failures).toEqual(['boom', 'network']);
});

test('setup submit then optional load-demo mirrors Setup.vue: demo failure does not undo login', async () => {
  const calls: string[] = [];
  const ok = await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {
      calls.push('setup');
    },
    login: async () => {
      calls.push('login');
    },
    setError: () => {},
  });
  expect(ok).toBe(true);
  expect(setupWantsDemo({ WithDemo: true })).toBe(true);
  const demoOk = await runSetupLoadDemo({
    requestLoadDemo: async () => {
      calls.push('demo');
      return { Ok: false, ErrorMessage: 'demo boom' };
    },
    failedMessage: 'demo failed',
    onFailure: m => {
      calls.push(`fail:${m}`);
    },
  });
  expect(demoOk).toBe(false);
  expect(calls).toEqual(['setup', 'login', 'demo', 'fail:demo boom']);
});

test('runSetupSubmit completes, reports success, then logs in', async () => {
  const calls: string[] = [];
  const ok = await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async r => {
      calls.push(`setup:${r.Admin.Username}`);
    },
    login: async (u, p) => {
      calls.push(`login:${u}:${p}`);
    },
    setError: () => {},
    onSetupSucceeded: () => calls.push('succeeded'),
  });
  expect(ok).toBe(true);
  expect(calls).toEqual(['setup: admin ', 'succeeded', 'login:admin:secret1']);
});

test('runSetupSubmit surfaces CompleteInitialSetup errors without a field', async () => {
  let message = '';
  let setupFailedHook = 0;
  const ok = await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {
      throw new ChoysumError({ domain: 'auth', code: 'VALIDATION_FAILED', message: 'already done' });
    },
    login: async () => {
      throw new Error('should not login');
    },
    setError: m => {
      message = m;
    },
    onFieldError: () => true,
    onSetupFailed: () => {
      setupFailedHook += 1;
    },
  });
  expect(ok).toBe(false);
  expect(message).toBe('already done');
  expect(setupFailedHook).toBe(1);
});

test('runSetupSubmit routes Field errors to onFieldError and skips the header', async () => {
  let message = '';
  const fieldCalls: string[] = [];
  const err = new ChoysumError({ domain: 'auth', code: 'VALIDATION_FAILED', message: 'taken' }).withMetadata({
    Field: 'Admin.Username',
  });
  await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {
      throw err;
    },
    login: async () => {},
    setError: m => {
      message = m;
    },
    onFieldError: (field, msg) => {
      fieldCalls.push(`${field}:${msg}`);
      return true;
    },
  });
  expect(fieldCalls).toEqual(['Admin.Username:taken']);
  expect(message).toBe('');

  // Unhandled fields fall back to the header error.
  await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {
      throw err;
    },
    login: async () => {},
    setError: m => {
      message = m;
    },
    onFieldError: () => false,
  });
  expect(message).toBe('taken');
});

test('runSetupSubmit: login failure after setup only fires the fallback hook', async () => {
  let message = '';
  let loginFailedHook = 0;
  const ok = await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {},
    login: async () => {
      throw new ChoysumError({ domain: 'auth', code: 'INVALID_CREDENTIALS', message: 'bad login' });
    },
    setError: m => {
      message = m;
    },
    onSetupCompleteLoginFailed: () => {
      loginFailedHook += 1;
    },
  });
  expect(ok).toBe(false);
  expect(message).toBe('');
  expect(loginFailedHook).toBe(1);
});

test('runSetupSubmit maps non-ChoysumError setup failures to the fallback message', async () => {
  let message = '';
  const ok = await runSetupSubmit({
    loading: false,
    req: req(),
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {
      throw new Error('network down');
    },
    login: async () => {
      throw new Error('should not login');
    },
    setError: m => {
      message = m;
    },
  });
  expect(ok).toBe(false);
  expect(message).toBe('setup failed');
});

test('runSetupSubmit skips while loading or when credentials are empty', async () => {
  const base = {
    setupFailedMessage: 'setup failed',
    completeInitialSetup: async () => {
      throw new Error('no');
    },
    login: async () => {
      throw new Error('no');
    },
    setError: () => {},
  };
  expect(await runSetupSubmit({ ...base, loading: true, req: req() })).toBe(false);
  expect(await runSetupSubmit({ ...base, loading: false, req: req('  ', 'secret1') })).toBe(false);
  expect(await runSetupSubmit({ ...base, loading: false, req: req('admin', '') })).toBe(false);
});
