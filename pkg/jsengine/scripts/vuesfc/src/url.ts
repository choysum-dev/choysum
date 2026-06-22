// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Minimal URL parser and formatter — replaces the npm 'url' package
// to avoid esm.sh CJS interop issues in QuickJS runtime.

interface ParsedUrl {
  protocol: string;
  slashes: boolean;
  auth: string;
  host: string;
  port: string;
  hostname: string;
  hash: string;
  search: string;
  query: string;
  pathname: string;
  path: string;
  href: string;
}

function parse(url: string): ParsedUrl {
  const result: ParsedUrl = {
    protocol: '',
    slashes: false,
    auth: '',
    host: '',
    port: '',
    hostname: '',
    hash: '',
    search: '',
    query: '',
    pathname: '',
    path: '',
    href: url,
  };

  // Match protocol (e.g. "file://") or protocol-relative "//" URLs.
  const protoMatch = url.match(/^([a-z][a-z0-9+\-.]*):\/\//i);
  const isProtoRelative = url.startsWith('//');
  let rest = url;
  if (protoMatch || isProtoRelative) {
    if (protoMatch) {
      result.protocol = protoMatch[1].toLowerCase() + ':';
      result.slashes = true;
      rest = url.slice(protoMatch[0].length);
    } else {
      result.slashes = true;
      rest = url.slice(2);
    }

    // Extract authority (host[:port]) when present.
    const slashIdx = rest.indexOf('/');
    const questionIdx = rest.indexOf('?');
    const hashIdx = rest.indexOf('#');

    let authorityEnd = rest.length;
    if (slashIdx >= 0) {
      authorityEnd = slashIdx;
    }
    if (questionIdx >= 0 && questionIdx < authorityEnd) {
      authorityEnd = questionIdx;
    }
    if (hashIdx >= 0 && hashIdx < authorityEnd) {
      authorityEnd = hashIdx;
    }

    const authority = rest.slice(0, authorityEnd);
    rest = rest.slice(authorityEnd);

    if (authority) {
      const authIdx = authority.indexOf('@');
      let hostPart = authority;
      if (authIdx >= 0) {
        result.auth = authority.slice(0, authIdx);
        hostPart = authority.slice(authIdx + 1);
      }

      result.host = hostPart;
      // Keep IPv6 host intact when enclosed in [] and parse trailing :port.
      if (hostPart.startsWith('[')) {
        const closingIdx = hostPart.indexOf(']');
        if (closingIdx >= 0) {
          result.hostname = hostPart.slice(0, closingIdx + 1);
          if (closingIdx + 1 < hostPart.length && hostPart[closingIdx + 1] === ':') {
            result.port = hostPart.slice(closingIdx + 2);
          }
        } else {
          result.hostname = hostPart;
        }
      } else {
        const portIdx = hostPart.lastIndexOf(':');
        if (portIdx > 0) {
          result.hostname = hostPart.slice(0, portIdx);
          result.port = hostPart.slice(portIdx + 1);
        } else {
          result.hostname = hostPart;
        }
      }
    }
  }

  // Split path from hash/search for both absolute and relative inputs.
  const hashIdx = rest.indexOf('#');
  const searchIdx = rest.indexOf('?');
  let pathEnd = rest.length;
  if (hashIdx >= 0) {
    // Keep the leading '#' to match Node legacy url.parse behavior.
    result.hash = rest.slice(hashIdx);
    pathEnd = hashIdx;
  }
  if (searchIdx >= 0 && searchIdx < pathEnd) {
    result.search = rest.slice(searchIdx, pathEnd);
    result.query = rest.slice(searchIdx + 1, pathEnd);
    pathEnd = searchIdx;
  }
  result.pathname = rest.slice(0, pathEnd);
  result.path = result.pathname + result.search;

  return result;
}

function normalizePath(pathname: string): string {
  const isAbsolute = pathname.startsWith('/');
  const segments = pathname.split('/');
  const normalized: string[] = [];

  for (const segment of segments) {
    if (!segment || segment === '.') {
      continue;
    }
    if (segment === '..') {
      if (normalized.length > 0) {
        normalized.pop();
      }
      continue;
    }
    normalized.push(segment);
  }

  const joined = normalized.join('/');
  if (isAbsolute) {
    return '/' + joined;
  }
  return joined || '';
}

function applyBase(parsed: ParsedUrl, baseParsed: ParsedUrl): ParsedUrl {
  const hasOwnAuthority = parsed.host !== '' || parsed.hostname !== '' || parsed.auth !== '';

  const resolved: ParsedUrl = {
    ...parsed,
    protocol: baseParsed.protocol,
    slashes: parsed.slashes || baseParsed.slashes,
    auth: hasOwnAuthority ? parsed.auth : baseParsed.auth,
    host: hasOwnAuthority ? parsed.host : baseParsed.host,
    port: hasOwnAuthority ? parsed.port : baseParsed.port,
    hostname: hasOwnAuthority ? parsed.hostname : baseParsed.hostname,
  };

  if (hasOwnAuthority) {
    resolved.path = resolved.pathname + resolved.search;
    return resolved;
  }

  if (resolved.pathname.startsWith('/')) {
    resolved.path = resolved.pathname + resolved.search;
    return resolved;
  }

  const basePath = baseParsed.pathname || '/';
  if (resolved.pathname) {
    const baseDirIdx = basePath.lastIndexOf('/');
    const baseDir = baseDirIdx >= 0 ? basePath.slice(0, baseDirIdx + 1) : '/';
    resolved.pathname = normalizePath(baseDir + resolved.pathname);
  } else {
    resolved.pathname = basePath;
    if (!resolved.search) {
      resolved.search = baseParsed.search;
      resolved.query = baseParsed.query;
    }
  }

  resolved.path = resolved.pathname + resolved.search;
  return resolved;
}

function decodeQueryComponent(value: string): string {
  const normalized = value.replace(/\+/g, ' ');
  try {
    return decodeURIComponent(normalized);
  } catch {
    return normalized;
  }
}

function encodeQueryComponent(value: string): string {
  return encodeURIComponent(value);
}

class URLSearchParamsShim {
  private params: Array<[string, string]> = [];

  constructor(search: string = '') {
    const query = search.startsWith('?') ? search.slice(1) : search;
    if (!query) {
      return;
    }
    for (const pair of query.split('&')) {
      if (!pair) {
        continue;
      }
      const idx = pair.indexOf('=');
      const rawKey = idx >= 0 ? pair.slice(0, idx) : pair;
      const rawValue = idx >= 0 ? pair.slice(idx + 1) : '';
      const key = decodeQueryComponent(rawKey);
      const value = decodeQueryComponent(rawValue);
      this.params.push([key, value]);
    }
  }

  get(name: string): string | null {
    for (const [key, value] of this.params) {
      if (key === name) {
        return value;
      }
    }
    return null;
  }

  getAll(name: string): string[] {
    const values: string[] = [];
    for (const [key, value] of this.params) {
      if (key === name) {
        values.push(value);
      }
    }
    return values;
  }

  has(name: string): boolean {
    return this.get(name) !== null;
  }

  toString(): string {
    return this.params.map(([key, value]) => encodeQueryComponent(key) + '=' + encodeQueryComponent(value)).join('&');
  }
}

function format(urlObj: ParsedUrl): string {
  let result = '';
  if (urlObj.protocol) {
    const proto = urlObj.protocol.endsWith(':') ? urlObj.protocol : urlObj.protocol + ':';
    result += proto + (urlObj.slashes ? '//' : '');
  }
  if (urlObj.auth) {
    result += urlObj.auth + '@';
  }
  let host = urlObj.host;
  if (!host && urlObj.hostname) {
    host = urlObj.hostname;
    if (urlObj.port) {
      host += ':' + urlObj.port;
    }
  }
  if (host) {
    result += host;
  }
  let pathname = urlObj.pathname || '';
  if ((host || urlObj.slashes) && pathname && !pathname.startsWith('/')) {
    pathname = '/' + pathname;
  }
  result += pathname;
  if (urlObj.search) {
    result += urlObj.search;
  }
  if (urlObj.hash) {
    result += urlObj.hash.startsWith('#') ? urlObj.hash : '#' + urlObj.hash;
  }
  return result;
}

export class URL {
  protocol: string = '';
  slashes: boolean = false;
  auth: string = '';
  host: string = '';
  port: string = '';
  hostname: string = '';
  hash: string = '';
  search: string = '';
  query: string = '';
  pathname: string = '';
  path: string = '';
  href: string = '';
  url: string = '';

  origin: string = '';
  password: string = '';
  searchParams: any = new URLSearchParamsShim();
  username: string = '';

  constructor(url: string, base?: string | URL) {
    this.parse(url, base);
  }

  parse(url: string, base?: string | URL) {
    let parsed = parse(url);
    if (!parsed.protocol && base) {
      const baseSource = typeof base === 'string' ? base : base.toString();
      const baseParsed = parse(baseSource);
      if (baseParsed.protocol) {
        parsed = applyBase(parsed, baseParsed);
      }
    }

    this.url = url;
    this.protocol = parsed.protocol || '';
    this.slashes = parsed.slashes || false;
    this.auth = parsed.auth || '';
    this.host = parsed.host || '';
    this.port = parsed.port || '';
    this.hostname = parsed.hostname || '';
    this.hash = parsed.hash || '';
    this.search = parsed.search || '';
    this.query = parsed.query || '';
    this.pathname = parsed.pathname || '';
    this.path = this.pathname + this.search;
    this.searchParams = new URLSearchParamsShim(this.search);
    this.href = this.toString();

    this.origin = this.protocol && this.host ? this.protocol + '//' + this.host : '';
    if (this.auth) {
      const sep = this.auth.indexOf(':');
      if (sep >= 0) {
        this.username = this.auth.slice(0, sep);
        this.password = this.auth.slice(sep + 1);
      } else {
        this.username = this.auth;
        this.password = '';
      }
    } else {
      this.username = '';
      this.password = '';
    }
  }

  toString() {
    return format(this);
  }

  toJSON() {
    return this.toString();
  }
}

if (typeof (globalThis as any).URL === 'undefined') {
  (globalThis as any).URL = URL;
}
