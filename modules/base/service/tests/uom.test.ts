// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import UoM from '@/base/service/models/uom';
import UoMCategory from '@/base/service/models/uom_category';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

import { uid } from './_helpers';

async function expectRepoValidationFailed(mode: 'create' | 'update', fn: () => Promise<void>): Promise<void> {
  let error: unknown;
  try {
    await fn();
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe(mode);
  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
}

async function createCategory(): Promise<string> {
  const category = await UoMCategory.Create(
    {
      Name: uid('UomCategory'),
      Code: uid('UCODE').slice(-12),
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((category as any).Id);
}

test('base.uom: validates Factor and Rounding must be positive', async () => {
  const categoryId = await createCategory();

  await expectRepoValidationFailed('create', async () => {
    await UoM.Create(
      {
        Name: uid('UomBadFactor'),
        CategoryId: categoryId,
        IsReference: true,
        Factor: '0',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed('create', async () => {
    await UoM.Create(
      {
        Name: uid('UomBadRounding'),
        CategoryId: categoryId,
        IsReference: true,
        Factor: '1',
        Rounding: '0',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.uom: reference unit must have Factor=1 and category needs exactly one reference', async () => {
  const categoryId = await createCategory();

  await expectRepoValidationFailed('create', async () => {
    await UoM.Create(
      {
        Name: uid('UomNonRefFirst'),
        CategoryId: categoryId,
        IsReference: false,
        Factor: '2',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed('create', async () => {
    await UoM.Create(
      {
        Name: uid('UomBadRefFactor'),
        CategoryId: categoryId,
        IsReference: true,
        Factor: '2',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  const ref = await UoM.Create(
    {
      Name: uid('UomRef'),
      CategoryId: categoryId,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('create', async () => {
    await UoM.Create(
      {
        Name: uid('UomSecondRef'),
        CategoryId: categoryId,
        IsReference: true,
        Factor: '1',
        IsActive: true,
      } as any,
      ['Id'] as any
    );
  });

  await expectRepoValidationFailed('update', async () => {
    await UoM.UpdateById(
      String((ref as any).Id),
      {
        IsReference: false,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.uom: non-reference update still enforces positive Factor', async () => {
  const categoryId = await createCategory();

  await UoM.Create(
    {
      Name: uid('UomRefOk'),
      CategoryId: categoryId,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const secondary = await UoM.Create(
    {
      Name: uid('UomSecondary'),
      CategoryId: categoryId,
      IsReference: false,
      Factor: '2',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('update', async () => {
    await UoM.UpdateById(
      String((secondary as any).Id),
      {
        Factor: '-1',
      } as any,
      ['Id'] as any
    );
  });
});

test('base.uom: Name is translate and may repeat within CategoryId', async () => {
  const nameField = MetadataStorage.instance.getModelMetadata(UoM).fields.get('Name');
  expect(nameField?.translate).toBe(true);
  expect(nameField?.column?.index).toBe('trigram');
  expect(nameField?.column?.uniqueIndex).toBeFalsy();

  const categoryA = await createCategory();
  const categoryB = await createCategory();

  const sharedName = `SameName_${uid('uomname')}`;
  const enName = uid('UomEn');
  const zhName = uid('UomZh');

  const created = await UoM.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      CategoryId: categoryA,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id', 'Name'] as any
  );
  expect(String((created as any).Name)).toBe(enName);

  const zhBrowse = await withContext({ lang: 'zh_CN' }, () =>
    UoM.Browse(String((created as any).Id), ['Id', 'Name'] as any)
  );
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  // Same Name in the same category is allowed after translate migration
  await UoM.Create(
    {
      Name: sharedName,
      CategoryId: categoryA,
      IsReference: false,
      Factor: '2',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  // Same Name in a different category is allowed
  await UoM.Create(
    {
      Name: sharedName,
      CategoryId: categoryB,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );
});

test('base.uom: UpdateById allows renaming to an existing Name within CategoryId', async () => {
  const categoryId = await createCategory();
  const nameA = `NameA_${uid('uoma')}`;
  const nameB = `NameB_${uid('uomb')}`;

  await UoM.Create(
    {
      Name: nameA,
      CategoryId: categoryId,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const secondary = await UoM.Create(
    {
      Name: nameB,
      CategoryId: categoryId,
      IsReference: false,
      Factor: '2',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const updated = await UoM.UpdateById(
    String((secondary as any).Id),
    {
      Name: nameA,
      CategoryId: categoryId,
      IsReference: false,
      Factor: '2',
    } as any,
    ['Id', 'Name'] as any
  );
  expect(String((updated as any).Name)).toBe(nameA);
});

test('base.uom: Batch update cannot change Name or CategoryId', async () => {
  const categoryId = await createCategory();

  await UoM.Create(
    {
      Name: uid('UomRefForBatch'),
      CategoryId: categoryId,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await UoM.Create(
    {
      Name: uid('UomSecondaryForBatch'),
      CategoryId: categoryId,
      IsReference: false,
      Factor: '2',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('update', async () => {
    await UoM.Update(
      {
        And: [['CategoryId', '=', categoryId]],
      } as any,
      {
        Name: uid('NewName'),
      } as any,
      ['Id'] as any
    );
  });
});

test('base.uom: ReferenceSlotKey backs one-reference-per-category uniqueness', async () => {
  const categoryId = await createCategory();

  const ref = await UoM.Create(
    {
      Name: uid('UomRefSlot'),
      CategoryId: categoryId,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id', 'ReferenceSlotKey', 'IsReference'] as any
  );
  expect((ref as any).ReferenceSlotKey).toBe('__REF__');

  const secondary = await UoM.Create(
    {
      Name: uid('UomNonRefSlot'),
      CategoryId: categoryId,
      IsReference: false,
      Factor: '2',
      IsActive: true,
    } as any,
    ['Id', 'ReferenceSlotKey'] as any
  );
  expect((secondary as any).ReferenceSlotKey == null || (secondary as any).ReferenceSlotKey === '').toBe(true);

  const slotField = MetadataStorage.instance.getModelMetadata(UoM).fields.get('ReferenceSlotKey');
  expect(slotField?.column?.uniqueIndex).toBe('uidx_base_uom_category_reference_slot');
  const categoryField = MetadataStorage.instance.getModelMetadata(UoM).fields.get('CategoryId');
  expect(categoryField?.column?.uniqueIndex).toBe('uidx_base_uom_category_reference_slot');
});

test('base.uom: concurrent reference creates leave at most one reference', async () => {
  const categoryId = await createCategory();

  const results = await Promise.allSettled([
    UoM.Create(
      {
        Name: uid('UomRaceRefA'),
        CategoryId: categoryId,
        IsReference: true,
        Factor: '1',
        IsActive: true,
      } as any,
      ['Id'] as any
    ),
    UoM.Create(
      {
        Name: uid('UomRaceRefB'),
        CategoryId: categoryId,
        IsReference: true,
        Factor: '1',
        IsActive: true,
      } as any,
      ['Id'] as any
    ),
  ]);

  const fulfilled = results.filter(item => item.status === 'fulfilled');
  const rejected = results.filter(item => item.status === 'rejected');
  expect(fulfilled.length).toBe(1);
  expect(rejected.length).toBe(1);

  const refs = await UoM.Search(
    {
      And: [
        ['CategoryId', '=', categoryId],
        ['IsReference', '=', true],
      ],
    } as any,
    { fields: ['Id'] as any, limit: 5 } as any
  );
  expect((refs || []).length).toBe(1);
});
