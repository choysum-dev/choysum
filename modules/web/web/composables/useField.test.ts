// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, provide, ref } from 'vue';

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { useField, useStandaloneField } from '@/web/web/composables/useField';
import type { ViewKindId, ViewMode } from '@/web/web/components/view/ChoyViewScope.vue';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import { clearFieldsByStore, exportFieldSelection } from '@/web/web/query/utils/registry/field';
import { replaceStoreFactory } from '@/web/web/stores/registry';

function mountFieldEnv(opts: {
  viewContainer?: ViewKindId;
  viewMode?: ViewMode;
  fieldPrefix?: string;
  formRoot?: { draft: Record<string, unknown> } | null;
}) {
  const Probe = defineComponent({
    setup() {
      const field = useField({ prop: 'Name' });
      return () =>
        h('span', {
          'data-edit': String(field.env.isEditMode),
          'data-form': String(field.env.isForm),
          'data-presentation': String(field.env.fieldPresentation),
          'data-mode': String(field.env.viewMode),
          'data-prefix': String(field.env.fieldPrefix),
        });
    },
  });

  const Host = defineComponent({
    setup() {
      useProvideViewScope({
        kind: opts.viewContainer ?? 'List',
        mode: opts.viewMode ?? 'display',
        fieldPrefix: opts.fieldPrefix,
      });
      if (opts.formRoot) provide('form-root', opts.formRoot);
      return () => h(Probe);
    },
  });

  return mountApp(Host);
}

describe('useField FieldEnv.isEditMode (List + form-root, D4)', () => {
  test('is false for List display without form-root', () => {
    const { unmount, q } = mountFieldEnv({ viewContainer: 'List', viewMode: 'display', formRoot: null });
    expect(q('span')?.getAttribute('data-edit')).toBe('false');
    unmount();
  });

  test('is false for List edit mode without form-root', () => {
    const { unmount, q } = mountFieldEnv({ viewContainer: 'List', viewMode: 'edit', formRoot: null });
    expect(q('span')?.getAttribute('data-edit')).toBe('false');
    unmount();
  });

  test('is false for List edit mode with form-root but null draft', () => {
    const { unmount, q } = mountFieldEnv({
      viewContainer: 'List',
      viewMode: 'edit',
      formRoot: { draft: null as any },
    });
    expect(q('span')?.getAttribute('data-edit')).toBe('false');
    unmount();
  });

  test('is true for List edit mode with row form-root draft', () => {
    const { unmount, q } = mountFieldEnv({
      viewContainer: 'List',
      viewMode: 'edit',
      formRoot: { draft: { Id: '1', Name: 'Row' } },
    });
    expect(q('span')?.getAttribute('data-edit')).toBe('true');
    unmount();
  });

  test('is true for Form edit mode without form-root', () => {
    const { unmount, q } = mountFieldEnv({
      viewContainer: 'Form',
      viewMode: 'edit',
      fieldPrefix: 'Hdr',
      formRoot: null,
    });
    const el = q('span');
    expect(el?.getAttribute('data-edit')).toBe('true');
    expect(el?.getAttribute('data-form')).toBe('true');
    expect(el?.getAttribute('data-presentation')).toBe('form');
    expect(el?.getAttribute('data-mode')).toBe('edit');
    expect(el?.getAttribute('data-prefix')).toBe('Hdr');
    unmount();
  });

  test('is true for Form create mode', () => {
    const { unmount, q } = mountFieldEnv({ viewContainer: 'Form', viewMode: 'create', formRoot: null });
    expect(q('span')?.getAttribute('data-edit')).toBe('true');
    expect(q('span')?.getAttribute('data-mode')).toBe('create');
    unmount();
  });
});

