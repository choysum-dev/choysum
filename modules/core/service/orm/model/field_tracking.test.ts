// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata';
import BaseModel from './model';
import {
  __setFieldTrackingActiveCompanyIdForTest,
  __setFieldTrackingAppendForTest,
  __setFieldTrackingDialForTest,
  recordFieldTrackingEvents,
  resolveTrackingCompanyField,
  serializeTrackedValue,
  __valuesEqualForTest,
} from './field_tracking';

function resetTrackingTestSeams(): void {
  __setFieldTrackingAppendForTest(undefined);
  __setFieldTrackingDialForTest(undefined);
  __setFieldTrackingActiveCompanyIdForTest(undefined);
}

test('recordFieldTrackingEvents appends field/create/unlink and skips untracked fields', async () => {
  resetTrackingTestSeams();
  @Model('TrackingProbe', { application: 'core', softDelete: false })
  class TrackingProbe extends BaseModel {
    @Field({ type: 'varchar', size: 64, tracking: true } as any)
    Name!: string;

    @Field({ type: 'varchar', size: 64 } as any)
    Note!: string;
  }

  const appended: any[] = [];
  __setFieldTrackingAppendForTest(async req => {
    appended.push(req);
    return req;
  });

  try {
    await recordFieldTrackingEvents({
      childCtor: TrackingProbe as any,
      operation: 'create',
      afterEntity: { Id: 'row1', Name: 'n1', Note: 'x' },
    });
    await recordFieldTrackingEvents({
      childCtor: TrackingProbe as any,
      operation: 'update',
      changedFields: ['Name', 'Note'],
      beforeEntity: { Id: 'row1', Name: 'n1', Note: 'x' },
      afterEntity: { Id: 'row1', Name: 'n2', Note: 'y' },
    });
    await recordFieldTrackingEvents({
      childCtor: TrackingProbe as any,
      operation: 'delete',
      beforeEntity: { Id: 'row1', Name: 'n2' },
    });

    expect(appended.map(r => r.Kind)).toEqual(['create', 'field', 'unlink']);
    expect(appended[1].Field).toBe('Name');
    expect(appended[1].OldValue).toBe('n1');
    expect(appended[1].NewValue).toBe('n2');
    expect(appended.filter(r => r.Field === 'Note').length).toBe(0);
    expect(MetadataStorage.instance.getModelMetadata(TrackingProbe as any).fields.get('Name')?.tracking).toBe(true);
  } finally {
    resetTrackingTestSeams();
  }
});

test('recordFieldTrackingEvents no-ops when model has no tracking fields', async () => {
  resetTrackingTestSeams();
  @Model('NoTrackingProbe', { application: 'core', softDelete: false })
  class NoTrackingProbe extends BaseModel {
    @Field({ type: 'varchar', size: 32 } as any)
    Name!: string;
  }

  let dialed = false;
  __setFieldTrackingAppendForTest(async () => {
    dialed = true;
    return {};
  });
  try {
    await recordFieldTrackingEvents({
      childCtor: NoTrackingProbe as any,
      operation: 'create',
      afterEntity: { Id: 'x', Name: 'n' },
    });
    expect(dialed).toBe(false);
  } finally {
    resetTrackingTestSeams();
  }
});

test('recordFieldTrackingEvents fails closed when Append is unavailable for tracked models', async () => {
  resetTrackingTestSeams();
  @Model('TrackingMissingAudit', { application: 'core', softDelete: false })
  class TrackingMissingAudit extends BaseModel {
    @Field({ type: 'varchar', size: 32, tracking: true } as any)
    Name!: string;
  }

  __setFieldTrackingAppendForTest(null);
  try {
    let err: unknown;
    try {
      await recordFieldTrackingEvents({
        childCtor: TrackingMissingAudit as any,
        operation: 'create',
        afterEntity: { Id: 'x', Name: 'n' },
      });
    } catch (e) {
      err = e;
    }
    expect(String((err as Error)?.message || '')).toMatch(/audit\.FieldChange is not available/);
  } finally {
    resetTrackingTestSeams();
  }
});

test('recordFieldTrackingEvents uses model companyField for CompanyId attribution', async () => {
  resetTrackingTestSeams();
  @Model('TrackingCompanyProbe', {
    application: 'core',
    softDelete: false,
    companyField: 'OwningCompanyId',
  })
  class TrackingCompanyProbe extends BaseModel {
    @Field({ type: 'char', size: 20 } as any)
    OwningCompanyId!: string;

    @Field({ type: 'varchar', size: 32, tracking: true } as any)
    Name!: string;
  }

  expect(resolveTrackingCompanyField(MetadataStorage.instance.getModelMetadata(TrackingCompanyProbe as any))).toBe(
    'OwningCompanyId'
  );
  expect(resolveTrackingCompanyField(undefined)).toBe('CompanyId');
  expect(resolveTrackingCompanyField({ companyField: '  ' })).toBe('CompanyId');

  const appended: any[] = [];
  __setFieldTrackingAppendForTest(async req => {
    appended.push(req);
    return req;
  });
  try {
    await recordFieldTrackingEvents({
      childCtor: TrackingCompanyProbe as any,
      operation: 'create',
      afterEntity: { Id: 'row1', Name: 'n1', OwningCompanyId: 'co_custom' },
    });
    expect(appended[0].CompanyId).toBe('co_custom');
  } finally {
    resetTrackingTestSeams();
  }
});

