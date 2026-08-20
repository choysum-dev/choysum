// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  AUDIT_FIELD_CHANGE_TIP_SOURCE,
  TOPIC_AUDIT_FIELD_CHANGE_APPENDED,
  __setAuditPublishTipForTest,
  publishFieldChangeAppendedTip,
  resolvePublishTip,
} from '../tips';

function withSeams(fn: () => void | Promise<void>): Promise<void> {
  __setAuditPublishTipForTest(undefined);
  return Promise.resolve()
    .then(() => fn())
    .finally(() => {
      __setAuditPublishTipForTest(undefined);
    });
}

test('audit.tips: resolvePublishTip uses live bus, override, and missing publish', async () => {
  await withSeams(() => {
    const root: any = (globalThis as any).$choysum ?? {};
    const priorBus = root.bus;
    try {
      root.bus = {};
      (globalThis as any).$choysum = root;
      expect(resolvePublishTip()).toBeNull();

      root.bus = { publish: 'nope' };
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

      __setAuditPublishTipForTest(null);
      expect(resolvePublishTip()).toBeNull();
    } finally {
      if (priorBus === undefined) delete root.bus;
      else root.bus = priorBus;
    }
  });
});

test('audit.tips: publishFieldChangeAppendedTip covers skip, timestamps, and best-effort catch', async () => {
  await withSeams(async () => {
    __setAuditPublishTipForTest(null);
    await publishFieldChangeAppendedTip({ Id: 'fc1', Model: 'partner.Partner', ResId: 'r1' });

    const published: any[] = [];
    __setAuditPublishTipForTest(event => {
      published.push(event);
    });
    await publishFieldChangeAppendedTip({ Id: '', Model: 'partner.Partner', ResId: 'r1' });
    await publishFieldChangeAppendedTip({ Id: 'fc1', Model: '', ResId: 'r1' });
    await publishFieldChangeAppendedTip({ Id: 'fc1', Model: 'partner.Partner', ResId: '' });
    expect(published).toHaveLength(0);

    const ts = Date.UTC(2024, 0, 2, 3, 4, 5);
    await publishFieldChangeAppendedTip({
      Id: 'fc1',
      Model: 'partner.Partner',
      ResId: 'r1',
      At: new Date(ts),
    });
    expect(published[0].topic).toBe(TOPIC_AUDIT_FIELD_CHANGE_APPENDED);
    expect(published[0].source).toBe(AUDIT_FIELD_CHANGE_TIP_SOURCE);
    expect(published[0].at).toBe(ts);
    expect(published[0].payload).toEqual({
      model: 'partner.Partner',
      resId: 'r1',
      fieldChangeId: 'fc1',
    });

    published.length = 0;
    await publishFieldChangeAppendedTip({
      Id: 'fc1',
      Model: 'partner.Partner',
      ResId: 'r1',
      At: ts + 1,
    });
    expect(published[0].at).toBe(ts + 1);

    published.length = 0;
    await publishFieldChangeAppendedTip({
      Id: 'fc1',
      Model: 'partner.Partner',
      ResId: 'r1',
      At: new Date(ts + 2).toISOString(),
    });
    expect(published[0].at).toBe(ts + 2);

    published.length = 0;
    await publishFieldChangeAppendedTip({
      Id: 'fc1',
      Model: 'partner.Partner',
      ResId: 'r1',
      At: new Date('not-a-date'),
    });
    expect(published[0].at).toBeUndefined();

    published.length = 0;
    __setAuditPublishTipForTest(() => {
      throw new Error('boom');
    });
    await publishFieldChangeAppendedTip({ Id: 'fc1', Model: 'partner.Partner', ResId: 'r1' });
  });
});
