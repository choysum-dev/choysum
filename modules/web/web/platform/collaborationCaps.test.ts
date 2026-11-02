// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import {
  CollaborationCapsKey,
  resolveCollaborationCaps,
  useCollaborationCaps,
  useReactiveCollaborationCaps,
} from './collaborationCaps';
import { replaceStoreFactory } from '@/core/web/stores/registry';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';

const MESSAGE_MODELS = ['message.Message', 'message.Notification', 'message.Follower'] as const;

test('resolveCollaborationCaps: empty registry → all false', () => {
  expect(resolveCollaborationCaps([])).toEqual({
    message: false,
    audit: false,
    document: false,
  });
});

test('resolveCollaborationCaps: message models alone → message true', () => {
  expect(
    resolveCollaborationCaps([
      'message.Message',
      'message.Notification',
      'message.Follower',
      'other.Model',
    ])
  ).toEqual({
    message: true,
    audit: false,
    document: false,
  });
});

test('resolveCollaborationCaps: partial message models → message false', () => {
  expect(resolveCollaborationCaps(['message.Message', 'message.Notification'])).toEqual({
    message: false,
    audit: false,
    document: false,
  });
});

test('resolveCollaborationCaps: full set → all true', () => {
  expect(
    resolveCollaborationCaps([
      'message.Message',
      'message.Notification',
      'message.Follower',
      'audit.FieldChange',
      'document.AttachmentBinding',
    ])
  ).toEqual({
    message: true,
    audit: true,
    document: true,
  });
});

test('useCollaborationCaps / useReactiveCollaborationCaps: registry fallback when no provide', async () => {
  const Probe = defineComponent({
    name: 'CapsProbe',
    setup() {
      const snap = useCollaborationCaps();
      const reactiveCaps = useReactiveCollaborationCaps();
      return () =>
        h('div', {
          'data-snap': JSON.stringify(snap),
          'data-reactive': JSON.stringify(reactiveCaps.value),
        });
    },
  });
  const restores: Array<() => void> = [];
  try {
    const mounted = mountApp(Probe as any);
    await flushPromises();
    // Empty live registry → all false (collaborationCaps uses the real registry).
    expect(JSON.parse(String(mounted.q('[data-snap]')?.getAttribute('data-snap')))).toEqual({
      message: false,
      audit: false,
      document: false,
    });
    expect(JSON.parse(String(mounted.q('[data-reactive]')?.getAttribute('data-reactive')))).toEqual({
      message: false,
      audit: false,
      document: false,
    });

    for (const model of MESSAGE_MODELS) {
      restores.push(replaceStoreFactory(model, () => ({}) as any));
    }
    await nextTick();
    await flushPromises();
    // Snapshot is fixed at setup; reactive caps track registry version.
    expect(JSON.parse(String(mounted.q('[data-snap]')?.getAttribute('data-snap')))).toEqual({
      message: false,
      audit: false,
      document: false,
    });
    expect(JSON.parse(String(mounted.q('[data-reactive]')?.getAttribute('data-reactive')))).toEqual({
      message: true,
      audit: false,
      document: false,
    });
    mounted.unmount();
  } finally {
    while (restores.length) restores.pop()?.();
  }
});

test('useCollaborationCaps / useReactiveCollaborationCaps: honor CollaborationCapsKey override', async () => {
  const override = { message: true, audit: false, document: true };
  const Probe = defineComponent({
    name: 'CapsOverrideProbe',
    setup() {
      const snap = useCollaborationCaps();
      const reactiveCaps = useReactiveCollaborationCaps();
      return () =>
        h('div', {
          'data-snap': JSON.stringify(snap),
          'data-reactive': JSON.stringify(reactiveCaps.value),
        });
    },
  });
  const mounted = mountApp(Probe as any, {
    provide: { [CollaborationCapsKey]: override },
  });
  await flushPromises();
  expect(JSON.parse(String(mounted.q('[data-snap]')?.getAttribute('data-snap')))).toEqual(override);
  expect(JSON.parse(String(mounted.q('[data-reactive]')?.getAttribute('data-reactive')))).toEqual(override);
  mounted.unmount();
});