test('serializeTrackedValue covers primitives, dates, JSON undefined, and stringify failures', () => {
  expect(serializeTrackedValue(undefined)).toBeNull();
  expect(serializeTrackedValue(null)).toBeNull();
  expect(serializeTrackedValue(new Date('2024-01-02T03:04:05.000Z'))).toBe('2024-01-02T03:04:05.000Z');
  expect(serializeTrackedValue('s')).toBe('s');
  expect(serializeTrackedValue(12)).toBe('12');
  expect(serializeTrackedValue(true)).toBe('true');
  expect(serializeTrackedValue(1n)).toBe('1');
  expect(serializeTrackedValue({ toJSON: () => undefined })).toBeNull();
  const circular: any = {};
  circular.self = circular;
  expect(typeof serializeTrackedValue(circular)).toBe('string');

  expect(__valuesEqualForTest('x', 'x')).toBe(true);
  expect(__valuesEqualForTest(null, undefined)).toBe(true);
  expect(__valuesEqualForTest(undefined, null)).toBe(true);
  expect(__valuesEqualForTest(new Date('2024-01-01T00:00:00.000Z'), new Date('2024-01-01T00:00:00.000Z'))).toBe(true);
  expect(__valuesEqualForTest(new Date('2024-01-01T00:00:00.000Z'), new Date('2024-01-02T00:00:00.000Z'))).toBe(false);
  expect(__valuesEqualForTest({ a: 1 }, { a: 1 })).toBe(true);
  expect(__valuesEqualForTest('a', 'b')).toBe(false);
  expect(__valuesEqualForTest(new Date('2024-01-01T00:00:00.000Z'), 'nope')).toBe(false);
});

test('recordFieldTrackingEvents covers early exits, equal updates, relation skip, and company fallback', async () => {
  resetTrackingTestSeams();
  @Model('TrackingCoverageProbe', { application: 'core', softDelete: false })
  class TrackingCoverageProbe extends BaseModel {
    @Field({ type: 'varchar', size: 32, tracking: true } as any)
    Name!: string;

    @Field({ type: 'varchar', size: 32, tracking: true } as any)
    Stamp!: string;
  }

  const meta = MetadataStorage.instance.getModelMetadata(TrackingCoverageProbe as any);
  // Force a relation-like tracked field so isTrackableScalar rejects it at runtime.
  meta.fields.set('Children', { name: 'Children', type: 'OneToMany', tracking: true } as any);
  meta.fields.set('Tags', { name: 'Tags', type: 'ManyToMany', tracking: true } as any);
  meta.fields.set('Props', { name: 'Props', type: 'properties', tracking: true } as any);

  const appended: any[] = [];
  __setFieldTrackingAppendForTest(async req => {
    appended.push(req);
    return req;
  });

  try {
    await recordFieldTrackingEvents({ childCtor: null as any, operation: 'create', afterEntity: { Id: 'x' } });
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'create',
      afterEntity: { Name: 'n' },
    });
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'update',
      beforeEntity: { Id: 'row1', Name: 'same' },
      afterEntity: { Id: 'row1', Name: 'same' },
    });
    const t0 = new Date('2024-01-01T00:00:00.000Z');
    const t1 = new Date('2024-01-01T00:00:00.000Z');
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'update',
      changedFields: ['Stamp', 'Children', 'Tags', 'Props', 'Missing', 'Name'],
      beforeEntity: { Id: 'row1', Stamp: t0, Children: 'a', Tags: 'a', Props: 'a', Name: null },
      afterEntity: { Id: 'row1', Stamp: t1, Children: 'b', Tags: 'b', Props: 'b', Name: undefined },
    });
    expect(appended.length).toBe(0);

    // tracking:true with empty type still goes through isTrackableScalar type fallback.
    meta.fields.set('NoType', { name: 'NoType', type: '', tracking: true } as any);
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'update',
      changedFields: ['NoType'],
      beforeEntity: { Id: 'row1', NoType: 'a' },
      afterEntity: { Id: 'row1', NoType: 'b' },
    });
    expect(appended.length).toBe(1);
    appended.length = 0;

    __setFieldTrackingActiveCompanyIdForTest(() => 'co_active');
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'create',
      afterEntity: { Id: 'row2', Name: 'n', CompanyId: '   ' },
    });
    expect(appended[0].CompanyId).toBe('co_active');

    appended.length = 0;
    __setFieldTrackingActiveCompanyIdForTest(() => '');
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'create',
      afterEntity: { Id: 'row2b', Name: 'n' },
    });
    expect(appended[0].CompanyId).toBeNull();

    appended.length = 0;
    __setFieldTrackingActiveCompanyIdForTest(() => {
      throw new Error('no company context');
    });
    await recordFieldTrackingEvents({
      childCtor: TrackingCoverageProbe as any,
      operation: 'delete',
      beforeEntity: { Id: 'row3', Name: 'n' },
    });
    expect(appended[0].CompanyId).toBeNull();
  } finally {
    resetTrackingTestSeams();
  }
});

