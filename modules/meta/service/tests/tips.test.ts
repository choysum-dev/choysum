// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  META_MODULE_OP_TIP_MODEL,
  META_MODULE_OP_TIP_SOURCE,
  TOPIC_META_MODULE_OP_CHANGED,
  __setMetaPublishTipForTest,
  publishModuleOpChangedTip,
  resolvePublishTip,
} from '../tips';

test('meta.tips: resolvePublishTip uses live bus, override, and missing publish', () => {
  __setMetaPublishTipForTest(undefined);
  const root: any = (globalThis as any).$choysum ?? {};
  const priorBus = root.bus;
  try {
    root.bus = {};
    (globalThis as any).$choysum = root;
    expect(resolvePublishTip()).toBeNull();

    const published: any[] = [];
    root.bus = {
      publish(event: unknown) {
        published.push(event);
      },
    };
    const live = resolvePublishTip();
    expect(typeof live).toBe('function');
    live!({ topic: 't', source: 's' });
    expect(published).toHaveLength(1);

    __setMetaPublishTipForTest(null);
    expect(resolvePublishTip()).toBeNull();
  } finally {
    if (priorBus === undefined) delete root.bus;
    else root.bus = priorBus;
    __setMetaPublishTipForTest(undefined);
  }
});

test('meta.tips: publishModuleOpChangedTip skips, publishes, and swallows errors', async () => {
  try {
    __setMetaPublishTipForTest(null);
    await publishModuleOpChangedTip({ jobId: 'job-1', userId: 'u1' });

    const published: any[] = [];
    __setMetaPublishTipForTest(event => {
      published.push(event);
    });
    await publishModuleOpChangedTip({ jobId: '', userId: 'u1' });
    await publishModuleOpChangedTip({ jobId: '  ', userId: 'u1' });
    expect(published).toHaveLength(0);

    await publishModuleOpChangedTip({ jobId: 'job-1', userId: 'u1' });
    expect(published).toHaveLength(1);
    expect(published[0].topic).toBe(TOPIC_META_MODULE_OP_CHANGED);
    expect(published[0].source).toBe(META_MODULE_OP_TIP_SOURCE);
    expect(published[0].payload).toEqual({
      model: META_MODULE_OP_TIP_MODEL,
      resId: 'job-1',
      jobId: 'job-1',
      userId: 'u1',
    });

    published.length = 0;
    await publishModuleOpChangedTip({ jobId: 'job-2', source: 'meta.ExecuteInstall' });
    expect(published[0].source).toBe('meta.ExecuteInstall');
    expect(published[0].payload.userId).toBe(undefined);

    published.length = 0;
    await publishModuleOpChangedTip({ jobId: 'job-whitespace-source', source: '   ' });
    expect(published[0].source).toBe(META_MODULE_OP_TIP_SOURCE);

    __setMetaPublishTipForTest(() => {
      throw new Error('bus down');
    });
    await publishModuleOpChangedTip({ jobId: 'job-3', userId: 'u1' });
    expect(published).toHaveLength(1);
  } finally {
    __setMetaPublishTipForTest(undefined);
  }
});
