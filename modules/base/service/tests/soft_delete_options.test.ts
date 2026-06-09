// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';

import UoMCategory from '@/base/service/models/uom_category';

import { uid, updateInstanceWithRetry } from './_helpers';

async function expectInvalidArgument(fn: () => Promise<any>): Promise<void> {
  try {
    await fn();
    throw new Error('expected InvalidArgument error');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.code).toBe('InvalidArgument');
  }
}

test('base.soft_delete_options: withDeleted/onlyDeleted read and update deleted rows', async () => {
  const active = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteActiveCategory'),
      Code: uid('SDA').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const deleted = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteDeletedCategory'),
      Code: uid('SDD').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await UoMCategory.DeleteById(String((deleted as any).Id));

  const deletedId = String((deleted as any).Id);
  const activeId = String((active as any).Id);

  const defaultDeletedSearch = await UoMCategory.Search(['Id', '=', deletedId] as any, { fields: ['Id'] as any } as any);
  expect(defaultDeletedSearch.length).toBe(0);

  const withDeletedSearch = await UoMCategory.Search(
    ['Id', '=', deletedId] as any,
    {
      fields: ['Id'] as any,
      withDeleted: true,
    } as any
  );
  expect(withDeletedSearch.length).toBe(1);
  expect(String((withDeletedSearch[0] as any).Id)).toBe(deletedId);

  const onlyDeletedSearch = await UoMCategory.Search(
    ['Id', '=', deletedId] as any,
    {
      fields: ['Id'] as any,
      onlyDeleted: true,
    } as any
  );
  expect(onlyDeletedSearch.length).toBe(1);
  expect(String((onlyDeletedSearch[0] as any).Id)).toBe(deletedId);

  const onlyDeletedActiveSearch = await UoMCategory.Search(
    ['Id', '=', activeId] as any,
    {
      fields: ['Id'] as any,
      onlyDeleted: true,
    } as any
  );
  expect(onlyDeletedActiveSearch.length).toBe(0);

  const countDefault = await UoMCategory.Count(['Id', '=', deletedId] as any);
  const countWithDeleted = await UoMCategory.Count(['Id', '=', deletedId] as any, { withDeleted: true });
  const countOnlyDeleted = await UoMCategory.Count(['Id', '=', deletedId] as any, { onlyDeleted: true });
  expect(countDefault).toBe(0);
  expect(countWithDeleted).toBe(1);
  expect(countOnlyDeleted).toBe(1);

  try {
    await UoMCategory.Browse(deletedId, ['Id'] as any);
    throw new Error('expected NotFound for default Browse on deleted row');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.code).toBe('NotFound');
  }

  const deletedBrowseWithDeleted = await UoMCategory.Browse(deletedId, ['Id'] as any, { withDeleted: true });
  expect(String((deletedBrowseWithDeleted as any).Id)).toBe(deletedId);

  const updatedName = uid('SoftDeletedUpdatedName');
  const updateOnlyDeleted = await UoMCategory.Update(['Id', '=', deletedId] as any, { Name: updatedName } as any, ['Id', 'Name'] as any, { onlyDeleted: true });
  expect(updateOnlyDeleted.length).toBe(1);
  expect(String((updateOnlyDeleted[0] as any).Id)).toBe(deletedId);

  const afterUpdate = await UoMCategory.Browse(deletedId, ['Id', 'Name'] as any, { withDeleted: true });
  expect(String((afterUpdate as any).Name)).toBe(updatedName);
});

test('base.soft_delete_options: withDeleted and onlyDeleted conflict returns InvalidArgument', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteConflictCategory'),
      Code: uid('SDC').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);

  await expectInvalidArgument(async () => {
    await UoMCategory.Search([] as any, { fields: ['Id'] as any, withDeleted: true, onlyDeleted: true } as any);
  });

  await expectInvalidArgument(async () => {
    await UoMCategory.Count([] as any, { withDeleted: true, onlyDeleted: true });
  });

  await expectInvalidArgument(async () => {
    await UoMCategory.Update(['Id', '=', id] as any, { Name: uid('ShouldNotUpdate') } as any, ['Id'] as any, { withDeleted: true, onlyDeleted: true });
  });

  await expectInvalidArgument(async () => {
    await UoMCategory.Delete(['Id', '=', id] as any, { withDeleted: true, onlyDeleted: true });
  });
});

