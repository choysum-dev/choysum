// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pin a desktop or mobile viewport for layoutStore / VueUse breakpoints.
 * Handles compound `(min-width) and (max-width)` queries.
 */
export function pinViewportWidth(width: number): void {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    writable: true,
    value: width,
  });
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    value: (query: string) => {
      const parts = String(query)
        .replace(/[()]/g, '')
        .split(/\s+and\s+/i)
        .map((p) => p.trim())
        .filter(Boolean);
      const matches =
        parts.length === 0
          ? false
          : parts.every((part) => {
              const min = /min-width:\s*(\d+)px/i.exec(part);
              const max = /max-width:\s*(\d+)px/i.exec(part);
              if (min && width < Number(min[1])) return false;
              if (max && width > Number(max[1])) return false;
              return !!(min || max);
            });
      return {
        matches,
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      };
    },
  });
}
