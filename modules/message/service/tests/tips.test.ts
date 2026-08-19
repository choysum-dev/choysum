// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  MESSAGE_NOTIFICATION_TIP_SOURCE,
  MESSAGE_POST_TIP_SOURCE,
  TOPIC_MESSAGE_NOTIFICATION_USER,
  TOPIC_MESSAGE_THREAD_CHANGED,
  __setMessagePublishTipForTest,
  publishNotificationUserTip,
  publishNotificationUserTips,
  publishThreadChangedTip,
  resolvePublishTip,
} from '../tips';

test('message.tips: resolvePublishTip uses live bus, override, and missing publish', () => {
  __setMessagePublishTipForTest(undefined);
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

    __setMessagePublishTipForTest(null);
    expect(resolvePublishTip()).toBeNull();
  } finally {
    if (priorBus === undefined) delete root.bus;
    else root.bus = priorBus;
    __setMessagePublishTipForTest(undefined);
  }
});

test('message.tips: publishThreadChangedTip covers skip, timestamps, and best-effort catch', async () => {
  __setMessagePublishTipForTest(null);
  await publishThreadChangedTip({ Id: 'm1', Model: 'partner.Partner', ResId: 'r1' });

  const published: any[] = [];
  __setMessagePublishTipForTest(event => {
    published.push(event);
  });
  await publishThreadChangedTip({ Id: '', Model: 'partner.Partner', ResId: 'r1' });
  await publishThreadChangedTip({ Id: 'm1', Model: '', ResId: 'r1' });
  await publishThreadChangedTip({ Id: 'm1', Model: 'partner.Partner', ResId: '' });
  expect(published).toHaveLength(0);

  const ts = Date.UTC(2024, 0, 2, 3, 4, 5);
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: new Date(ts),
  });
  expect(published[0].topic).toBe(TOPIC_MESSAGE_THREAD_CHANGED);
  expect(published[0].source).toBe(MESSAGE_POST_TIP_SOURCE);
  expect(published[0].at).toBe(ts);

  published.length = 0;
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: ts + 1,
  });
  expect(published[0].at).toBe(ts + 1);

  published.length = 0;
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: new Date(ts + 2).toISOString(),
  });
  expect(published[0].at).toBe(ts + 2);

  published.length = 0;
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: new Date('not-a-date'),
  });
  expect(published[0].at).toBeUndefined();

  published.length = 0;
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: Number.NaN,
  });
  expect(published[0].at).toBeUndefined();

  published.length = 0;
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: '   ',
  });
  expect(published[0].at).toBeUndefined();

  published.length = 0;
  await publishThreadChangedTip({
    Id: 'm1',
    Model: 'partner.Partner',
    ResId: 'r1',
    CreatedAt: 'not-parseable',
  });
  expect(published[0].at).toBeUndefined();

  __setMessagePublishTipForTest(() => {
    throw new Error('bus down');
  });
  await publishThreadChangedTip({ Id: 'm1', Model: 'partner.Partner', ResId: 'r1' });
  __setMessagePublishTipForTest(undefined);
});

test('message.tips: publishNotificationUserTip skips and swallows errors', async () => {
  __setMessagePublishTipForTest(null);
  await publishNotificationUserTip('usr_a');

  const published: any[] = [];
  __setMessagePublishTipForTest(event => {
    published.push(event);
  });
  await publishNotificationUserTip('   ');
  await publishNotificationUserTip('');
  expect(published).toHaveLength(0);

  const ts = Date.UTC(2025, 5, 1);
  await publishNotificationUserTip('usr_a', ts);
  expect(published[0].topic).toBe(TOPIC_MESSAGE_NOTIFICATION_USER);
  expect(published[0].source).toBe(MESSAGE_NOTIFICATION_TIP_SOURCE);
  expect(published[0].payload.userId).toBe('usr_a');
  expect(published[0].at).toBe(ts);

  __setMessagePublishTipForTest(() => {
    throw 'bus string boom';
  });
  await publishNotificationUserTip('usr_a', new Date(ts));
  __setMessagePublishTipForTest(undefined);
});

test('message.tips: publishNotificationUserTips dedupes blank and duplicate user ids', async () => {
  const published: any[] = [];
  __setMessagePublishTipForTest(event => {
    published.push(event);
  });
  await publishNotificationUserTips(['usr_a', '', '  ', 'usr_a', 'usr_b'], '2024-01-15T00:00:00.000Z');
  expect(published).toHaveLength(2);
  expect(published.map(item => item.payload.userId).sort()).toEqual(['usr_a', 'usr_b']);
  expect(typeof published[0].at).toBe('number');
  __setMessagePublishTipForTest(undefined);
});
