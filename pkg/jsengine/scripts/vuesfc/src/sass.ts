// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import { join, dirname, basename, isAbsolute, extname } from 'path-browserify';
import { compileString, ImporterResult, CanonicalizeContext, CompileResult } from 'sass';

import { compilerFs } from './compiler-fs';
import { URL } from './url';

let sasslocation: string | undefined;

type SassRenderResult = {
  css: string;
  map: string;
  stats: {
    entry: string;
    start: number;
    end: number;
    duration: number;
    includedFiles: string[];
  };
};

function ensureRuntimeLocation(baseDir?: string) {
  const g: any = globalThis as any;
  if (g.location && typeof g.location.href === 'string' && g.location.href.length > 0) {
    return;
  }

  const normalized = toPosixPath(baseDir || '/');
  const root = normalized.endsWith('/') ? normalized : normalized + '/';
  g.location = { href: `file://${root}` };
}

function toPosixPath(path: string): string {
  return path.replace(/\\/g, '/');
}

function getPossibleFilenames(path: string) {
  const dir = dirname(path);
  const base = basename(path);
  const ext = extname(base);
  if (ext.length > 0) {
    return [join(dir, base), join(dir, '_' + base)];
  } else {
    return [
      join(dir, base + '.scss'),
      join(dir, base + '.sass'),
      join(dir, base + '.css'),
      join(dir, '_' + base + '.scss'),
      join(dir, '_' + base + '.sass'),
      join(dir, '_' + base + '.css'),
      join(dir, base, '_index.scss'),
      join(dir, base, 'index.scss'),
      join(dir, base, '_index.sass'),
      join(dir, base, 'index.sass'),
    ];
  }
}

function findFile(path: string): string {
  const possibleFilenames = getPossibleFilenames(path);
  for (const possibleFilename of possibleFilenames) {
    if (compilerFs().fileExists(possibleFilename)) {
      return possibleFilename;
    }
  }
  throw new Error('file not found: ' + path);
}

function loadContent(path: string): string {
  const filename = findFile(path);
  const content = compilerFs().readFile(filename);
  if (!content) {
    throw new Error(`File not found: ${filename}`);
  }
  return content;
}

const sassImporter = {
  canonicalize(urlString: string, context: CanonicalizeContext): URL | null {
    urlString = toPosixPath(urlString);
    if (!urlString.startsWith('file://')) {
      const location = sasslocation || '/';
      if (!isAbsolute(urlString)) {
        urlString = join(location, urlString);
      }
    } else {
      urlString = urlString.slice('file://'.length);
    }
    urlString = findFile(urlString);
    return new URL(`file://${urlString}`);
  },

  load(canonicalUrl: URL): ImporterResult | null {
    const path = canonicalUrl.path;
    const syntaxMap: Record<string, 'scss' | 'indented'> = {
      '.scss': 'scss',
      '.sass': 'indented',
    };
    const ext = extname(path);
    const contents = loadContent(path);

    return {
      contents: contents,
      syntax: syntaxMap[ext] || 'css',
    };
  },
};

export function renderSync(options: any): SassRenderResult {
  try {
    sasslocation = toPosixPath(options?.sasslocation || '/');
    ensureRuntimeLocation(sasslocation);
    const source: string = options.data;
    const sourceMap: boolean = options.sourceMap || false;
    const style: 'expanded' | 'compressed' = options.style || 'compressed';

    const result = compileString(source, {
      importer: sassImporter,
      sourceMap: sourceMap,
      style: style,
      logger: {
        warn(message: string) {
          if (typeof console !== 'undefined' && typeof console.warn === 'function') {
            console.warn(message);
          }
        },
        debug() {
          // no-op
        },
      },
    }) as CompileResult;
    return {
      css: result.css || '',
      map: result?.sourceMap ? JSON.stringify(result.sourceMap) : '',
      stats: {
        entry: '',
        start: 0,
        end: 0,
        duration: 0,
        includedFiles: [], // Placeholder for included files
      },
    };
  } catch (e) {
    throw e;
  } finally {
    sasslocation = undefined;
  }
}
