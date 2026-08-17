// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata';
import BaseModel from './model';
import { __setFieldTrackingAppendForTest, recordFieldTrackingEvents } from './field_tracking';

test('recordFieldTrackingEvents appends field/create/unlink and skips untracked fields', async () => {
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
  } finally {
    __setFieldTrackingAppendForTest(undefined);
  }

  expect(MetadataStorage.instance.getModelMetadata(TrackingProbe as any).fields.get('Name')?.tracking).toBe(true);
});

test('recordFieldTrackingEvents no-ops when model has no tracking fields', async () => {
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
    __setFieldTrackingAppendForTest(undefined);
  }
});

test('recordFieldTrackingEvents fails closed when Append is unavailable for tracked models', async () => {
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
    __setFieldTrackingAppendForTest(undefined);
  }
});