test('recordFieldTrackingEvents skips audit.FieldChange recursion and empty-field models', async () => {
  resetTrackingTestSeams();
  @Model('TrackingAuditSelfProbe', { application: 'core', softDelete: false })
  class TrackingAuditSelfProbe extends BaseModel {
    @Field({ type: 'varchar', size: 32, tracking: true } as any)
    Name!: string;
  }

  const selfMeta = MetadataStorage.instance.getModelMetadata(TrackingAuditSelfProbe as any);
  // Force the recursion guard name without depending on decorator name inference.
  (selfMeta as any).application = 'audit';
  (selfMeta as any).name = 'FieldChange';

  let dialed = false;
  __setFieldTrackingAppendForTest(async () => {
    dialed = true;
    return {};
  });
  try {
    await recordFieldTrackingEvents({
      childCtor: TrackingAuditSelfProbe as any,
      operation: 'create',
      afterEntity: { Id: 'x', Name: 'n' },
    });
    expect(dialed).toBe(false);

    @Model('EmptyFieldsProbe', { application: 'core', softDelete: false })
    class EmptyFieldsProbe extends BaseModel {}
    const emptyMeta = MetadataStorage.instance.getModelMetadata(EmptyFieldsProbe as any);
    (emptyMeta as any).fields = new Map();
    dialed = false;
    await recordFieldTrackingEvents({
      childCtor: EmptyFieldsProbe as any,
      operation: 'create',
      afterEntity: { Id: 'x' },
    });
    expect(dialed).toBe(false);

    // fullModelName empty → early return
    const Anon = class extends BaseModel {};
    const anonMeta = MetadataStorage.instance.getModelMetadata(Anon as any);
    if (anonMeta) {
      (anonMeta as any).application = '';
      (anonMeta as any).name = '';
    }
    Object.defineProperty(Anon, 'name', { value: '' });
    dialed = false;
    await recordFieldTrackingEvents({
      childCtor: Anon as any,
      operation: 'create',
      afterEntity: { Id: 'x' },
    });
    expect(dialed).toBe(false);
  } finally {
    resetTrackingTestSeams();
  }
});

test('resolveAppend uses dial override, missing Append, and dial errors', async () => {
  resetTrackingTestSeams();
  @Model('TrackingDialProbe', { application: 'core', softDelete: false })
  class TrackingDialProbe extends BaseModel {
    @Field({ type: 'varchar', size: 32, tracking: true } as any)
    Name!: string;
  }

  const appended: any[] = [];
  try {
    __setFieldTrackingAppendForTest(undefined);
    __setFieldTrackingDialForTest(() => ({
      Append: async (req: any) => {
        appended.push(req);
        return req;
      },
    }));
    await recordFieldTrackingEvents({
      childCtor: TrackingDialProbe as any,
      operation: 'create',
      afterEntity: { Id: 'dial1', Name: 'n' },
    });
    expect(appended.length).toBe(1);

    __setFieldTrackingDialForTest(() => ({} as any));
    let missingErr: unknown;
    try {
      await recordFieldTrackingEvents({
        childCtor: TrackingDialProbe as any,
        operation: 'create',
        afterEntity: { Id: 'dial2', Name: 'n' },
      });
    } catch (e) {
      missingErr = e;
    }
    expect(String((missingErr as Error)?.message || '')).toMatch(/not available/);

    __setFieldTrackingDialForTest(() => {
      throw new Error('dial boom');
    });
    let dialErr: unknown;
    try {
      await recordFieldTrackingEvents({
        childCtor: TrackingDialProbe as any,
        operation: 'create',
        afterEntity: { Id: 'dial3', Name: 'n' },
      });
    } catch (e) {
      dialErr = e;
    }
    expect(String((dialErr as Error)?.message || '')).toMatch(/not available/);

    // Dial returns undefined — fail-closed regardless of installed modules.
    __setFieldTrackingAppendForTest(undefined);
    __setFieldTrackingDialForTest(() => undefined as any);
    let missingSvcErr: unknown;
    try {
      await recordFieldTrackingEvents({
        childCtor: TrackingDialProbe as any,
        operation: 'create',
        afterEntity: { Id: 'dial4', Name: 'n' },
      });
    } catch (e) {
      missingSvcErr = e;
    }
    expect(String((missingSvcErr as Error)?.message || '')).toMatch(/not available/);

    // Clear dial override so resolveAppend uses the live `dial` fallback branch.
    __setFieldTrackingDialForTest(undefined);
    let liveDialErr: unknown;
    try {
      await recordFieldTrackingEvents({
        childCtor: TrackingDialProbe as any,
        operation: 'create',
        afterEntity: { Id: 'dial5', Name: 'n' },
      });
    } catch (e) {
      liveDialErr = e;
    }
    expect(String((liveDialErr as Error)?.message || '')).toMatch(/not available/);
  } finally {
    resetTrackingTestSeams();
  }
});
