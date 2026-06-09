// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import UoM from '@/base/service/models/uom';
import UoMCategory from '@/base/service/models/uom_category';
import { ChoysumError } from '@/core/service/error';
import { resolveValidationSummary } from '@/core/service/api/validation';

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

test('base.uom: Name must be unique within CategoryId', async () => {
  const categoryA = await createCategory();
  const categoryB = await createCategory();

  const sharedName = `SameName_${uid('uomname')}`;

  await UoM.Create(
    {
      Name: sharedName,
      CategoryId: categoryA,
      IsReference: true,
      Factor: '1',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectRepoValidationFailed('create', async () => {
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
  });

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

test('base.uom: UpdateById rejects Name conflict within CategoryId', async () => {
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

  await expectRepoValidationFailed('update', async () => {
    await UoM.UpdateById(
      String((secondary as any).Id),
      {
        Name: nameA,
      } as any,
      ['Id'] as any
    );
  });
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
