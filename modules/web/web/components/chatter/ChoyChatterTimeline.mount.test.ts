// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount coverage for W3 Chatter chrome: Empty/Spinner, Message/Bubble items,
 * and MessageScroller when the timeline has entries.
 */
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyChatterFieldChangeItem from './ChoyChatterFieldChangeItem.vue';
import ChoyChatterMessageItem from './ChoyChatterMessageItem.vue';
import ChoyChatterTimeline from './ChoyChatterTimeline.vue';

describe('ChoyChatterTimeline mount', () => {
  test('loading shows spinner; empty uses ChoyEmpty description', async () => {
    const loading = mountApp(ChoyChatterTimeline as any, {
      props: { loading: true, entries: [], loadingLabel: 'Loading activity...' },
    });
    await flushPromises();
    expect(loading.q('[data-testid=choy-spinner]')).not.toBeNull();
    loading.unmount();

    const empty = mountApp(ChoyChatterTimeline as any, {
      props: { entries: [], emptyLabel: 'No activity yet' },
    });
    await flushPromises();
    expect(empty.q('[data-testid=choy-empty]')).not.toBeNull();
    expect(empty.text()).toContain('No activity yet');
    empty.unmount();
  });

  test('error branch shows alert text and skips empty/spinner chrome', async () => {
    const mounted = mountApp(ChoyChatterTimeline as any, {
      props: { error: 'Failed to load activity', entries: [], loading: false },
    });
    await flushPromises();
    const alert = mounted.q('[role=alert]');
    expect(alert).not.toBeNull();
    expect(alert?.textContent || '').toContain('Failed to load activity');
    expect(mounted.q('[data-testid=choy-spinner]')).toBeNull();
    expect(mounted.q('[data-testid=choy-empty]')).toBeNull();
    expect(mounted.q('[data-slot=message-scroller]')).toBeNull();
    mounted.unmount();
  });

  test('message and field-change items render Message/Bubble anchors and initials', async () => {
    const msg = mountApp(ChoyChatterMessageItem as any, {
      props: {
        authorLabel: 'Ada Lovelace',
        entry: {
          kind: 'message',
          id: 'm1',
          at: Date.UTC(2024, 0, 2, 12, 0, 0),
          type: 'comment',
          body: 'Hello chatter',
          authorUid: 'u1',
        },
      },
    });
    await flushPromises();
    expect(msg.q('[data-anchor="choy.chatter.message"]')).not.toBeNull();
    expect(msg.q('[data-slot=message]')).not.toBeNull();
    expect(msg.q('[data-slot=bubble]')).not.toBeNull();
    expect(msg.text()).toContain('AL');
    expect(msg.text()).toContain('Ada Lovelace');
    expect(msg.text()).toContain('Hello chatter');
    msg.unmount();

    const change = mountApp(ChoyChatterFieldChangeItem as any, {
      props: {
        authorLabel: 'Bob',
        entry: {
          kind: 'fieldChange',
          id: 'fc1',
          at: Date.UTC(2024, 0, 2, 13, 0, 0),
          field: 'Name',
          changeKind: 'update',
          oldValue: 'A',
          newValue: 'B',
          actorUid: 'u2',
        },
      },
    });
    await flushPromises();
    expect(change.q('[data-anchor="choy.chatter.field-change"]')).not.toBeNull();
    expect(change.q('[data-slot=bubble]')).not.toBeNull();
    expect(change.text()).toContain('BO');
    expect(change.text()).toContain('Name');
    change.unmount();
  });

  test('populated timeline mounts MessageScroller items for each entry', async () => {
    const mounted = mountApp(ChoyChatterTimeline as any, {
      props: {
        entries: [
          {
            kind: 'message',
            id: 'm1',
            at: Date.UTC(2024, 0, 2, 12, 0, 0),
            type: 'comment',
            body: 'First',
            authorUid: 'u1',
          },
          {
            kind: 'fieldChange',
            id: 'fc1',
            at: Date.UTC(2024, 0, 2, 13, 0, 0),
            field: 'State',
            changeKind: 'update',
            oldValue: 'draft',
            newValue: 'done',
            actorUid: 'u2',
          },
        ],
        resolveAuthorLabel: (id: string | null | undefined) =>
          id === 'u1' ? 'Ann' : 'Ben',
      },
    });
    await flushPromises();
    expect(mounted.q('[data-anchor="choy.chatter.timeline"]')).not.toBeNull();
    expect(mounted.q('[data-slot=message-scroller]')).not.toBeNull();
    expect(mounted.q('[data-anchor="choy.chatter.message"]')).not.toBeNull();
    expect(mounted.q('[data-anchor="choy.chatter.field-change"]')).not.toBeNull();
    expect(mounted.text()).toContain('First');
    expect(mounted.text()).toContain('State');
    mounted.unmount();
  });
});
