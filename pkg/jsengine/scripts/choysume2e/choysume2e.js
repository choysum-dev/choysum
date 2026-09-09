// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * @choysum/e2e — Playwright-subset facade for QuickJS e2e host.
 * Host globals: __choysum_e2e_host__, __choysum_e2e_runtime__ (set by pagehost.Install).
 */

function getHost() {
  const h = globalThis.__choysum_e2e_host__;
  if (!h) {
    throw new Error('@choysum/e2e: __choysum_e2e_host__ is not installed');
  }
  return h;
}

function getRuntime() {
  const r = globalThis.__choysum_e2e_runtime__;
  if (r == null) {
    throw new Error('@choysum/e2e: __choysum_e2e_runtime__ is not installed');
  }
  return r;
}

function sleep(ms) {
  // EvalAwait does not pump QuickJS os.setTimeout; use Go-scheduled host.delay.
  const n = typeof ms === 'number' && ms > 0 ? ms : 0;
  return getHost().delay(n);
}

function decodeBase64ToUint8Array(b64) {
  const s = String(b64 || '');
  if (!s) return new Uint8Array(0);
  if (typeof atob === 'function') {
    const bin = atob(s);
    const out = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
    return out;
  }
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
  const clean = s.replace(/[^A-Za-z0-9+/]/g, '');
  const len = clean.length;
  const out = [];
  for (let i = 0; i < len; i += 4) {
    const a = chars.indexOf(clean[i]);
    const b = chars.indexOf(clean[i + 1]);
    const c = chars.indexOf(clean[i + 2]);
    const d = chars.indexOf(clean[i + 3]);
    out.push((a << 2) | (b >> 4));
    if (clean[i + 2] !== '=' && c >= 0) out.push(((b & 15) << 4) | (c >> 2));
    if (clean[i + 3] !== '=' && d >= 0) out.push(((c & 3) << 6) | d);
  }
  return new Uint8Array(out);
}

function wrapResponse(rawJSON) {
  const raw = typeof rawJSON === 'string' ? JSON.parse(rawJSON) : rawJSON;
  const headersObj = raw.headers || {};
  return {
    status() {
      return Number(raw.status) || 0;
    },
    headers() {
      return headersObj;
    },
    url() {
      return String(raw.url || '');
    },
    async body() {
      return decodeBase64ToUint8Array(raw.bodyBase64 || '');
    },
  };
}

