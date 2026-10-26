// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { formatTokenRevokedChartLabel } from './token_chart_labels';

test('formatTokenRevokedChartLabel maps boolean and field=value keys', () => {
  const labels = { revoked: 'Revoked', notRevoked: 'Not Revoked' };
  expect(formatTokenRevokedChartLabel('true', labels)).toBe('Revoked');
  expect(formatTokenRevokedChartLabel('false', labels)).toBe('Not Revoked');
  expect(formatTokenRevokedChartLabel('Revoked=true', labels)).toBe('Revoked');
  expect(formatTokenRevokedChartLabel('Revoked=false', labels)).toBe('Not Revoked');
  expect(formatTokenRevokedChartLabel('access', labels)).toBe('access');
});