test('base.soft_delete_options: DeleteById with onlyDeleted targets soft-deleted rows', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteDeleteByIdOnlyDeletedCategory'),
      Code: uid('SDO').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);

  const firstDelete = await UoMCategory.DeleteById(id);
  expect(firstDelete).toBe(1);

  const defaultDeleteAgain = await UoMCategory.DeleteById(id);
  expect(defaultDeleteAgain).toBe(0);

  const onlyDeletedDelete = await UoMCategory.DeleteById(id, { onlyDeleted: true });
  expect(onlyDeletedDelete).toBe(1);
});

test('base.soft_delete_options: restore deleted row requires withDeleted context', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteRestoreCategory'),
      Code: uid('SDR').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const countDefaultBefore = await UoMCategory.Count(['Id', '=', id] as any);
  const countOnlyDeletedBefore = await UoMCategory.Count(['Id', '=', id] as any, { onlyDeleted: true });
  expect(countDefaultBefore).toBe(0);
  expect(countOnlyDeletedBefore).toBe(1);

  const restored = await UoMCategory.Update(['Id', '=', id] as any, { DeletedAt: null } as any, ['Id', 'DeletedAt'] as any, { withDeleted: true });
  expect(restored.length).toBe(1);

  const countDefaultAfter = await UoMCategory.Count(['Id', '=', id] as any);
  const countOnlyDeletedAfter = await UoMCategory.Count(['Id', '=', id] as any, { onlyDeleted: true });
  expect(countDefaultAfter).toBe(1);
  expect(countOnlyDeletedAfter).toBe(0);

  const browsed = await UoMCategory.Browse(id, ['Id'] as any);
  expect(String((browsed as any).Id)).toBe(id);
});

test('base.soft_delete_options: ReadGroupCount withDeleted includes deleted rows', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteReadGroupCountWithDeletedCategory'),
      Code: uid('SDRGC').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const countDefault = await UoMCategory.ReadGroupCount(['Id'] as any, ['Id', '=', id] as any, {} as any);
  const countWithDeleted = await UoMCategory.ReadGroupCount(['Id'] as any, ['Id', '=', id] as any, { withDeleted: true } as any);
  const countOnlyDeleted = await UoMCategory.ReadGroupCount(['Id'] as any, ['Id', '=', id] as any, { onlyDeleted: true } as any);

  expect(countDefault).toBe(0);
  expect(countWithDeleted).toBe(1);
  expect(countOnlyDeleted).toBe(1);
});

test('base.soft_delete_options: ReadGroup withDeleted and onlyDeleted conflict returns InvalidArgument', async () => {
  await expectInvalidArgument(async () => {
    await UoMCategory.ReadGroup(['Id'] as any, [] as any, { withDeleted: true, onlyDeleted: true } as any);
  });

  await expectInvalidArgument(async () => {
    await UoMCategory.ReadGroupCount(['Id'] as any, [] as any, { withDeleted: true, onlyDeleted: true } as any);
  });
});

test('base.soft_delete_options: ReadGroup withDeleted includes soft-deleted rows', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteReadGroupWithDeletedCategory'),
      Code: uid('SDRGW').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const groupedDefault = await UoMCategory.ReadGroup(['Id'] as any, ['Id', '=', id] as any, {} as any);
  const groupedWithDeleted = await UoMCategory.ReadGroup(['Id'] as any, ['Id', '=', id] as any, { withDeleted: true } as any);

  expect(groupedDefault.length).toBe(0);
  expect(groupedWithDeleted.length).toBe(1);
  expect(String((groupedWithDeleted[0] as any).keys?.Id)).toBe(id);
});

test('base.soft_delete_options: ReadGroup onlyDeleted returns only soft-deleted rows', async () => {
  const active = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteReadGroupOnlyDeletedActiveCategory'),
      Code: uid('SDRGA').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const deleted = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteReadGroupOnlyDeletedDeletedCategory'),
      Code: uid('SDRGD').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const activeId = String((active as any).Id);
  const deletedId = String((deleted as any).Id);
  await UoMCategory.DeleteById(deletedId);

  const groupedOnlyDeletedDeleted = await UoMCategory.ReadGroup(['Id'] as any, ['Id', '=', deletedId] as any, { onlyDeleted: true } as any);
  const groupedOnlyDeletedActive = await UoMCategory.ReadGroup(['Id'] as any, ['Id', '=', activeId] as any, { onlyDeleted: true } as any);

  expect(groupedOnlyDeletedDeleted.length).toBe(1);
  expect(String((groupedOnlyDeletedDeleted[0] as any).keys?.Id)).toBe(deletedId);
  expect(groupedOnlyDeletedActive.length).toBe(0);
});

