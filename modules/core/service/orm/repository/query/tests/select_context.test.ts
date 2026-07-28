// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../../../metadata/storage';
import { makeSelectCtx } from '..';

function withFakeMetadata<T>(metas: Map<Function, any>, fn: () => T): T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    if (metas.has(model)) return metas.get(model);
    return original.call(this, model);
  };

  try {
    return fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

test('repository select context unwraps translated scalar fields for $sql.field', () => {
  class DemoModel {
    sqlDisplayName() {
      return (this as any).$sql.field(DemoModel as any, 'Name');
    }
  }

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { type: 'varchar', translate: true, column: { name: 'Name' } }],
      ['Code', { type: 'varchar', column: { name: 'Code' } }],
      ['DisplayName', {}],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      return { table };
    },
  };

  withFakeMetadata(new Map([[DemoModel, demoMeta]]), () => {
    const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
    // Non-translate stays a plain column ref.
    expect(ctx.field(DemoModel as any, 'Code') as any).toBe('ref:demo_table.Code');
    // Translate Name (and DisplayName SqlCompute that aliases it) must not be a bare jsonb ref.
    const nameExpr = ctx.field(DemoModel as any, 'Name') as any;
    expect(nameExpr).not.toBe('ref:demo_table.Name');
    const displayExpr = ctx.field(DemoModel as any, 'DisplayName') as any;
    expect(displayExpr).not.toBe('ref:demo_table.Name');
  });
});

test('repository select context resolves scalar/select fields and path existence from metadata', () => {
  class DemoModel {
    sqlDisplayName() {
      return (this as any).$sql.field(DemoModel as any, 'Name');
    }
  }

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['DisplayName', {}],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      return { table };
    },
  };

  withFakeMetadata(new Map([[DemoModel, demoMeta]]), () => {
    const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
    expect(ctx.model as any).toBe(DemoModel as any);
    expect(ctx.field(DemoModel as any, 'Name') as any).toBe('ref:demo_table.Name');
    expect(ctx.field('Name' as any) as any).toBe('ref:demo_table.Name');
    expect(ctx.field(DemoModel as any, 'DisplayName') as any).toBe('ref:demo_table.Name');
    expect(ctx.field('DisplayName' as any) as any).toBe('ref:demo_table.Name');
    expect(ctx.fieldExist(DemoModel as any, 'DisplayName')).toBe(true);
    expect(ctx.fieldExist('DisplayName' as any)).toBe(true);
    expect(ctx.fieldExist(DemoModel as any, 'Missing')).toBe(false);
    expect(ctx.fieldExist('Missing' as any)).toBe(false);
  });
});

test('repository select context builds many2one path subqueries without pulling root table into join scope', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
      const query = ctx.field(DemoModel as any, 'Owner.Name') as any;
      expect(query.ops).toEqual([
        { type: 'selectFrom', table: 'owner_table' },
        { type: 'select', selection: 'owner_table.Name' },
        { type: 'whereRef', left: 'owner_table.Id', op: '=', right: 'demo_table.Owner' },
      ]);
      expect(ctx.fieldExist(DemoModel as any, 'Owner.Name')).toBe(true);
      expect(ctx.fieldExist(DemoModel as any, 'Owner.Missing')).toBe(false);
    }
  );
});

test('repository select context reports field errors for missing or unsupported scalar metadata', () => {
  class DemoModel {}

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Broken', { type: 'char' }]]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };
  const db = { selectFrom: (table: string) => ({ table }) };

  withFakeMetadata(new Map([[DemoModel, demoMeta]]), () => {
    const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
    expect(() => ctx.field(DemoModel as any, 'Missing')).toThrow('field(DemoModel.Missing) not found');
    expect(() => ctx.field(DemoModel as any, 'Broken')).toThrow('field(DemoModel.Broken) has neither sql compute handler nor column');
    expect(ctx.fieldExist(DemoModel as any, 'Broken')).toBe(false);
  });
});

