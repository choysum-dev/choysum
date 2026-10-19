// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stub for `@unovis/ts` (peer of `@unovis/vue`). */
export const Scale = {};

/** Minimal CurveType enum used by VisLine / VisArea. */
export const CurveType = {
  Linear: 'linear',
  MonotoneX: 'monotoneX',
};

export const GroupedBar = {
  selectors: { bar: '.bar', barGroup: '.barGroup' },
};
export const StackedBar = {
  selectors: { bar: '.bar', barGroup: '.barGroup' },
};
export const Line = {
  selectors: { line: '.line' },
};
export const Scatter = {
  selectors: { point: '.point', pointGroup: '.pointGroup', root: '.root' },
};
export const Donut = {
  selectors: { segment: '.segment', background: '.background' },
};

export default {
  Scale,
  CurveType,
  GroupedBar,
  StackedBar,
  Line,
  Scatter,
  Donut,
};