function escapeRe(s) {
  return String(s).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function placeholderNeedle(reOrString) {
  if (reOrString && typeof reOrString === 'object' && reOrString.source != null) {
    return String(reOrString.source).replace(/^\^/, '').replace(/\$$/, '').replace(/\\/g, '');
  }
  return String(reOrString);
}

function makeLocator(kind, value) {
  const loc = {
    __choysum_e2e_locator__: true,
    kind,
    value,
    _css: '',
    first() {
      return makeLocator(kind, value);
    },
    async click() {
      const sel = await ensureCSS(loc);
      await getHost().click(sel);
    },
    async fill(text) {
      const sel = await ensureCSS(loc);
      await getHost().fill(sel, String(text ?? ''));
    },
  };
  return loc;
}

async function ensureCSS(loc) {
  if (loc._css) return loc._css;
  loc._css = await resolveToCSS(loc.kind, loc.value);
  return loc._css;
}

async function resolveToCSS(kind, value) {
  if (kind === 'css') return String(value);
  if (kind === 'placeholder') {
    const needle = placeholderNeedle(value);
    const stamp = 'e2e' + String(Date.now()) + Math.random().toString(16).slice(2);
    const raw = await getHost().evaluate(`(() => {
      const needle = ${JSON.stringify(needle)}.toLowerCase();
      const stamp = ${JSON.stringify(stamp)};
      const nodes = Array.from(document.querySelectorAll('input, textarea'));
      for (const el of nodes) {
        const attrs = [
          el.getAttribute('placeholder'),
          el.getAttribute('aria-label'),
          el.getAttribute('name'),
          el.getAttribute('autocomplete'),
          el.id,
        ];
        for (const a of attrs) {
          if (a && String(a).toLowerCase().includes(needle)) {
            el.setAttribute('data-choysum-e2e-id', stamp);
            return stamp;
          }
        }
      }
      const body = document.body ? String(document.body.innerText || '').slice(0, 240) : '';
      throw new Error('getByPlaceholder: no match for ' + needle + ' url=' + location.href + ' body=' + JSON.stringify(body));
    })()`);
    const id = JSON.parse(raw);
    return '[data-choysum-e2e-id="' + id + '"]';
  }
  if (kind === 'text') {
    const needle = String(value);
    const stamp = 'e2e' + String(Date.now()) + Math.random().toString(16).slice(2);
    const raw = await getHost().evaluate(`(() => {
      const needle = ${JSON.stringify(needle)};
      const stamp = ${JSON.stringify(stamp)};
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT);
      let n = walker.currentNode;
      while (n) {
        if (n.childElementCount === 0) {
          const t = (n.textContent || '').trim();
          if (t && t.includes(needle)) {
            n.setAttribute('data-choysum-e2e-id', stamp);
            return stamp;
          }
        }
        n = walker.nextNode();
      }
      throw new Error('getByText: no match for ' + needle + ' url=' + location.href);
    })()`);
    const id = JSON.parse(raw);
    return '[data-choysum-e2e-id="' + id + '"]';
  }
  throw new Error('unknown locator kind: ' + kind);
}

async function countLocator(kind, value) {
  if (kind === 'css') {
    return await getHost().count(String(value));
  }
  if (kind === 'placeholder') {
    const needle = placeholderNeedle(value);
    const raw = await getHost().evaluate(`(() => {
      const needle = ${JSON.stringify(needle)}.toLowerCase();
      let c = 0;
      for (const el of document.querySelectorAll('input[placeholder],textarea[placeholder]')) {
        const p = String(el.getAttribute('placeholder') || '').toLowerCase();
        if (p.includes(needle)) c++;
      }
      return c;
    })()`);
    return JSON.parse(raw);
  }
  if (kind === 'text') {
    const needle = String(value);
    const raw = await getHost().evaluate(`(() => {
      const needle = ${JSON.stringify(needle)};
      let c = 0;
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_ELEMENT);
      let n = walker.currentNode;
      while (n) {
        if (n.childElementCount === 0) {
          const t = (n.textContent || '').trim();
          if (t && t.includes(needle)) c++;
        }
        n = walker.nextNode();
      }
      return c;
    })()`);
    return JSON.parse(raw);
  }
  return 0;
}

const page = {
  __choysum_e2e_page__: true,
  async goto(url, opts) {
    const waitUntil = opts && opts.waitUntil ? opts.waitUntil : 'load';
    await getHost().goto(String(url), waitUntil);
  },
  locator(sel) {
    return makeLocator('css', String(sel));
  },
  getByPlaceholder(reOrString) {
    return makeLocator('placeholder', reOrString);
  },
  getByText(text) {
    return makeLocator('text', text);
  },
  async click(sel) {
    await getHost().click(String(sel));
  },
  async fill(sel, value) {
    await getHost().fill(String(sel), String(value ?? ''));
  },
  async evaluate(fnOrSource, arg) {
    let src;
    if (typeof fnOrSource === 'function') {
      src = `(${fnOrSource.toString()})(${arg === undefined ? '' : JSON.stringify(arg)})`;
    } else {
      src = String(fnOrSource);
    }
    const raw = await getHost().evaluate(src);
    try {
      return JSON.parse(raw);
    } catch {
      return raw;
    }
  },
  async waitForFunction(fnOrSource, arg, opts) {
    let timeout = 30000;
    let realArg = arg;
    let realOpts = opts;
    // Playwright signature: (fn, arg?, options?) — login passes (fn, undefined, {timeout}).
    if (arg && typeof arg === 'object' && !opts && (arg.timeout != null || arg.polling != null)) {
      realOpts = arg;
      realArg = undefined;
    }
    if (realOpts && typeof realOpts.timeout === 'number') timeout = realOpts.timeout;
    let src;
    if (typeof fnOrSource === 'function') {
      src = `(${fnOrSource.toString()})(${realArg === undefined ? '' : JSON.stringify(realArg)})`;
    } else {
      src = String(fnOrSource);
    }
    await getHost().waitForFunction(src, timeout);
  },
  async waitForResponse(match, opts) {
    const timeout = opts && typeof opts.timeout === 'number' ? opts.timeout : 30000;
    let matchObj = match;
    if (typeof match === 'string') {
      matchObj = { urlIncludes: match };
    } else if (typeof match === 'function') {
      throw new Error(
        '@choysum/e2e: waitForResponse(predicate) is not supported; pass {urlIncludes, method, contentTypePrefix}'
      );
    }
    const raw = await getHost().waitForResponse(JSON.stringify(matchObj || {}), timeout);
    return wrapResponse(raw);
  },
  async url() {
    return await getHost().url();
  },
  async screenshot(opts) {
    const path = opts && opts.path ? opts.path : String(opts || '');
    await getHost().screenshot(path);
  },
};

async function poll(timeoutMs, fn) {
  const deadline = Date.now() + (typeof timeoutMs === 'number' ? timeoutMs : 30000);
  let lastErr;
  while (Date.now() < deadline) {
    try {
      if (await fn()) return;
    } catch (e) {
      lastErr = e;
    }
    await sleep(50);
  }
  if (lastErr) throw lastErr;
  throw new Error('expect polling timeout');
}

function e2eExpect(target) {
  const api = {
    async toBeVisible(opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        throw new Error('toBeVisible: expected locator');
      }
      await poll(timeout, async () => {
        try {
          const sel = await ensureCSS(target);
          return await getHost().isVisible(sel);
        } catch {
          target._css = '';
          return false;
        }
      });
    },
    async toBeEnabled(opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        throw new Error('toBeEnabled: expected locator');
      }
      await poll(timeout, async () => {
        try {
          const sel = await ensureCSS(target);
          return await getHost().isEnabled(sel);
        } catch {
          target._css = '';
          return false;
        }
      });
    },
    async toHaveCount(n, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        throw new Error('toHaveCount: expected locator');
      }
      await poll(timeout, async () => {
        target._css = '';
        const count = await countLocator(target.kind, target.value);
        return count === n;
      });
    },
    async toHaveURL(reOrString, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_page__) {
        throw new Error('toHaveURL: expected page');
      }
      const re =
        reOrString && typeof reOrString === 'object' && typeof reOrString.test === 'function'
          ? reOrString
          : new RegExp(String(reOrString));
      await poll(timeout, async () => {
        const href = await getHost().url();
        return re.test(String(href));
      });
    },
  };

  if (target && (target.__choysum_e2e_locator__ || target.__choysum_e2e_page__)) {
    return api;
  }
  if (typeof globalThis.expect === 'function') {
    return globalThis.expect(target);
  }
  throw new Error('@choysum/e2e: expect target not supported');
}

function randomUUID() {
  if (typeof globalThis.crypto !== 'undefined' && typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }
  let d = Date.now();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = (d + Math.random() * 16) % 16 | 0;
    d = Math.floor(d / 16);
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}

const test = globalThis.test;
const expect = e2eExpect;
const runtime = new Proxy(
  {},
  {
    get(_t, prop) {
      const r = getRuntime();
      return r != null ? r[prop] : undefined;
    },
  }
);

if (!globalThis.__choysum_e2e_hooks_installed__ && typeof globalThis.beforeEach === 'function') {
  globalThis.__choysum_e2e_hooks_installed__ = true;
  globalThis.beforeEach(async () => {
    await getHost().newPage();
  });
  if (typeof globalThis.afterEach === 'function') {
    globalThis.afterEach(async () => {
      try {
        await getHost().closePage();
      } catch {
        // ignore
      }
    });
  }
}

export { test, expect, page, runtime, randomUUID };