test('repository select context dotted path errors when intermediate relation or leaf metadata is invalid', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Name', { type: 'char' }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Name', { column: { name: 'Name' } }],
      ['Owner', { type: 'ManyToOne', relation: { targetModel: () => OwnerModel } }],
      ['Owner2', { type: 'ManyToOne', column: { name: 'Owner2' }, relation: { targetModel: () => OwnerModel } }],
    ]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      return {
        table,
        innerJoin() {
          return this;
        },
        select() {
          return this;
        },
        whereRef() {
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
      expect(() => ctx.field(DemoModel as any, 'Name.Other')).toThrow(
        'path Name.Other only supports ManyToOne chains; intermediate segment Name is not ManyToOne'
      );
      expect(() => ctx.field(DemoModel as any, 'Owner.Name')).toThrow('path segment Owner must be a ManyToOne field with a column');
      expect(() => ctx.field(DemoModel as any, 'Owner2.Missing')).toThrow('field(OwnerModel.Missing) not found');
      expect(() => ctx.field(DemoModel as any, 'Owner2.Name')).toThrow('field(OwnerModel.Name) has neither sql compute handler nor column');

      expect(ctx.fieldExist(DemoModel as any, 'Name.Other')).toBe(false);
      expect(ctx.fieldExist(DemoModel as any, 'Owner2.Name')).toBe(false);
      expect((ctx.selectFrom('x_table') as any).table).toBe('x_table');
      expect(ctx.col('demo_table', 'Name') as any).toBe('ref:demo_table.Name');
    }
  );
});

test('repository select context supports dotted leaf select, multi-hop anchor join and BaseModel fallback', () => {
  class DemoModel {}
  class OwnerModel {}
  class CompanyModel {
    sqlLabel() {
      return 'expr:company_label';
    }
  }

  const companyMeta = {
    type: CompanyModel,
    tableName: () => 'company_table',
    fields: new Map([
      ['Label', {}],
      ['Id', { column: { name: 'Id' } }],
    ]),
    sqlComputeHandlers: new Map([['Label', { field: 'Label', method: 'sqlLabel' }]]),
  } as any;

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([
      ['Company', { type: 'ManyToOne', column: { name: 'Company' }, relation: { targetModel: () => CompanyModel } }],
      ['Id', { column: { name: 'Id' } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }],
      ['Id', { column: { name: 'Id' } }],
    ]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
      [CompanyModel, companyMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
      const query = ctx.field(DemoModel as any, 'Owner.Company.Label') as any;
      expect(query.ops[0]).toEqual({ type: 'selectFrom', table: 'company_table' });
      expect(query.ops.some((x: any) => x.type === 'innerJoin' && x.left === 'owner_table')).toBe(true);
      expect(query.ops.some((x: any) => x.type === 'whereRef' && x.left === 'owner_table.Id')).toBe(true);
      expect(ctx.fieldExist(DemoModel as any, 'Owner.Company.Label')).toBe(true);

      const fallbackCtx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', { tableName: () => 'demo_table' } as any);
      expect(
        String((fallbackCtx.model as any).name || '')
          .toLowerCase()
          .includes('basemodel')
      ).toBe(true);
    }
  );
});

test('repository select context can hit missing segment branch after relation target side-effect mutation', () => {
  class DemoModel {}
  class OwnerModel {}

  const demoFields = new Map<string, any>();
  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: demoFields,
  } as any;

  demoFields.set('Owner', {
    type: 'ManyToOne',
    column: { name: 'Owner' },
    relation: {
      targetModel: () => {
        demoFields.delete('Owner');
        return OwnerModel;
      },
    },
  });

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Name', { column: { name: 'Name' } }]]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom() {
      return {
        innerJoin() {
          return this;
        },
        select() {
          return this;
        },
        whereRef() {
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
      expect(() => ctx.field(DemoModel as any, 'Owner.Name')).toThrow('field(DemoModel.Owner) not found');
    }
  );
});

test('repository select context executes dotted leaf select callback with nested select context', () => {
  class DemoModel {}
  class OwnerModel {
    sqlDisplayName() {
      return (this as any).$sql.col('owner_table', 'Name');
    }
  }

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([
      ['DisplayName', {}],
      ['Id', { column: { name: 'Id' } }],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }],
      ['Id', { column: { name: 'Id' } }],
    ]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          const value = typeof selection === 'function' ? selection(builder as any) : selection;
          ops.push({ type: 'select', value });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
      const query = ctx.field(DemoModel as any, 'Owner.DisplayName') as any;
      const selectOp = query.ops.find((x: any) => x.type === 'select');
      expect(selectOp?.value).toBe('ref:owner_table.Name');
    }
  );
});

