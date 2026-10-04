// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, provide, ref } from 'vue';
import { mount, flushPromises } from '@choysum/test-utils';
import { PreferenceDraftSeeder, seedPreferenceDraft } from './PreferenceDraftSeeder';

test('seedPreferenceDraft: fills empty LanguageId and Timezone', () => {
  const draft: Record<string, unknown> = { LanguageId: '', Timezone: '' };
  seedPreferenceDraft(
    {
      getField: path => draft[path],
      setField: (path, value) => {
        draft[path] = value;
      },
    },
    { languageId: 'lang-1', timezone: 'Asia/Shanghai' },
  );
  expect(draft.LanguageId).toBe('lang-1');
  expect(draft.Timezone).toBe('Asia/Shanghai');
});

test('seedPreferenceDraft: does not overwrite persisted LanguageId / Timezone', () => {
  const draft: Record<string, unknown> = { LanguageId: 'saved-lang', Timezone: 'UTC' };
  seedPreferenceDraft(
    {
      getField: path => draft[path],
      setField: (path, value) => {
        draft[path] = value;
      },
    },
    { languageId: 'hint-lang', timezone: 'Asia/Shanghai' },
  );
  expect(draft.LanguageId).toBe('saved-lang');
  expect(draft.Timezone).toBe('UTC');
});

test('seedPreferenceDraft: no-ops without form-root', () => {
  seedPreferenceDraft(null, { languageId: 'lang-1', timezone: 'UTC' });
});

test('seedPreferenceDraft: treats empty ManyToOne LanguageId as empty and seeds Id', () => {
  const draft: Record<string, unknown> = { LanguageId: { Id: '' }, Timezone: null };
  seedPreferenceDraft(
    {
      getField: path => draft[path],
      setField: (path, value) => {
        draft[path] = value;
      },
    },
    { languageId: { Id: 'lang-2' }, timezone: 'UTC' },
  );
  expect(draft.LanguageId).toEqual({ Id: 'lang-2' });
  expect(draft.Timezone).toBe('UTC');
});

test('seedPreferenceDraft: non-empty non-Id field values are left alone', () => {
  const draft: Record<string, unknown> = { LanguageId: 1, Timezone: 2 };
  seedPreferenceDraft(
    {
      getField: path => draft[path],
      setField: (path, value) => {
        draft[path] = value;
      },
    },
    { languageId: 'lang-1', timezone: 'UTC' },
  );
  expect(draft.LanguageId).toBe(1);
  expect(draft.Timezone).toBe(2);
});

test('PreferenceDraftSeeder: injects form-root and seeds after ready + late hints', async () => {
  const draft: Record<string, unknown> = { LanguageId: '', Timezone: '' };
  const ready = ref(false);
  const languageId = ref('');
  const timezone = ref('');
  const Host = defineComponent({
    setup() {
      provide('form-root', {
        getField: (path: string) => draft[path],
        setField: (path: string, value: unknown) => {
          draft[path] = value;
        },
      });
      return () =>
        h(PreferenceDraftSeeder, {
          ready: ready.value,
          languageId: languageId.value,
          timezone: timezone.value,
        });
    },
  });
  const wrapper = mount(Host as any);
  await flushPromises();
  // Form ready before seeds arrive must not latch forever.
  ready.value = true;
  await flushPromises();
  expect(draft.LanguageId).toBe('');
  expect(draft.Timezone).toBe('');

  languageId.value = 'lang-late';
  await flushPromises();
  expect(draft.LanguageId).toBe('lang-late');
  expect(draft.Timezone).toBe('');

  timezone.value = 'Asia/Tokyo';
  await flushPromises();
  expect(draft.Timezone).toBe('Asia/Tokyo');
  wrapper.unmount();
});

test('PreferenceDraftSeeder: without form-root inject, seeding is a no-op', async () => {
  const Host = defineComponent({
    setup() {
      return () => h(PreferenceDraftSeeder, { ready: true, languageId: 'lang-1', timezone: 'UTC' });
    },
  });
  const wrapper = mount(Host as any);
  await flushPromises();
  wrapper.unmount();
});

test('PreferenceDraftSeeder: does not overwrite server LanguageId when seed arrives late', async () => {
  const draft: Record<string, unknown> = { LanguageId: 'saved-lang', Timezone: 'UTC' };
  const languageId = ref('');
  const Host = defineComponent({
    setup() {
      provide('form-root', {
        getField: (path: string) => draft[path],
        setField: (path: string, value: unknown) => {
          draft[path] = value;
        },
      });
      return () => h(PreferenceDraftSeeder, { ready: true, languageId: languageId.value, timezone: '' });
    },
  });
  const wrapper = mount(Host as any);
  languageId.value = 'hint-lang';
  await flushPromises();
  expect(draft.LanguageId).toBe('saved-lang');
  wrapper.unmount();
});
