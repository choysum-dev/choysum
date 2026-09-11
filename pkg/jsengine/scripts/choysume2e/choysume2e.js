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
  // Prefer a pure JS decoder: QuickJS atob (when present) is not reliable for binary.
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
  const clean = s.replace(/[^A-Za-z0-9+/=]/g, '');
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

function encodeUint8ArrayToBase64(bytes) {
  const u8 = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
  let out = '';
  for (let i = 0; i < u8.length; i += 3) {
    const a = u8[i];
    const b = i + 1 < u8.length ? u8[i + 1] : 0;
    const c = i + 2 < u8.length ? u8[i + 2] : 0;
    out += chars[a >> 2];
    out += chars[((a & 3) << 4) | (b >> 4)];
    out += i + 1 < u8.length ? chars[((b & 15) << 2) | (c >> 6)] : '=';
    out += i + 2 < u8.length ? chars[c & 63] : '=';
  }
  return out;
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

function placeholderNeedle(reOrString) {
  if (reOrString && typeof reOrString === 'object' && reOrString.source != null) {
    return { kind: 're', source: String(reOrString.source), flags: String(reOrString.flags || '') };
  }
  return { kind: 'str', value: String(reOrString) };
}

function serializeNameMatch(name, exact) {
  if (name && typeof name === 'object' && name.source != null) {
    return { kind: 're', source: String(name.source), flags: String(name.flags || '') };
  }
  if (name == null || name === '') return null;
  return { kind: 'str', value: String(name), exact: !!exact };
}

function newStamp() {
  return 'e2e' + String(Date.now()) + Math.random().toString(16).slice(2);
}

function makeLocator(kind, value, opts) {
  const options = opts && typeof opts === 'object' ? opts : {};
  const loc = {
    __choysum_e2e_locator__: true,
    kind,
    value,
    index: typeof options.index === 'number' ? options.index : null,
    hasText: options.hasText != null ? options.hasText : null,
    nameMatch: options.nameMatch != null ? options.nameMatch : null,
    parent: options.parent || null,
    _css: '',
    first() {
      return makeLocator(kind, value, { ...options, index: 0 });
    },
    last() {
      return makeLocator(kind, value, { ...options, index: -1 });
    },
    nth(i) {
      return makeLocator(kind, value, { ...options, index: Number(i) || 0 });
    },
    locator(sel, childOpts) {
      return makeLocator('css', String(sel), {
        hasText: childOpts && childOpts.hasText != null ? childOpts.hasText : null,
        parent: loc,
      });
    },
    getByRole(role, roleOpts) {
      const nameMatch = serializeNameMatch(roleOpts && roleOpts.name, roleOpts && roleOpts.exact);
      return makeLocator('role', String(role), { nameMatch, parent: loc });
    },
    async click() {
      const sel = await ensureCSS(loc);
      await getHost().click(sel);
    },
    async fill(text) {
      const sel = await ensureCSS(loc);
      await getHost().fill(sel, String(text ?? ''));
    },
    async count() {
      return await countLocator(loc);
    },
    async textContent() {
      const sel = await ensureCSS(loc);
      const raw = await getHost().evaluate(`(() => {
        const el = document.querySelector(${JSON.stringify(sel)});
        return el ? el.textContent : null;
      })()`);
      return JSON.parse(raw);
    },
    async innerText() {
      const sel = await ensureCSS(loc);
      const raw = await getHost().evaluate(`(() => {
        const el = document.querySelector(${JSON.stringify(sel)});
        if (!el) return null;
        return typeof el.innerText === 'string' ? el.innerText : el.textContent;
      })()`);
      return JSON.parse(raw);
    },
    async getAttribute(name) {
      const sel = await ensureCSS(loc);
      const raw = await getHost().evaluate(`(() => {
        const el = document.querySelector(${JSON.stringify(sel)});
        if (!el) return null;
        return el.getAttribute(${JSON.stringify(String(name))});
      })()`);
      return JSON.parse(raw);
    },
    async isEnabled() {
      const sel = await ensureCSS(loc);
      return await getHost().isEnabled(sel);
    },
    async isVisible() {
      try {
        const sel = await ensureCSS(loc);
        return await getHost().isVisible(sel);
      } catch {
        return false;
      }
    },
    async waitFor(opts) {
      const timeout = opts && typeof opts.timeout === 'number' ? opts.timeout : 30000;
      const state = opts && opts.state ? String(opts.state) : 'visible';
      await poll(timeout, async () => {
        if (state === 'attached') {
          try {
            await ensureCSS(loc);
            return (await countLocator(loc)) > 0;
          } catch {
            return false;
          }
        }
        if (state === 'detached' || state === 'hidden') {
          try {
            if ((await countLocator(loc)) === 0) return true;
            if (state === 'detached') return false;
            return !(await loc.isVisible());
          } catch {
            return true;
          }
        }
        return await loc.isVisible();
      });
    },
    async press(key) {
      await loc.waitFor({ state: 'visible' });
      const sel = await ensureCSS(loc);
      const keyName = String(key || '');
      await getHost().evaluate(`(() => {
        const el = document.querySelector(${JSON.stringify(sel)});
        if (!el) throw new Error('press: element not found');
        if (typeof el.focus === 'function') el.focus();
        const key = ${JSON.stringify(keyName)};
        const keyCode = key === 'Enter' ? 13 : key === 'Escape' ? 27 : 0;
        const opts = { key, code: key, keyCode, which: keyCode, bubbles: true, cancelable: true };
        el.dispatchEvent(new KeyboardEvent('keydown', opts));
        el.dispatchEvent(new KeyboardEvent('keypress', opts));
        el.dispatchEvent(new KeyboardEvent('keyup', opts));
        return true;
      })()`);
    },
    async allTextContents() {
      const matches = await collectMatchedElements(loc);
      return matches.map(t => (t == null ? '' : String(t)));
    },
  };
  return loc;
}

async function parentScopeSelector(loc) {
  if (!loc || !loc.parent) return '';
  return await ensureCSS(loc.parent);
}

async function ensureCSS(loc) {
  // Dynamic locators re-resolve each action in case the DOM re-rendered.
  if (loc.kind === 'css' && loc.index == null && loc.hasText == null && !loc.parent) {
    return String(loc.value);
  }
  return await resolveToCSS(loc);
}

async function resolveToCSS(loc) {
  const kind = loc.kind;
  const value = loc.value;
  const index = loc.index == null ? 0 : loc.index;
  const stamp = newStamp();
  const parentSel = await parentScopeSelector(loc);

  if (kind === 'css') {
    const hasText = loc.hasText;
    const hasTextJSON =
      hasText && typeof hasText === 'object' && hasText.source != null
        ? JSON.stringify({ kind: 're', source: String(hasText.source), flags: String(hasText.flags || '') })
        : hasText != null
          ? JSON.stringify({ kind: 'str', value: String(hasText) })
          : 'null';
    const raw = await getHost().evaluate(`(() => {
      const rootSel = ${JSON.stringify(parentSel)};
      const css = ${JSON.stringify(String(value))};
      const stamp = ${JSON.stringify(stamp)};
      const index = ${JSON.stringify(index)};
      const hasText = ${hasTextJSON};
      const root = rootSel ? document.querySelector(rootSel) : document;
      if (!root) throw new Error('locator: parent not found ' + rootSel);
      let nodes = Array.from(root.querySelectorAll(css));
      if (hasText) {
        const matchText = (t) => {
          const text = String(t || '');
          if (hasText.kind === 're') {
            try { return new RegExp(hasText.source, hasText.flags || '').test(text); }
            catch { return false; }
          }
          return text.includes(String(hasText.value || ''));
        };
        nodes = nodes.filter(el => matchText(el.textContent || ''));
      }
      const resolved = index < 0 ? nodes.length + index : index;
      if (resolved < 0 || resolved >= nodes.length) {
        throw new Error('locator: css index ' + index + ' out of range (' + nodes.length + ') for ' + css);
      }
      nodes[resolved].setAttribute('data-choysum-e2e-id', stamp);
      return stamp;
    })()`);
    return '[data-choysum-e2e-id="' + JSON.parse(raw) + '"]';
  }

  if (kind === 'testid') {
    const css = '[data-testid=' + JSON.stringify(String(value)) + ']';
    return await resolveToCSS({
      kind: 'css',
      value: css,
      index: loc.index,
      hasText: loc.hasText != null ? loc.hasText : null,
      parent: loc.parent,
    });
  }

  if (kind === 'placeholder') {
    const match = placeholderNeedle(value);
    const matchJSON = JSON.stringify(match);
    const raw = await getHost().evaluate(`(() => {
      const match = ${matchJSON};
      const stamp = ${JSON.stringify(stamp)};
      const index = ${JSON.stringify(index)};
      const rootSel = ${JSON.stringify(parentSel)};
      const root = rootSel ? document.querySelector(rootSel) : document;
      if (!root) throw new Error('getByPlaceholder: parent not found');
      const attrOk = (el) => {
        const attrs = [
          el.getAttribute('placeholder'),
          el.getAttribute('aria-label'),
          el.getAttribute('name'),
          el.getAttribute('autocomplete'),
          el.id,
        ].filter(Boolean).map(a => String(a));
        if (match.kind === 're') {
          let re;
          try { re = new RegExp(match.source, match.flags || ''); }
          catch { return false; }
          return attrs.some(a => re.test(a));
        }
        const needle = String(match.value || '').toLowerCase();
        return attrs.some(a => a.toLowerCase().includes(needle));
      };
      const nodes = [];
      for (const el of root.querySelectorAll('input, textarea')) {
        if (attrOk(el)) nodes.push(el);
      }
      const resolved = index < 0 ? nodes.length + index : index;
      if (resolved < 0 || resolved >= nodes.length) {
        throw new Error('getByPlaceholder: no match for ' + JSON.stringify(match) + ' index=' + index);
      }
      nodes[resolved].setAttribute('data-choysum-e2e-id', stamp);
      return stamp;
    })()`);
    return '[data-choysum-e2e-id="' + JSON.parse(raw) + '"]';
  }

  if (kind === 'text') {
    const needle = String(value);
    const raw = await getHost().evaluate(`(() => {
      const needle = ${JSON.stringify(needle)};
      const stamp = ${JSON.stringify(stamp)};
      const index = ${JSON.stringify(index)};
      const rootSel = ${JSON.stringify(parentSel)};
      const root = rootSel ? document.querySelector(rootSel) : document.body;
      if (!root) throw new Error('getByText: no root');
      const matches = [];
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT);
      let n = walker.currentNode;
      while (n) {
        const hasDirectText = Array.from(n.childNodes).some(
          c => c.nodeType === 3 && (c.nodeValue || '').trim().includes(needle)
        );
        if (n.childElementCount === 0 || hasDirectText) {
          const t = (n.textContent || '').trim();
          if (t && t.includes(needle)) matches.push(n);
        }
        n = walker.nextNode();
      }
      const resolved = index < 0 ? matches.length + index : index;
      if (resolved < 0 || resolved >= matches.length) {
        throw new Error('getByText: no match for ' + needle + ' index=' + index);
      }
      matches[resolved].setAttribute('data-choysum-e2e-id', stamp);
      return stamp;
    })()`);
    return '[data-choysum-e2e-id="' + JSON.parse(raw) + '"]';
  }

  if (kind === 'role') {
    const role = String(value);
    const nameMatch = loc.nameMatch;
    const nameJSON = nameMatch ? JSON.stringify(nameMatch) : 'null';
    const raw = await getHost().evaluate(`(() => {
      const role = ${JSON.stringify(role)};
      const stamp = ${JSON.stringify(stamp)};
      const index = ${JSON.stringify(index)};
      const nameMatch = ${nameJSON};
      const rootSel = ${JSON.stringify(parentSel)};
      const root = rootSel ? document.querySelector(rootSel) : document;
      if (!root) throw new Error('getByRole: parent not found');

      const roleSelectors = {
        button: 'button, [role="button"], input[type="button"], input[type="submit"], input[type="reset"]',
        option: '[role="option"], option',
        dialog: '[role="dialog"], dialog, .el-dialog',
        menuitem: '[role="menuitem"]',
      };
      const sel = roleSelectors[role] || ('[role="' + String(role).replace(/"/g, '') + '"]');
      if (!sel || sel === '[role=""]') throw new Error('getByRole: unsupported role ' + role);

      const isVisibleEl = (el) => {
        const style = window.getComputedStyle(el);
        if (!style || style.visibility === 'hidden' || style.display === 'none' || parseFloat(style.opacity) === 0) return false;
        const r = el.getBoundingClientRect();
        return r.width > 0 && r.height > 0;
      };

      const accessibleName = (el) => {
        const labelled = el.getAttribute('aria-label');
        if (labelled) return String(labelled).trim();
        const labelledBy = el.getAttribute('aria-labelledby');
        if (labelledBy) {
          const parts = String(labelledBy).split(/\\s+/).map(id => {
            const n = document.getElementById(id);
            return n ? (n.textContent || '').trim() : '';
          }).filter(Boolean);
          if (parts.length) return parts.join(' ');
        }
        if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
          const v = el.getAttribute('value') || el.value || '';
          if (v) return String(v).trim();
        }
        return String(el.textContent || '').replace(/\\s+/g, ' ').trim();
      };

      const nameOk = (el) => {
        if (!nameMatch) return true;
        const name = accessibleName(el);
        if (nameMatch.kind === 're') {
          try { return new RegExp(nameMatch.source, nameMatch.flags || '').test(name); }
          catch { return false; }
        }
        const want = String(nameMatch.value || '');
        if (nameMatch.exact) return name === want;
        return name.includes(want);
      };

      let nodes = Array.from(root.querySelectorAll(sel)).filter(nameOk);
      // Element Plus: role=dialog may sit on the overlay wrapper; map to .el-dialog.
      if (role === 'dialog') {
        nodes = nodes.map((el) => {
          if (el.classList && el.classList.contains('el-dialog')) return el;
          const inner = el.querySelector ? el.querySelector('.el-dialog') : null;
          return inner || el;
        });
        // Wrapper + .el-dialog both match the selector; mapping collapses them to one node.
        nodes = nodes.filter((el, i) => nodes.indexOf(el) === i);
      }
      // Prefer visible candidates before preferring .el-dialog content panels.
      const visible = nodes.filter(isVisibleEl);
      if (visible.length) nodes = visible;
      if (role === 'dialog') {
        const content = nodes.filter((el) => el.classList && el.classList.contains('el-dialog'));
        if (content.length) nodes = content;
      }
      const resolved = index < 0 ? nodes.length + index : index;
      if (resolved < 0 || resolved >= nodes.length) {
        throw new Error('getByRole: no match for role=' + role + ' index=' + index + ' count=' + nodes.length);
      }
      nodes[resolved].setAttribute('data-choysum-e2e-id', stamp);
      return stamp;
    })()`);
    return '[data-choysum-e2e-id="' + JSON.parse(raw) + '"]';
  }

  throw new Error('unknown locator kind: ' + kind);
}

async function countLocator(loc) {
  if (loc.kind === 'css' && loc.hasText == null && !loc.parent && loc.index == null) {
    return await getHost().count(String(loc.value));
  }
  const texts = await collectMatchedElements(loc);
  if (loc.index != null) {
    const resolved = loc.index < 0 ? texts.length + loc.index : loc.index;
    return resolved >= 0 && resolved < texts.length ? 1 : 0;
  }
  return texts.length;
}

async function collectMatchedElements(loc) {
  const kind = loc.kind;
  const value = loc.value;
  const parentSel = await parentScopeSelector(loc);

  if (kind === 'css') {
    const hasText = loc.hasText;
    const hasTextJSON =
      hasText && typeof hasText === 'object' && hasText.source != null
        ? JSON.stringify({ kind: 're', source: String(hasText.source), flags: String(hasText.flags || '') })
        : hasText != null
          ? JSON.stringify({ kind: 'str', value: String(hasText) })
          : 'null';
    const raw = await getHost().evaluate(`(() => {
      const rootSel = ${JSON.stringify(parentSel)};
      const css = ${JSON.stringify(String(value))};
      const hasText = ${hasTextJSON};
      const root = rootSel ? document.querySelector(rootSel) : document;
      if (!root) return [];
      let nodes = Array.from(root.querySelectorAll(css));
      if (hasText) {
        const matchText = (t) => {
          const text = String(t || '');
          if (hasText.kind === 're') {
            try { return new RegExp(hasText.source, hasText.flags || '').test(text); }
            catch { return false; }
          }
          return text.includes(String(hasText.value || ''));
        };
        nodes = nodes.filter(el => matchText(el.textContent || ''));
      }
      return nodes.map(el => el.textContent);
    })()`);
    return JSON.parse(raw);
  }

  if (kind === 'testid') {
    return collectMatchedElements({
      kind: 'css',
      value: '[data-testid=' + JSON.stringify(String(value)) + ']',
      hasText: loc.hasText != null ? loc.hasText : null,
      parent: loc.parent,
    });
  }

  if (kind === 'placeholder') {
    const match = placeholderNeedle(value);
    const matchJSON = JSON.stringify(match);
    const raw = await getHost().evaluate(`(() => {
      const match = ${matchJSON};
      const rootSel = ${JSON.stringify(parentSel)};
      const root = rootSel ? document.querySelector(rootSel) : document;
      if (!root) return [];
      const attrOk = (el) => {
        const attrs = [
          el.getAttribute('placeholder'),
          el.getAttribute('aria-label'),
          el.getAttribute('name'),
          el.getAttribute('autocomplete'),
          el.id,
        ].filter(Boolean).map(a => String(a));
        if (match.kind === 're') {
          let re;
          try { re = new RegExp(match.source, match.flags || ''); }
          catch { return false; }
          return attrs.some(a => re.test(a));
        }
        const needle = String(match.value || '').toLowerCase();
        return attrs.some(a => a.toLowerCase().includes(needle));
      };
      const out = [];
      for (const el of root.querySelectorAll('input, textarea')) {
        if (attrOk(el)) out.push(el.value || '');
      }
      return out;
    })()`);
    return JSON.parse(raw);
  }

  if (kind === 'text') {
    const needle = String(value);
    const raw = await getHost().evaluate(`(() => {
      const needle = ${JSON.stringify(needle)};
      const rootSel = ${JSON.stringify(parentSel)};
      const root = rootSel ? document.querySelector(rootSel) : document.body;
      if (!root) return [];
      const out = [];
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT);
      let n = walker.currentNode;
      while (n) {
        const hasDirectText = Array.from(n.childNodes).some(
          c => c.nodeType === 3 && (c.nodeValue || '').trim().includes(needle)
        );
        if (n.childElementCount === 0 || hasDirectText) {
          const t = (n.textContent || '').trim();
          if (t && t.includes(needle)) out.push(t);
        }
        n = walker.nextNode();
      }
      return out;
    })()`);
    return JSON.parse(raw);
  }

  if (kind === 'role') {
    const role = String(value);
    const nameMatch = loc.nameMatch;
    const nameJSON = nameMatch ? JSON.stringify(nameMatch) : 'null';
    const raw = await getHost().evaluate(`(() => {
      const role = ${JSON.stringify(role)};
      const nameMatch = ${nameJSON};
      const rootSel = ${JSON.stringify(parentSel)};
      const root = rootSel ? document.querySelector(rootSel) : document;
      if (!root) return [];
      const roleSelectors = {
        button: 'button, [role="button"], input[type="button"], input[type="submit"], input[type="reset"]',
        option: '[role="option"], option',
        dialog: '[role="dialog"], dialog, .el-dialog',
        menuitem: '[role="menuitem"]',
      };
      const sel = roleSelectors[role] || ('[role="' + String(role).replace(/"/g, '') + '"]');
      if (!sel || sel === '[role=""]') return [];
      const isVisibleEl = (el) => {
        const style = window.getComputedStyle(el);
        if (!style || style.visibility === 'hidden' || style.display === 'none' || parseFloat(style.opacity) === 0) return false;
        const r = el.getBoundingClientRect();
        return r.width > 0 && r.height > 0;
      };
      const accessibleName = (el) => {
        const labelled = el.getAttribute('aria-label');
        if (labelled) return String(labelled).trim();
        const labelledBy = el.getAttribute('aria-labelledby');
        if (labelledBy) {
          const parts = String(labelledBy).split(/\\s+/).map(id => {
            const n = document.getElementById(id);
            return n ? (n.textContent || '').trim() : '';
          }).filter(Boolean);
          if (parts.length) return parts.join(' ');
        }
        if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
          const v = el.getAttribute('value') || el.value || '';
          if (v) return String(v).trim();
        }
        return String(el.textContent || '').replace(/\\s+/g, ' ').trim();
      };
      const nameOk = (el) => {
        if (!nameMatch) return true;
        const name = accessibleName(el);
        if (nameMatch.kind === 're') {
          try { return new RegExp(nameMatch.source, nameMatch.flags || '').test(name); }
          catch { return false; }
        }
        const want = String(nameMatch.value || '');
        if (nameMatch.exact) return name === want;
        return name.includes(want);
      };
      let nodes = Array.from(root.querySelectorAll(sel)).filter(nameOk);
      if (role === 'dialog') {
        nodes = nodes.map((el) => {
          if (el.classList && el.classList.contains('el-dialog')) return el;
          const inner = el.querySelector ? el.querySelector('.el-dialog') : null;
          return inner || el;
        });
        nodes = nodes.filter((el, i) => nodes.indexOf(el) === i);
      }
      const visible = nodes.filter(isVisibleEl);
      if (visible.length) nodes = visible;
      if (role === 'dialog') {
        const content = nodes.filter((el) => el.classList && el.classList.contains('el-dialog'));
        if (content.length) nodes = content;
      }
      return nodes.map(el => el.textContent);
    })()`);
    return JSON.parse(raw);
  }

  return [];
}

let routeEntries = [];
let routePumpRunning = false;
let routePumpStop = false;

function globToRegExp(glob) {
  const s = String(glob || '');
  let out = '^';
  for (let i = 0; i < s.length; i++) {
    const ch = s[i];
    if (ch === '*' && s[i + 1] === '*') {
      out += '.*';
      i++;
      continue;
    }
    if (ch === '*') {
      out += '[^/]*';
      continue;
    }
    if (ch === '?') {
      out += '.';
      continue;
    }
    if ('\\.[]{}()+-^$|'.includes(ch)) {
      out += '\\' + ch;
      continue;
    }
    out += ch;
  }
  out += '(?:\\?.*)?$';
  return new RegExp(out);
}

function urlMatchesPattern(href, pattern) {
  if (pattern && typeof pattern === 'object' && typeof pattern.test === 'function') {
    return pattern.test(href);
  }
  const s = String(pattern || '');
  if (!s) return true;
  if (s.includes('*') || s.includes('?')) {
    return globToRegExp(s).test(href);
  }
  return href === s || href.includes(s);
}

function routePatternsEqual(a, b) {
  if (a && typeof a === 'object' && typeof a.test === 'function') {
    if (!(b && typeof b === 'object' && typeof b.test === 'function')) return false;
    return a === b || (a.source === b.source && a.flags === b.flags);
  }
  return String(a) === String(b);
}

function matchRouteEntry(url) {
  for (let i = 0; i < routeEntries.length; i++) {
    if (urlMatchesPattern(url, routeEntries[i].pattern)) {
      return routeEntries[i];
    }
  }
  return null;
}

function makeRouteHandle(paused) {
  let settled = false;
  const settle = async fn => {
    if (settled) return;
    settled = true;
    await fn();
  };
  return {
    request() {
      return {
        url: () => paused.url,
        method: () => paused.method,
      };
    },
    async fulfill(opts) {
      const o = opts || {};
      await settle(() =>
        getHost().fulfillRequest(
          paused.id,
          JSON.stringify({
            status: o.status != null ? o.status : 200,
            contentType: o.contentType || '',
            body: o.body != null ? String(o.body) : '',
            headers: o.headers || {},
          })
        )
      );
    },
    async continue() {
      await settle(() => getHost().continueRequest(paused.id));
    },
    async abort() {
      // Minimal surface: abort by fulfilling a connection-reset-like 500 empty body.
      await settle(() =>
        getHost().fulfillRequest(
          paused.id,
          JSON.stringify({ status: 500, contentType: 'text/plain', body: '' })
        )
      );
    },
  };
}

function startRoutePump() {
  // Clear stop before the running check so unroute→route in the same test
  // (or while a prior wait is in flight) keeps draining paused requests.
  routePumpStop = false;
  if (routePumpRunning) return;
  routePumpRunning = true;
  (async () => {
    while (!routePumpStop && routeEntries.length > 0) {
      let raw;
      try {
        raw = await getHost().waitPausedRequest(1000);
      } catch {
        if (routePumpStop || routeEntries.length === 0) break;
        continue;
      }
      if (raw == null || raw === '') continue;
      let paused;
      try {
        paused = typeof raw === 'string' ? JSON.parse(raw) : raw;
      } catch {
        continue;
      }
      if (!paused || !paused.id) continue;
      const entry = matchRouteEntry(String(paused.url || ''));
      if (!entry) {
        try {
          await getHost().continueRequest(paused.id);
        } catch {
          // ignore
        }
        continue;
      }
      const handle = makeRouteHandle({
        id: String(paused.id),
        url: String(paused.url || ''),
        method: String(paused.method || ''),
      });
      // Do not await the handler: parallel paused requests must keep draining
      // (Playwright also invokes route handlers without serializing them).
      (async () => {
        try {
          await entry.handler(handle);
          // If the handler returned without fulfill/continue, fail open.
          await handle.continue().catch(() => {});
        } catch (err) {
          await handle.continue().catch(() => {});
          // Keep pumping; surface via the originating test if it awaits the same work.
          // QuickJS host may not define global console.
          if (typeof console !== 'undefined' && console.warn) {
            console.warn('[choysum/e2e] route handler error:', err && err.message ? err.message : err);
          }
        }
      })();
    }
    routePumpRunning = false;
  })().catch(() => {
    routePumpRunning = false;
  });
}

const page = {
  __choysum_e2e_page__: true,
  async goto(url, opts) {
    const waitUntil = opts && opts.waitUntil ? opts.waitUntil : 'load';
    await getHost().goto(String(url), waitUntil);
  },
  async reload(opts) {
    const waitUntil = opts && opts.waitUntil ? opts.waitUntil : 'load';
    await getHost().reload(waitUntil);
  },
  locator(sel, opts) {
    return makeLocator('css', String(sel), {
      hasText: opts && opts.hasText != null ? opts.hasText : null,
    });
  },
  getByPlaceholder(reOrString) {
    return makeLocator('placeholder', reOrString);
  },
  getByText(text) {
    return makeLocator('text', text);
  },
  getByTestId(id) {
    return makeLocator('testid', String(id));
  },
  getByRole(role, opts) {
    const nameMatch = serializeNameMatch(opts && opts.name, opts && opts.exact);
    return makeLocator('role', String(role), { nameMatch });
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
  async waitForTimeout(ms) {
    await sleep(ms);
  },
  async waitForURL(pattern, opts) {
    const timeout = opts && typeof opts.timeout === 'number' ? opts.timeout : 30000;
    await poll(timeout, async () => {
      const href = String(await getHost().url());
      return urlMatchesPattern(href, pattern);
    });
  },
  async waitForSelector(sel, opts) {
    const timeout = opts && typeof opts.timeout === 'number' ? opts.timeout : 30000;
    const state = opts && opts.state ? String(opts.state) : 'visible';
    await poll(timeout, async () => {
      const css = String(sel);
      if (state === 'attached') {
        return (await getHost().count(css)) > 0;
      }
      if (state === 'hidden') {
        const n = await getHost().count(css);
        if (n === 0) return true;
        return !(await getHost().isVisible(css));
      }
      // visible (default)
      return await getHost().isVisible(css);
    });
  },
  async route(url, handler) {
    if (typeof handler !== 'function') {
      throw new Error('@choysum/e2e: page.route requires a handler function');
    }
    // Keep RegExp objects so urlMatchesPattern can call .test(); String(re) breaks matching.
    const pattern =
      url && typeof url === 'object' && typeof url.test === 'function' ? url : String(url);
    routeEntries.unshift({ pattern, handler });
    await getHost().enableFetch();
    startRoutePump();
  },
  async unroute(url, handler) {
    const want = url != null ? (url && typeof url === 'object' && typeof url.test === 'function' ? url : String(url)) : null;
    routeEntries = routeEntries.filter(entry => {
      if (want != null && !routePatternsEqual(entry.pattern, want)) return true;
      if (handler && entry.handler !== handler) return true;
      return false;
    });
    if (routeEntries.length === 0) {
      routePumpStop = true;
      try {
        await getHost().disableFetch();
      } catch {
        // ignore
      }
    }
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

async function pollOrFail(timeoutMs, diag, fn) {
  try {
    await poll(timeoutMs, fn);
  } catch (e) {
    const msg = e && e.message ? e.message : String(e);
    throw new Error(diag ? `${diag}: ${msg}` : msg);
  }
}

function valuesEqual(a, b) {
  if (Object.is(a, b)) return true;
  if (typeof a === 'number' && typeof b === 'number' && Number.isNaN(a) && Number.isNaN(b)) return true;
  return false;
}

function matchValue(actual, expected) {
  if (expected && typeof expected === 'object' && typeof expected.test === 'function') {
    return expected.test(String(actual == null ? '' : actual));
  }
  return String(actual == null ? '' : actual).includes(String(expected));
}

function matchTextExact(actual, expected) {
  if (expected && typeof expected === 'object' && typeof expected.test === 'function') {
    return expected.test(String(actual == null ? '' : actual));
  }
  const normalize = value => String(value == null ? '' : value).replace(/\s+/g, ' ').trim();
  return normalize(actual) === normalize(expected);
}

function makePollMatchers(fn, timeoutMs, negate) {
  const api = {
    async toBe(expected) {
      await poll(timeoutMs, async () => {
        const v = await fn();
        return negate ? !valuesEqual(v, expected) : valuesEqual(v, expected);
      });
    },
    async toBeGreaterThan(expected) {
      await poll(timeoutMs, async () => {
        const v = await fn();
        return negate ? !(Number(v) > Number(expected)) : Number(v) > Number(expected);
      });
    },
    async toBeGreaterThanOrEqual(expected) {
      await poll(timeoutMs, async () => {
        const v = await fn();
        return negate ? !(Number(v) >= Number(expected)) : Number(v) >= Number(expected);
      });
    },
    async toMatch(expected) {
      await poll(timeoutMs, async () => {
        const v = await fn();
        const ok = matchValue(v, expected);
        return negate ? !ok : ok;
      });
    },
  };
  if (!negate) {
    api.not = makePollMatchers(fn, timeoutMs, true);
  }
  return api;
}

function e2eExpect(target, message) {
  const diag = message != null && message !== '' ? String(message) : '';
  const fail = msg => {
    throw new Error(diag ? `${diag}: ${msg}` : msg);
  };
  const api = {
    async toBeVisible(opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('toBeVisible: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const sel = await ensureCSS(target);
          const ok = await getHost().isVisible(sel);
          // Role/text locators stamp DOM nodes; Vue re-renders drop the stamp.
          if (!ok) target._css = '';
          return ok;
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
    async toBeEnabled(opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('toBeEnabled: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const sel = await ensureCSS(target);
          const exists = !!JSON.parse(
            await getHost().evaluate(`!!document.querySelector(${JSON.stringify(sel)})`)
          );
          // Only clear when the stamped node is gone; disabled-but-present must keep polling.
          if (!exists) {
            target._css = '';
            return false;
          }
          return await getHost().isEnabled(sel);
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
    async toBeChecked(opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('toBeChecked: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const sel = await ensureCSS(target);
          const raw = await getHost().evaluate(`(() => {
            const el = document.querySelector(${JSON.stringify(sel)});
            if (!el) return { exists: false, checked: false };
            let checked = false;
            if (typeof el.checked === 'boolean') checked = el.checked;
            else if (el.classList && el.classList.contains('is-checked')) checked = true;
            else checked = el.getAttribute('aria-checked') === 'true';
            return { exists: true, checked };
          })()`);
          const res = JSON.parse(raw);
          // Only clear the stamped CSS when the node is gone; unchecked must keep polling the same el.
          if (!res || !res.exists) {
            target._css = '';
            return false;
          }
          return !!res.checked;
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
    async toHaveCount(n, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('toHaveCount: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        const count = await countLocator(target);
        return count === n;
      });
    },
    async toHaveURL(reOrString, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_page__) {
        fail('toHaveURL: expected page');
      }
      const isRe =
        reOrString && typeof reOrString === 'object' && typeof reOrString.test === 'function';
      const re = isRe ? reOrString : null;
      let exact = '';
      if (!isRe) {
        exact = String(reOrString);
        const base = getRuntime().baseURL;
        if (base && !/^[a-z][a-z0-9+.-]*:\/\//i.test(exact)) {
          exact = new URL(exact, base).href;
        }
      }
      await pollOrFail(timeout, diag, async () => {
        const href = String(await getHost().url());
        if (re) return re.test(href);
        // String form: match the full URL (Playwright string semantics), not a RegExp.
        return href === exact;
      });
    },
    async toHaveText(expected, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('toHaveText: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const text = await target.textContent();
          // Missing element → null; do not treat as empty text (Playwright waits for attach).
          if (text === null) return false;
          return matchTextExact(text, expected);
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
    async toHaveClass(expected, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('toHaveClass: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const sel = await ensureCSS(target);
          const raw = await getHost().evaluate(`(() => {
            const el = document.querySelector(${JSON.stringify(sel)});
            // getAttribute works for HTML and SVG (className is SVGAnimatedString on SVG).
            return el ? (el.getAttribute('class') || '') : null;
          })()`);
          const className = JSON.parse(raw);
          if (className == null) {
            target._css = '';
            return false;
          }
          return matchValue(className, expected);
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
  };

  api.not = {
    async toHaveText(expected, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('not.toHaveText: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const text = await target.textContent();
          if (text === null) return false;
          return !matchTextExact(text, expected);
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
    async toHaveClass(expected, opts) {
      const timeout = opts && opts.timeout != null ? opts.timeout : 30000;
      if (!target || !target.__choysum_e2e_locator__) {
        fail('not.toHaveClass: expected locator');
      }
      await pollOrFail(timeout, diag, async () => {
        try {
          const sel = await ensureCSS(target);
          const raw = await getHost().evaluate(`(() => {
            const el = document.querySelector(${JSON.stringify(sel)});
            return el ? (el.getAttribute('class') || '') : null;
          })()`);
          const className = JSON.parse(raw);
          if (className == null) {
            target._css = '';
            return false;
          }
          return !matchValue(className, expected);
        } catch (e) {
          target._css = '';
          throw e;
        }
      });
    },
  };

  if (target && (target.__choysum_e2e_locator__ || target.__choysum_e2e_page__)) {
    return api;
  }
  if (typeof globalThis.expect === 'function') {
    return globalThis.expect(target, message);
  }
  fail('@choysum/e2e: expect target not supported');
}

e2eExpect.poll = function pollExpect(fn, opts) {
  const timeout = opts && typeof opts.timeout === 'number' ? opts.timeout : 30000;
  if (typeof fn !== 'function') {
    throw new Error('@choysum/e2e: expect.poll requires a function');
  }
  return makePollMatchers(fn, timeout, false);
};

function randomUUID() {
  if (typeof globalThis.crypto !== 'undefined' && typeof globalThis.crypto.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}

async function readTextFile(path) {
  return await getHost().readTextFile(String(path));
}

function normalizeFetchHeaders(headers) {
  if (!headers) return {};
  // Arrays also have forEach; map [name, value] pairs before the Headers-like path.
  if (Array.isArray(headers)) {
    const out = {};
    for (const pair of headers) {
      if (!pair || pair.length < 2) continue;
      out[String(pair[0])] = String(pair[1]);
    }
    return out;
  }
  if (typeof headers.forEach === 'function') {
    const out = {};
    headers.forEach((v, k) => {
      out[String(k)] = String(v);
    });
    return out;
  }
  const out = {};
  for (const [k, v] of Object.entries(headers)) {
    out[String(k)] = String(v);
  }
  return out;
}

function installFetchPolyfill() {
  if (globalThis.__choysum_e2e_fetch_installed__) return;
  globalThis.__choysum_e2e_fetch_installed__ = true;

  // connect-web probes `new Headers()` at transport create time.
  if (typeof globalThis.Headers !== 'function') {
    globalThis.Headers = class Headers {
      constructor(init) {
        this._map = {};
        if (!init) return;
        // Arrays also have forEach; handle [name, value] tuples first.
        if (Array.isArray(init)) {
          for (const pair of init) {
            if (!pair || pair.length < 2) continue;
            this._map[String(pair[0]).toLowerCase()] = String(pair[1]);
          }
          return;
        }
        if (typeof init.forEach === 'function') {
          init.forEach((v, k) => {
            this._map[String(k).toLowerCase()] = String(v);
          });
          return;
        }
        for (const [k, v] of Object.entries(init)) {
          this._map[String(k).toLowerCase()] = String(v);
        }
      }
      get(name) {
        const v = this._map[String(name).toLowerCase()];
        return v == null ? null : v;
      }
      set(name, value) {
        this._map[String(name).toLowerCase()] = String(value);
      }
      has(name) {
        return Object.prototype.hasOwnProperty.call(this._map, String(name).toLowerCase());
      }
      append(name, value) {
        const key = String(name).toLowerCase();
        if (this._map[key] == null) this._map[key] = String(value);
        else this._map[key] = this._map[key] + ', ' + String(value);
      }
      delete(name) {
        delete this._map[String(name).toLowerCase()];
      }
      forEach(fn, thisArg) {
        for (const [k, v] of Object.entries(this._map)) {
          fn.call(thisArg, v, k, this);
        }
      }
      entries() {
        return Object.entries(this._map)[Symbol.iterator]();
      }
      keys() {
        return Object.keys(this._map)[Symbol.iterator]();
      }
      values() {
        return Object.values(this._map)[Symbol.iterator]();
      }
      [Symbol.iterator]() {
        return this.entries();
      }
    };
  }

  if (typeof globalThis.AbortController !== 'function') {
    globalThis.AbortSignal = class AbortSignal {
      constructor() {
        this.aborted = false;
        this.reason = undefined;
        this._listeners = [];
      }
      addEventListener(type, fn) {
        if (type === 'abort' && typeof fn === 'function') this._listeners.push(fn);
      }
      removeEventListener(type, fn) {
        if (type !== 'abort') return;
        this._listeners = this._listeners.filter(f => f !== fn);
      }
      throwIfAborted() {
        if (this.aborted) {
          const err = this.reason || new Error('Aborted');
          throw err;
        }
      }
      _abort(reason) {
        if (this.aborted) return;
        this.aborted = true;
        this.reason = reason;
        for (const fn of this._listeners.slice()) {
          try {
            fn();
          } catch {
            // ignore
          }
        }
      }
    };
    globalThis.AbortController = class AbortController {
      constructor() {
        this.signal = new globalThis.AbortSignal();
      }
      abort(reason) {
        this.signal._abort(reason !== undefined ? reason : new Error('Aborted'));
      }
    };
  }

  if (typeof globalThis.ReadableStream !== 'function') {
    globalThis.ReadableStream = class ReadableStream {
      constructor(src) {
        this._queue = [];
        this._closed = false;
        this._error = null;
        this._src = src || null;
        this._started = false;
        const self = this;
        this._controller = {
          enqueue: chunk => {
            if (self._closed) throw new TypeError('ReadableStream is closed');
            self._queue.push(chunk);
          },
          close: () => {
            if (self._closed) throw new TypeError('ReadableStream is closed');
            self._closed = true;
          },
          error: err => {
            self._error = err || new Error('ReadableStream error');
            self._closed = true;
          },
        };
      }
      static from(iterable) {
        const stream = new globalThis.ReadableStream();
        if (iterable == null) {
          stream._closed = true;
          return stream;
        }
        if (ArrayBuffer.isView(iterable)) {
          stream._queue.push(iterable);
          stream._closed = true;
          return stream;
        }
        if (iterable instanceof ArrayBuffer) {
          stream._queue.push(new Uint8Array(iterable));
          stream._closed = true;
          return stream;
        }
        if (Array.isArray(iterable)) {
          for (const c of iterable) stream._queue.push(c);
          stream._closed = true;
          return stream;
        }
        if (typeof iterable[Symbol.iterator] === 'function') {
          for (const c of iterable) stream._queue.push(c);
        }
        stream._closed = true;
        return stream;
      }
      async _ensureStart() {
        if (this._started) return;
        this._started = true;
        if (this._src && typeof this._src.start === 'function') {
          await this._src.start(this._controller);
        }
      }
      getReader() {
        const self = this;
        return {
          async read() {
            await self._ensureStart();
            let spins = 0;
            while (self._queue.length === 0 && !self._closed) {
              if (self._error) throw self._error;
              if (self._src && typeof self._src.pull === 'function') {
                const before = self._queue.length;
                const closedBefore = self._closed;
                try {
                  await self._src.pull(self._controller);
                } catch (e) {
                  // connect-web's envelope stream calls close() then loops; a
                  // second close() throws TypeError which ends pull.
                  if (self._closed || self._error) break;
                  throw e;
                }
                if (self._queue.length === before && self._closed === closedBefore) {
                  spins++;
                  if (spins > 8) {
                    self._closed = true;
                    break;
                  }
                  await sleep(0);
                } else {
                  spins = 0;
                }
                continue;
              }
              self._closed = true;
              break;
            }
            if (self._error) throw self._error;
            if (self._queue.length) return { done: false, value: self._queue.shift() };
            return { done: true, value: undefined };
          },
          async cancel() {
            self._queue = [];
            self._closed = true;
          },
        };
      }
    };
  }

  globalThis.fetch = async (input, init) => {
    const url = typeof input === 'string' ? input : String(input && input.url != null ? input.url : input);
    const method = (init && init.method) || (input && input.method) || 'GET';
    const headers = normalizeFetchHeaders((init && init.headers) || (input && input.headers));
    const signal = init && init.signal ? init.signal : null;
    if (signal && signal.aborted) {
      const err = new Error('The operation was aborted');
      err.name = 'AbortError';
      throw err;
    }
    const payload = { method: String(method), headers };
    const body = init && init.body != null ? init.body : null;
    if (body != null) {
      if (typeof body === 'string') {
        payload.body = body;
      } else if (body instanceof ArrayBuffer) {
        payload.bodyBase64 = encodeUint8ArrayToBase64(new Uint8Array(body));
      } else if (ArrayBuffer.isView(body)) {
        payload.bodyBase64 = encodeUint8ArrayToBase64(
          new Uint8Array(body.buffer, body.byteOffset, body.byteLength)
        );
      } else {
        payload.body = String(body);
      }
    }
    const hostFetch = getHost().fetch(url, JSON.stringify(payload));
    let rawJSON;
    if (!signal) {
      rawJSON = await hostFetch;
    } else {
      rawJSON = await new Promise((resolve, reject) => {
        const onAbort = () => {
          const err = new Error('The operation was aborted');
          err.name = 'AbortError';
          reject(err);
        };
        signal.addEventListener('abort', onAbort);
        hostFetch.then(
          (v) => {
            signal.removeEventListener('abort', onAbort);
            resolve(v);
          },
          (e) => {
            signal.removeEventListener('abort', onAbort);
            reject(e);
          }
        );
      });
    }
    const raw = typeof rawJSON === 'string' ? JSON.parse(rawJSON) : rawJSON;
    const bodyBytes = decodeBase64ToUint8Array(raw.bodyBase64 || '');
    const headersObj = raw.headers || {};
    const status = Number(raw.status) || 0;
    const headerBag = new globalThis.Headers(headersObj);

    // grpc-web reads response.body via getReader().
    const bodyStream = new globalThis.ReadableStream({
      start(controller) {
        if (bodyBytes.length) controller.enqueue(bodyBytes);
        controller.close();
      },
    });

    return {
      ok: status >= 200 && status < 300,
      status,
      statusText: String(raw.statusText || ''),
      url: String(raw.url || url),
      headers: headerBag,
      body: bodyStream,
      async arrayBuffer() {
        return bodyBytes.buffer.slice(bodyBytes.byteOffset, bodyBytes.byteOffset + bodyBytes.byteLength);
      },
      async text() {
        try {
          if (typeof TextDecoder === 'function') {
            return new TextDecoder('utf-8').decode(bodyBytes);
          }
        } catch {
          // fall through
        }
        let s = '';
        for (let i = 0; i < bodyBytes.length; i++) s += String.fromCharCode(bodyBytes[i]);
        return s;
      },
      async json() {
        return JSON.parse(await this.text());
      },
    };
  };
}

installFetchPolyfill();

function wrapTest(rawTest) {
  if (typeof rawTest !== 'function') {
    return rawTest;
  }
  const wrapped = function test(name, fn) {
    return rawTest(name, fn);
  };
  for (const key of Object.keys(rawTest)) {
    wrapped[key] = rawTest[key];
  }
  // choysumtest has no per-test timeout; keep call sites compiling.
  wrapped.setTimeout = function setTimeout() {};
  // Prefer early return in specs; ChoysumTestSkip is counted as pass (skipped) by choysumtest.
  wrapped.skip = function skip(cond, msg) {
    const shouldSkip = arguments.length === 0 ? true : !!cond;
    if (shouldSkip) {
      const e = new Error(msg != null ? String(msg) : 'skipped');
      e.name = 'ChoysumTestSkip';
      throw e;
    }
  };
  return wrapped;
}

const test = wrapTest(globalThis.test);
const expect = e2eExpect;
const runtime = new Proxy(
  {},
  {
    get(_t, prop) {
      return getRuntime()[prop];
    },
  }
);

if (!globalThis.__choysum_e2e_hooks_installed__ && typeof globalThis.beforeEach === 'function') {
  globalThis.__choysum_e2e_hooks_installed__ = true;
  globalThis.beforeEach(async () => {
    routeEntries = [];
    routePumpStop = true;
    // NewPage clears cookies and resets to about:blank (workers=1 reuses the tab).
    await getHost().newPage();
  });
  if (typeof globalThis.afterEach === 'function') {
    globalThis.afterEach(async () => {
      routeEntries = [];
      routePumpStop = true;
      try {
        await getHost().disableFetch();
      } catch {
        // ignore
      }
      try {
        await getHost().closePage();
      } catch {
        // ignore
      }
    });
  }
}

export { test, expect, page, runtime, randomUUID, readTextFile };
