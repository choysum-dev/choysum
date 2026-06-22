// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Minimal file:// URL parser and formatter — replaces the npm 'url' package
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

  // Match protocol (e.g. "file://")
  const protoMatch = url.match(/^([a-z][a-z0-9+\-.]*):\/\//i);
  if (protoMatch) {
    result.protocol = protoMatch[1].toLowerCase();
    result.slashes = true;
    const rest = url.slice(protoMatch[0].length);
    // Split path from hash/search
    const hashIdx = rest.indexOf('#');
    const searchIdx = rest.indexOf('?');
    let pathEnd = rest.length;
    if (hashIdx >= 0) {
      result.hash = rest.slice(hashIdx + 1);
      pathEnd = hashIdx;
    }
    if (searchIdx >= 0 && searchIdx < pathEnd) {
      result.search = rest.slice(searchIdx, pathEnd);
      result.query = rest.slice(searchIdx + 1, pathEnd);
      pathEnd = searchIdx;
    }
    result.pathname = rest.slice(0, pathEnd);
    result.path = result.pathname + result.search;
  } else {
    result.pathname = url;
    result.path = url;
  }

  return result;
}

function format(urlObj: ParsedUrl): string {
  let result = '';
  if (urlObj.protocol) {
    result += urlObj.protocol + '://';
  }
  result += urlObj.pathname || '';
  if (urlObj.search) {
    result += urlObj.search;
  }
  if (urlObj.hash) {
    result += '#' + urlObj.hash;
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
  searchParams: any = undefined;
  username: string = '';

  constructor(url: string) {
    this.parse(url);
  }

  parse(url: string) {
    const parsed = parse(url);
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
    this.path = parsed.path || '';
    this.href = parsed.href || '';
  }

  toString() {
    return format(this);
  }

  toJSON() {
    return JSON.stringify(this);
  }
}

(globalThis as any).URL = URL;