test('repository select context optimizes id-tail dotted paths to avoid terminal table access', () => {
  class DemoModel {}
  class OwnerModel {}
  class CompanyModel {}

  const companyMeta = {
    type: CompanyModel,
    tableName: () => 'company_table',
    fields: new Map([['Id', { column: { name: 'Id' } }]]),
  } as any;

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['Company', { type: 'ManyToOne', column: { name: 'Company' }, relation: { targetModel: () => CompanyModel } }],
    ]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Id', { column: { name: 'Id' } }],
      ['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }],
    ]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
      [CompanyModel, companyMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);

      // A single-hop Id path reuses the root-table foreign-key column directly.
      expect(ctx.field(DemoModel as any, 'Owner.Id') as any).toBe('ref:demo_table.Owner');

      // A multi-hop Id path reads the Company foreign key from owner_table instead of visiting company_table.
      const query = ctx.field(DemoModel as any, 'Owner.Company.Id') as any;
      expect(query.ops).toEqual([
        { type: 'selectFrom', table: 'owner_table' },
        { type: 'select', selection: 'owner_table.Company' },
        { type: 'whereRef', left: 'owner_table.Id', op: '=', right: 'demo_table.Owner' },
      ]);
    }
  );
});

test('repository select context unwraps companyDependent scalar fields for $sql.field', () => {
  class DemoModel {}

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([
      ['Cost', { type: 'number', companyDependent: true, column: { name: 'Cost' } }],
      ['Code', { type: 'varchar', column: { name: 'Code' } }],
    ]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      return { table };
    },
  };

  withFakeMetadata(new Map([[DemoModel, demoMeta]]), () => {
    const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
    expect(ctx.field(DemoModel as any, 'Code') as any).toBe('ref:demo_table.Code');
    const costExpr = ctx.field(DemoModel as any, 'Cost') as any;
    expect(costExpr).not.toBe('ref:demo_table.Cost');
    expect(typeof costExpr.toOperationNode).toBe('function');
  });
});

test('repository select context unwraps companyDependent leaf on ManyToOne path', () => {
  class DemoModel {}
  class OwnerModel {}

  const ownerMeta = {
    type: OwnerModel,
    tableName: () => 'owner_table',
    fields: new Map([['Cost', { type: 'number', companyDependent: true, column: { name: 'Cost' } }]]),
  } as any;

  const demoMeta = {
    type: DemoModel,
    tableName: () => 'demo_table',
    fields: new Map([['Owner', { type: 'ManyToOne', column: { name: 'Owner' }, relation: { targetModel: () => OwnerModel } }]]),
  } as any;

  const builder = {
    ref(value: string) {
      return `ref:${value}`;
    },
    fn: {},
  };

  const db = {
    selectFrom(table: string) {
      const ops: any[] = [{ type: 'selectFrom', table }];
      return {
        ops,
        innerJoin(left: string, opLeft: string, opRight: string) {
          ops.push({ type: 'innerJoin', left, opLeft, opRight });
          return this;
        },
        select(selection: any) {
          ops.push({ type: 'select', selection });
          return this;
        },
        whereRef(left: string, op: string, right: string) {
          ops.push({ type: 'whereRef', left, op, right });
          return this;
        },
      };
    },
  };

  withFakeMetadata(
    new Map([
      [DemoModel, demoMeta],
      [OwnerModel, ownerMeta],
    ]),
    () => {
      const ctx = makeSelectCtx(db as any, () => 'postgres', builder as any, 'demo_table', demoMeta);
      const subquery = ctx.field(DemoModel as any, 'Owner.Cost') as any;
      expect(subquery.ops[0]).toEqual({ type: 'selectFrom', table: 'owner_table' });
      expect(typeof subquery.ops.find((o: any) => o.type === 'select')?.selection).toBe('function');
      const sel = subquery.ops.find((o: any) => o.type === 'select').selection;
      const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
      eb.ref = (path: string) => ({ kind: 'ref', path });
      const expr = sel(eb);
      expect(typeof expr.toOperationNode).toBe('function');
    }
  );
});
