// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

(() => {
  'use strict';

  const registry = [];

  function nowMs() {
    return Date.now ? Date.now() : +new Date();
  }

  function stringifySafe(v) {
    try {
      if (typeof v === 'string') return v;
      return JSON.stringify(v);
    } catch {
      try {
        return String(v);
      } catch {
        return '[Unstringifiable]';
      }
    }
  }

  function makeErrorInfo(err) {
    if (!err) return { message: 'Unknown error' };
    if (typeof err === 'string') return { message: err };
    const info = {
      name: err.name,
      message: err.message || String(err),
    };
    if (err.stack) info.stack = String(err.stack);
    return info;
  }

  function deepEqual(a, b) {
    if (a === b) return true;
    if (typeof a !== typeof b) return false;
    if (a && b && typeof a === 'object') {
      if (Array.isArray(a) !== Array.isArray(b)) return false;
      if (Array.isArray(a)) {
        if (a.length !== b.length) return false;
        for (let i = 0; i < a.length; i++) {
          if (!deepEqual(a[i], b[i])) return false;
        }
        return true;
      }
      const aKeys = Object.keys(a).sort();
      const bKeys = Object.keys(b).sort();
      if (aKeys.length !== bKeys.length) return false;
      for (let i = 0; i < aKeys.length; i++) {
        if (aKeys[i] !== bKeys[i]) return false;
        if (!deepEqual(a[aKeys[i]], b[bKeys[i]])) return false;
      }
      return true;
    }
    return false;
  }

  function deepMatchObject(received, expected) {
    if (expected === received) return true;
    if (!expected || typeof expected !== 'object') {
      return deepEqual(received, expected);
    }
    if (!received || typeof received !== 'object') return false;

    if (Array.isArray(expected)) {
      if (!Array.isArray(received)) return false;
      if (received.length !== expected.length) return false;
      for (let i = 0; i < expected.length; i++) {
        if (!deepMatchObject(received[i], expected[i])) return false;
      }
      return true;
    }

    const keys = Object.keys(expected);
    for (let i = 0; i < keys.length; i++) {
      const key = keys[i];
      if (!Object.prototype.hasOwnProperty.call(received, key)) return false;
      if (!deepMatchObject(received[key], expected[key])) return false;
    }
    return true;
  }

  function normalizePropertyPath(path) {
    if (Array.isArray(path)) {
      if (!path.length) throw new Error('toHaveProperty requires a non-empty path');
      return path.slice();
    }
    if (typeof path === 'string') {
      const parts = path.split('.').filter(Boolean);
      if (!parts.length) throw new Error('toHaveProperty requires a non-empty path');
      return parts;
    }
    if (typeof path === 'number') {
      return [path];
    }
    throw new Error(`toHaveProperty requires a string, number, or path array, got ${typeof path}`);
  }

  function getPropertyPathInfo(value, path) {
    const parts = normalizePropertyPath(path);
    let current = value;
    for (let i = 0; i < parts.length; i++) {
      const key = parts[i];
      if ((typeof current !== 'object' && typeof current !== 'function') || current === null) {
        return { exists: false, value: undefined, parts };
      }
      if (!Object.prototype.hasOwnProperty.call(current, key)) {
        return { exists: false, value: undefined, parts };
      }
      current = current[key];
    }
    return { exists: true, value: current, parts };
  }

  function formatPropertyPath(parts) {
    return parts.map(part => String(part)).join('.');
  }

  function Expectation(received, negated) {
    this.received = received;
    this.negated = !!negated;
  }

  Expectation.prototype._assert = function (pass, positiveMessage, negativeMessage) {
    if (this.negated ? pass : !pass) {
      throw new Error(this.negated ? negativeMessage : positiveMessage);
    }
  };

  Object.defineProperty(Expectation.prototype, 'not', {
    get() {
      return new Expectation(this.received, !this.negated);
    },
  });

  function errorMessageOf(err) {
    if (!err) return '';
    if (typeof err === 'string') return err;
    if (typeof err.message === 'string') return err.message;
    return String(err);
  }

  function matchesExpectedError(err, expected) {
    if (expected === undefined || expected === null) {
      return true;
    }

    const msg = errorMessageOf(err);
    if (typeof expected === 'string') {
      return msg.indexOf(expected) >= 0;
    }
    if (expected && typeof expected.test === 'function') {
      return !!expected.test(msg);
    }

    return false;
  }

  Expectation.prototype.toBe = function (expected) {
    const pass = this.received === expected;
    this._assert(
      pass,
      `Expected ${stringifySafe(this.received)} to be ${stringifySafe(expected)}`,
      `Expected ${stringifySafe(this.received)} not to be ${stringifySafe(expected)}`
    );
  };

  Expectation.prototype.toEqual = function (expected) {
    const pass = deepEqual(this.received, expected);
    this._assert(
      pass,
      `Expected ${stringifySafe(this.received)} to equal ${stringifySafe(expected)}`,
      `Expected ${stringifySafe(this.received)} not to equal ${stringifySafe(expected)}`
    );
  };

  Expectation.prototype.toMatchObject = function (expected) {
    if (!expected || typeof expected !== 'object') {
      throw new Error(`toMatchObject requires a non-null object or array, got ${typeof expected}`);
    }

    const pass = deepMatchObject(this.received, expected);
    this._assert(
      pass,
      `Expected ${stringifySafe(this.received)} to match object ${stringifySafe(expected)}`,
      `Expected ${stringifySafe(this.received)} not to match object ${stringifySafe(expected)}`
    );
  };

  Expectation.prototype.toBeTruthy = function () {
    this._assert(!!this.received, `Expected ${stringifySafe(this.received)} to be truthy`, `Expected ${stringifySafe(this.received)} not to be truthy`);
  };

  Expectation.prototype.toBeFalsy = function () {
    this._assert(!this.received, `Expected ${stringifySafe(this.received)} to be falsy`, `Expected ${stringifySafe(this.received)} not to be falsy`);
  };

  Expectation.prototype.toBeUndefined = function () {
    this._assert(
      this.received === undefined,
      `Expected ${stringifySafe(this.received)} to be undefined`,
      `Expected ${stringifySafe(this.received)} not to be undefined`
    );
  };

  Expectation.prototype.toBeDefined = function () {
    this._assert(this.received !== undefined, 'Expected value to be defined', 'Expected value not to be defined');
  };

  Expectation.prototype.toBeNull = function () {
    this._assert(this.received === null, `Expected ${stringifySafe(this.received)} to be null`, `Expected ${stringifySafe(this.received)} not to be null`);
  };

  Expectation.prototype.toMatch = function (expected) {
    if (typeof this.received !== 'string') {
      throw new Error(`toMatch requires a string value, got ${typeof this.received}`);
    }

    let pass = false;

    if (typeof expected === 'string') {
      pass = this.received.indexOf(expected) >= 0;
      this._assert(
        pass,
        `Expected ${stringifySafe(this.received)} to match ${stringifySafe(expected)}`,
        `Expected ${stringifySafe(this.received)} not to match ${stringifySafe(expected)}`
      );
      return;
    }

    if (expected && typeof expected.test === 'function') {
      pass = !!expected.test(this.received);
      this._assert(
        pass,
        `Expected ${stringifySafe(this.received)} to match ${stringifySafe(expected)}`,
        `Expected ${stringifySafe(this.received)} not to match ${stringifySafe(expected)}`
      );
      return;
    }

    throw new Error(`toMatch requires a string or RegExp, got ${typeof expected}`);
  };

  Expectation.prototype.toHaveLength = function (expected) {
    const value = this.received;
    if (!(Array.isArray(value) || typeof value === 'string')) {
      throw new Error(`toHaveLength supports only string or array, got ${typeof value}`);
    }

    if (typeof expected !== 'number' || !Number.isFinite(expected)) {
      throw new Error(`toHaveLength requires a finite number, got ${stringifySafe(expected)}`);
    }
    this._assert(value.length === expected, `Expected length ${value.length} to be ${expected}`, `Expected length ${value.length} not to be ${expected}`);
  };

  Expectation.prototype.toHaveProperty = function (path, expectedValue) {
    const info = getPropertyPathInfo(this.received, path);
    const hasExpectedValue = arguments.length >= 2;
    const label = formatPropertyPath(info.parts);
    const pass = info.exists && (!hasExpectedValue || deepEqual(info.value, expectedValue));

    let positiveMessage = `Expected ${stringifySafe(this.received)} to have property ${stringifySafe(label)}`;
    let negativeMessage = `Expected ${stringifySafe(this.received)} not to have property ${stringifySafe(label)}`;

    if (info.exists && hasExpectedValue && !deepEqual(info.value, expectedValue)) {
      positiveMessage = `Expected property ${stringifySafe(label)} value ${stringifySafe(info.value)} to equal ${stringifySafe(expectedValue)}`;
    }
    if (info.exists && hasExpectedValue) {
      negativeMessage = `Expected property ${stringifySafe(label)} value ${stringifySafe(info.value)} not to equal ${stringifySafe(expectedValue)}`;
    }

    this._assert(pass, positiveMessage, negativeMessage);
  };

  function ensureNumber(received, matcherName) {
    if (typeof received !== 'number' || Number.isNaN(received)) {
      throw new Error(`${matcherName} requires a number value, got ${typeof received}`);
    }
  }

  function ensureNumberExpected(expected, matcherName) {
    if (typeof expected !== 'number' || Number.isNaN(expected)) {
      throw new Error(`${matcherName} requires a number expected value, got ${typeof expected}`);
    }
  }

  Expectation.prototype.toBeGreaterThan = function (expected) {
    ensureNumber(this.received, 'toBeGreaterThan');
    ensureNumberExpected(expected, 'toBeGreaterThan');
    this._assert(
      this.received > expected,
      `Expected ${this.received} to be greater than ${expected}`,
      `Expected ${this.received} not to be greater than ${expected}`
    );
  };

  Expectation.prototype.toBeLessThan = function (expected) {
    ensureNumber(this.received, 'toBeLessThan');
    ensureNumberExpected(expected, 'toBeLessThan');
    this._assert(
      this.received < expected,
      `Expected ${this.received} to be less than ${expected}`,
      `Expected ${this.received} not to be less than ${expected}`
    );
  };

  Expectation.prototype.toBeGreaterThanOrEqual = function (expected) {
    ensureNumber(this.received, 'toBeGreaterThanOrEqual');
    ensureNumberExpected(expected, 'toBeGreaterThanOrEqual');
    this._assert(
      this.received >= expected,
      `Expected ${this.received} to be greater than or equal to ${expected}`,
      `Expected ${this.received} not to be greater than or equal to ${expected}`
    );
  };

  Expectation.prototype.toBeLessThanOrEqual = function (expected) {
    ensureNumber(this.received, 'toBeLessThanOrEqual');
    ensureNumberExpected(expected, 'toBeLessThanOrEqual');
    this._assert(
      this.received <= expected,
      `Expected ${this.received} to be less than or equal to ${expected}`,
      `Expected ${this.received} not to be less than or equal to ${expected}`
    );
  };

  Expectation.prototype.toContain = function (expected) {
    const value = this.received;
    let pass = false;
    if (typeof value === 'string') {
      pass = value.indexOf(String(expected)) >= 0;
      this._assert(
        pass,
        `Expected ${stringifySafe(value)} to contain ${stringifySafe(expected)}`,
        `Expected ${stringifySafe(value)} not to contain ${stringifySafe(expected)}`
      );
      return;
    }

    if (Array.isArray(value)) {
      for (let i = 0; i < value.length; i++) {
        if (value[i] === expected) {
          pass = true;
          break;
        }
      }
      this._assert(
        pass,
        `Expected ${stringifySafe(value)} to contain ${stringifySafe(expected)}`,
        `Expected ${stringifySafe(value)} not to contain ${stringifySafe(expected)}`
      );
      return;
    }

    throw new Error(`toContain supports only string or array, got ${typeof value}`);
  };

  Expectation.prototype.toThrow = function (expected) {
    if (typeof this.received !== 'function') {
      throw new Error(`toThrow requires a function, got ${typeof this.received}`);
    }

    let thrown = false;
    let err = null;
    try {
      this.received();
    } catch (e) {
      thrown = true;
      err = e;
    }

    const pass = thrown && matchesExpectedError(err, expected);
    const actualMessage = stringifySafe(errorMessageOf(err));
    const positiveMessage = !thrown ? 'Expected function to throw' : `Thrown error message ${actualMessage} does not match ${stringifySafe(expected)}`;
    const negativeMessage =
      expected === undefined || expected === null
        ? 'Expected function not to throw'
        : `Expected thrown error message ${actualMessage} not to match ${stringifySafe(expected)}`;
    this._assert(pass, positiveMessage, negativeMessage);
  };

  function expect(received) {
    return new Expectation(received, false);
  }

  async function expectRejects(received, expected) {
    let promise;
    if (typeof received === 'function') {
      promise = received();
    } else {
      promise = received;
    }

    if (!promise || typeof promise.then !== 'function') {
      throw new Error(`expectRejects requires a Promise or async function, got ${typeof promise}`);
    }

    try {
      await promise;
      throw new Error('Expected promise to reject');
    } catch (err) {
      if (errorMessageOf(err) === 'Expected promise to reject') {
        throw err;
      }
      if (!matchesExpectedError(err, expected)) {
        throw new Error(`Rejected error message ${stringifySafe(errorMessageOf(err))} does not match ${stringifySafe(expected)}`);
      }
    }
  }

  function test(name, fn) {
    if (typeof name !== 'string' || name.trim() === '') {
      throw new Error('test(name, fn): name must be a non-empty string');
    }
    if (typeof fn !== 'function') {
      throw new Error('test(name, fn): fn must be a function');
    }
    registry.push({ name, fn });
  }

  function compilePattern(pattern) {
    if (!pattern) return null;
    try {
      return new RegExp(pattern);
    } catch (e) {
      throw new Error(`Invalid --pattern regex: ${pattern}`);
    }
  }

  function readDefaultUnitIdentity() {
    try {
      const jsCtx = globalThis.$choysum && globalThis.$choysum.request && globalThis.$choysum.request.context;
      if (!jsCtx || typeof jsCtx !== 'object') return null;
      const userId = jsCtx.identity && typeof jsCtx.identity.userId === 'string' ? String(jsCtx.identity.userId).trim() : '';
      const companyId =
        jsCtx.ctx && typeof jsCtx.ctx.activeCompanyId === 'string' ? String(jsCtx.ctx.activeCompanyId).trim() : '';
      if (!userId || !companyId) return null;
      return { userId, companyId };
    } catch {
      return null;
    }
  }

  function applyDefaultUnitIdentity(defaults) {
    const root = (globalThis.$choysum = globalThis.$choysum || {});
    if (!root.request) root.request = {};
    // Always rebuild identity/ctx/req so prior allowlists/lang/depth cannot leak —
    // including auth-free suites where defaults is null.
    const jsCtx = (root.request.context = {});
    jsCtx.identity = {};
    jsCtx.ctx = {};
    jsCtx.req = { depth: 0 };
    if (!defaults) return;
    jsCtx.identity.userId = defaults.userId;
    jsCtx.ctx.activeCompanyId = defaults.companyId;
    jsCtx.ctx.enabledCompanyIds = [defaults.companyId];
  }

  async function runAll(options) {
    const opts = options || {};
    const re = compilePattern(opts.pattern);
    const failFast = !!opts.failFast;

    const selected = re ? registry.filter(t => re.test(t.name)) : registry.slice();
    // Capture once: Go injects bootstrap admin when auth is installed; empty means no-op.
    const defaultIdentity = readDefaultUnitIdentity();

    const cases = [];
    let passed = 0;
    let failed = 0;

    for (let i = 0; i < selected.length; i++) {
      const t = selected[i];
      applyDefaultUnitIdentity(defaultIdentity);
      const start = nowMs();
      let ok = true;
      let errInfo = null;
      try {
        const r = t.fn();
        if (r && typeof r.then === 'function') {
          await r;
        }
      } catch (err) {
        ok = false;
        errInfo = makeErrorInfo(err);
      }
      const durationMs = Math.max(0, nowMs() - start);

      if (ok) {
        passed++;
      } else {
        failed++;
      }

      cases.push({ name: t.name, ok, durationMs, error: errInfo });

      if (!ok && failFast) {
        break;
      }
    }

    const total = selected.length;
    const report = {
      total,
      passed,
      failed,
      cases,
      coverageJSON: typeof globalThis !== 'undefined' && globalThis.__coverage__ ? JSON.stringify(globalThis.__coverage__) : null,
    };

    return report;
  }

  globalThis.test = test;
  globalThis.expect = expect;
  globalThis.expectRejects = expectRejects;
  globalThis.__choysum_test_run__ = runAll;
})();