test('base.soft_delete_options: instance load withDeleted can load soft-deleted row', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteInstanceLoadWithDeletedCategory'),
      Code: uid('SDILW').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const inst = await UoMCategory.Browse(id, ['Id'] as any, { withDeleted: true });

  let loadFailed = false;
  try {
    await (inst as any).load(['Id'] as any);
  } catch {
    loadFailed = true;
  }
  expect(loadFailed).toBe(true);

  await (inst as any).load(['Id'] as any, { withDeleted: true });
  expect(String((inst as any).Id)).toBe(id);
});

test('base.soft_delete_options: instance reload onlyDeleted can refresh soft-deleted row', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteInstanceReloadOnlyDeletedCategory'),
      Code: uid('SDIRO').slice(-16),
      IsActive: true,
    } as any,
    ['Id', 'Name'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const inst = await UoMCategory.Browse(id, ['Id', 'Name'] as any, { withDeleted: true });
  const updatedName = uid('SoftDeleteInstanceReloadOnlyDeletedUpdatedName');
  await UoMCategory.Update(['Id', '=', id] as any, { Name: updatedName } as any, ['Id', 'Name'] as any, { onlyDeleted: true });

  let reloadFailed = false;
  try {
    await (inst as any).reload();
  } catch {
    reloadFailed = true;
  }
  expect(reloadFailed).toBe(true);

  await (inst as any).reload({ onlyDeleted: true });
  expect(String((inst as any).Name)).toBe(updatedName);
});

test('base.soft_delete_options: instance load/reload conflict options return InvalidArgument', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteInstanceConflictCategory'),
      Code: uid('SDIC').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const inst = await UoMCategory.Browse(id, ['Id'] as any, { withDeleted: true });

  await expectInvalidArgument(async () => {
    await (inst as any).load(['Id'] as any, { withDeleted: true, onlyDeleted: true });
  });

  await expectInvalidArgument(async () => {
    await (inst as any).reload({ withDeleted: true, onlyDeleted: true });
  });
});

test('base.soft_delete_options: conflict options error domain is base', async () => {
  try {
    await UoMCategory.Search([] as any, { withDeleted: true, onlyDeleted: true } as any);
    throw new Error('expected InvalidArgument error');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.code).toBe('InvalidArgument');
    expect(oe.domain).toBe('base');
  }
});

test('base.soft_delete_options: instance update onlyDeleted updates soft-deleted row', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteInstanceUpdateOnlyDeletedCategory'),
      Code: uid('SDIU').slice(-16),
      IsActive: true,
    } as any,
    ['Id', 'Name', 'UpdatedAt'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const inst = await UoMCategory.Browse(id, ['Id', 'Name', 'UpdatedAt'] as any, { withDeleted: true });
  const updatedName = uid('SoftDeleteInstanceUpdateOnlyDeletedUpdatedName');
  await updateInstanceWithRetry(inst as any, { Name: updatedName }, { onlyDeleted: true });
  const after = await UoMCategory.Browse(id, ['Id', 'Name', 'UpdatedAt'] as any, { withDeleted: true });
  expect(String((after as any).Name)).toBe(updatedName);
});

test('base.soft_delete_options: instance delete onlyDeleted targets soft-deleted row', async () => {
  const row = await UoMCategory.Create(
    {
      Name: uid('SoftDeleteInstanceDeleteOnlyDeletedCategory'),
      Code: uid('SDID').slice(-16),
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const id = String((row as any).Id);
  await UoMCategory.DeleteById(id);

  const inst = await UoMCategory.Browse(id, ['Id'] as any, { withDeleted: true });

  let defaultDeleteFailed = false;
  try {
    await (inst as any).delete();
  } catch {
    defaultDeleteFailed = true;
  }
  expect(defaultDeleteFailed).toBe(true);

  await (inst as any).delete({ onlyDeleted: true });
  const defaultCount = await UoMCategory.Count(['Id', '=', id] as any);
  const onlyDeletedCount = await UoMCategory.Count(['Id', '=', id] as any, { onlyDeleted: true });
  expect(defaultCount).toBe(0);
  expect(onlyDeletedCount).toBe(1);
});