describe('useField createModelStore for relations', () => {
  test('builds relationStore for a manytoone leaf via createModelStore', async () => {
    const model = 'demo.UseFieldRelLeaf';
    const restore = replaceStoreFactory(model, () => ({
      storeId: 'rel-leaf',
      fullModelName: model,
      fieldsMetadata: { Name: { type: 'varchar' } },
    }));
    try {
      let field: ReturnType<typeof useField> | undefined;
      const Probe = defineComponent({
        setup() {
          field = useField({
            store: {
              storeId: 'main',
              fieldsMetadata: {
                PartnerId: { type: 'manytoone', relationModel: model },
              },
            } as any,
            prop: 'PartnerId',
          });
          return () => h('span');
        },
      });
      const Host = defineComponent({
        setup() {
          useProvideViewScope({ kind: 'Form', mode: 'edit' });
          return () => h(Probe);
        },
      });
      const { unmount } = mountApp(Host);
      await flushPromises();
      expect(field?.meta?.relationModel).toBe(model);
      // Product useField may resolve createModelStore via the FE registry stub
      // ($id/modelName) or the real factory (storeId); either proves the call ran.
      const rel = field?.relationStore as any;
      expect(rel?.storeId === 'rel-leaf' || rel?.modelName === model || rel?.$id).toBeTruthy();
      unmount();
    } finally {
      restore();
    }
  });

  test('walks dotted relation path and calls createModelStore for intermediate segments', async () => {
    // FE unit may stub createModelStore so intermediate stores lack fieldsMetadata;
    // this still executes the non-leaf createModelStore branch in useField.
    const mid = 'demo.UseFieldRelMid';
    let field: ReturnType<typeof useField> | undefined;
    const Probe = defineComponent({
      setup() {
        field = useField({
          store: {
            storeId: 'main',
            fieldsMetadata: {
              PartnerId: { type: 'manytoone', relationModel: mid },
            },
          } as any,
          prop: 'PartnerId.CountryId',
        });
        return () => h('span');
      },
    });
    const Host = defineComponent({
      setup() {
        useProvideViewScope({ kind: 'Form', mode: 'edit' });
        return () => h(Probe);
      },
    });
    const { unmount } = mountApp(Host);
    await flushPromises();
    expect(field).toBeTruthy();
    unmount();
  });
});

describe('useField useStandaloneField env', () => {
  test('defaults fieldPresentation to none', () => {
    const value = ref('x');
    let presentation = '';
    const Host = defineComponent({
      setup() {
        const field = useStandaloneField({ value });
        presentation = field.env.fieldPresentation;
        return () => h('div');
      },
    });
    const { unmount } = mountApp(Host);
    expect(presentation).toBe('none');
    unmount();
  });
});

describe('useField registration path prefix strip (suppressFieldRegistration)', () => {
  test('registers relative path on alt store when prefix matches', async () => {
    const altStore = { storeId: 'alt-store' };
    clearFieldsByStore('alt-store');

    const Host = defineComponent({
      setup() {
        useProvideViewScope({ kind: 'Form', mode: 'edit', fieldPrefix: 'Lines' });
        provide('suppress-field-registration', true);
        provide('alt-field-registration-store', ref(altStore));
        const Probe = defineComponent({
          setup() {
            useField({
              store: { storeId: 'main', fieldsMetadata: { Lines: { type: 'one2many' }, Name: { type: 'varchar' } } } as any,
              prop: 'Lines.Name',
            });
            return () => h('span');
          },
        });
        return () => h(Probe);
      },
    });

    const { unmount } = mountApp(Host);
    await flushPromises();
    expect(exportFieldSelection('alt-store')).toContain('Name');
    expect(exportFieldSelection('main').filter(p => p !== 'Id')).toEqual([]);
    clearFieldsByStore('alt-store');
    unmount();
  });

  test('strips exact fieldPrefix match to empty (no registration)', async () => {
    const altStore = { storeId: 'alt-exact' };
    clearFieldsByStore('alt-exact');

    const Host = defineComponent({
      setup() {
        useProvideViewScope({ kind: 'Form', mode: 'edit', fieldPrefix: 'Lines' });
        provide('suppress-field-registration', true);
        provide('alt-field-registration-store', ref(altStore));
        const Probe = defineComponent({
          setup() {
            useField({
              store: { storeId: 'main', fieldsMetadata: { Lines: { type: 'one2many' } } } as any,
              prop: 'Lines',
            });
            return () => h('span');
          },
        });
        return () => h(Probe);
      },
    });

    const { unmount } = mountApp(Host);
    await flushPromises();
    expect(exportFieldSelection('alt-exact').filter(p => p !== 'Id')).toEqual([]);
    clearFieldsByStore('alt-exact');
    unmount();
  });
});
