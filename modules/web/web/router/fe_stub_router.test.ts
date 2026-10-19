// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';

const createFeStubRouter = (VueRouter as any).createFeStubRouter as (overrides?: object) => {
  router: {
    push: (to: unknown) => Promise<unknown>;
    beforeEach: (fn: unknown) => () => void;
    afterEach: (fn: unknown) => () => void;
  };
};

/** Exercises FE stub beforeEach/afterEach register, fire, non-function, and unsubscribe paths. */
test('createFeStubRouter beforeEach/afterEach hooks register, run, and unsubscribe', async () => {
  const { router } = createFeStubRouter({ route: { path: '/', fullPath: '/' } });

  const ignoreBefore = router.beforeEach(null);
  const ignoreAfter = router.afterEach('not-a-function');
  expect(typeof ignoreBefore).toBe('function');
  expect(typeof ignoreAfter).toBe('function');
  ignoreBefore();
  ignoreAfter();

  const beforeCalls: Array<{ to: string; from: string }> = [];
  const afterCalls: Array<{ to: string; from: string }> = [];
  const unsubBefore = router.beforeEach((to: { path: string }, from: { path: string }) => {
    beforeCalls.push({ to: to.path, from: from.path });
  });
  const unsubAfter = router.afterEach((to: { path: string }, from: { path: string }) => {
    afterCalls.push({ to: to.path, from: from.path });
  });

  await router.push('/a');
  expect(beforeCalls).toEqual([{ to: '/a', from: '/' }]);
  expect(afterCalls).toEqual([{ to: '/a', from: '/' }]);

  unsubBefore();
  unsubAfter();
  // Second unsubscribe is a no-op when the hook is already gone.
  unsubBefore();
  unsubAfter();

  await router.push('/b');
  expect(beforeCalls).toHaveLength(1);
  expect(afterCalls).toHaveLength(1);
});
