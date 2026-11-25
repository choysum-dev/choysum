// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ImportPanel from './ImportPanel.vue';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';

type MappingSetup = {
  mappingRows: Array<{ header: string; fieldPath: string }>;
  mappingFormStore: UiModelStore | null;
  catalogFields: Array<{ path?: string; label?: string; children?: unknown[] }>;
  sanitizeHeaderFieldName: (header: string, used: Set<string>) => string;
  rebuildMappingFormStore: () => void;
  syncMappingRowsFromStore: () => void;
  mappingPropForHeader: (header: string) => string;
  onFooterCancel: () => Promise<void>;
  modelValue: boolean;
};

function mountPanel() {
  return mountApp(ImportPanel as any, {
    props: {
      model: 'partner.Partner',
      companyId: 'c1',
      modelValue: true,
    },
  });
}

function asSetup(m: ReturnType<typeof mountPanel>): MappingSetup {
  return m.setupState() as MappingSetup;
}

describe('ImportPanel mapping FormView store', () => {
  test('sanitizeHeaderFieldName handles empty, digits, and collisions', () => {
    const m = mountPanel();
    const s = asSetup(m);
    const used = new Set<string>();
    expect(s.sanitizeHeaderFieldName('  Name!  ', used)).toBe('Name');
    expect(s.sanitizeHeaderFieldName('', used)).toBe('column');
    expect(s.sanitizeHeaderFieldName('123', used)).toBe('c_123');
    expect(s.sanitizeHeaderFieldName('Name', used)).toBe('Name_2');
    m.unmount();
  });

  test('rebuildMappingFormStore builds selection fields and sync round-trips', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m);
    s.catalogFields.splice(0, s.catalogFields.length, { path: 'Name', label: 'Name' }, { path: 'Email', label: 'Email' });
    s.mappingRows.splice(
      0,
      s.mappingRows.length,
      { header: 'Full Name', fieldPath: '' },
      { header: 'Email', fieldPath: 'Email' },
    );
    s.rebuildMappingFormStore();
    expect(s.mappingFormStore).toBeTruthy();
    expect(s.mappingPropForHeader('Full Name')).toBe('Full_Name');
    expect(s.mappingPropForHeader('Email')).toBe('Email');
    expect(s.mappingFormStore!.getField('Email')).toBe('Email');

    s.mappingFormStore!.setField('Full_Name', 'Name');
    s.syncMappingRowsFromStore();
    expect(s.mappingRows[0]!.fieldPath).toBe('Name');

    s.mappingRows.splice(0, s.mappingRows.length);
    s.rebuildMappingFormStore();
    expect(s.mappingFormStore).toBeNull();
    m.unmount();
  });

  test('syncMappingRowsFromStore no-ops without store', () => {
    const m = mountPanel();
    const s = asSetup(m);
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'A', fieldPath: 'x' });
    // Ensure store cleared.
    s.mappingRows.splice(0, s.mappingRows.length);
    s.rebuildMappingFormStore();
    s.mappingRows.splice(0, s.mappingRows.length, { header: 'A', fieldPath: 'x' });
    s.syncMappingRowsFromStore();
    expect(s.mappingRows[0]!.fieldPath).toBe('x');
    m.unmount();
  });

  test('onFooterCancel closes via defineModel', async () => {
    const m = mountPanel();
    await flushPromises();
    const s = asSetup(m) as MappingSetup & { visible?: boolean };
    await s.onFooterCancel();
    // setupState unwraps defineModel; host may expose `visible` as false after close.
    expect(s.visible === false || s.modelValue === false).toBe(true);
    m.unmount();
  });
});
