// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as repositoryBarrel from './index';

test('orm/repository barrel is runtime-only (SF-1 / SF-D)', () => {
  expect(Object.keys(repositoryBarrel).sort()).toEqual(['Repository', 'RepositoryFactory', 'db']);
});
