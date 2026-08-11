// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Model } from '../decorator/model';
import { Field } from '../decorator/field';
import BaseModel from './model';
import PropertyDefinitionBaseModel from './property_definition_base_model';
import {
  __clearLookupPropertyDefinitionModelForTest,
  __setLookupPropertyDefinitionModelForTest,
} from './properties_lookup';
import { resolveProperties } from './properties_resolve';
import { validatePropertiesWrite } from './properties_write';
import { assertValidPropertyDefinitionItems, filterReadablePropertyDefinitionItems, isPlainPropertiesMap } from './properties_types';
import { validateModelPropertiesDefinitionFields } from '../metadata/properties_definition';
import { MetadataStorage } from '../metadata/storage';
import { ValidationPipelineError } from '../metadata';
import { isRegisteredLogicalModelName } from './logical_model_registry';

@Model('Pp1Project', { application: 'pp1test' })
class Pp1Project extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('Pp1Task', { application: 'pp1test' })
class Pp1Task extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Pp1Project },
  })
  ProjectId!: Pp1Project;

  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field<Pp1Task>({
    type: 'properties',
    definition: 'ProjectId',
  })
  TaskProperties!: Record<string, unknown>;
}

@Model('Pp1Partner', { application: 'pp1test' })
class Pp1Partner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'properties' })
  PartnerProperties!: Record<string, unknown>;
}

@Model('PropertyDefinition', { application: 'pp1test' })
class Pp1PropertyDefinition extends PropertyDefinitionBaseModel {}

function installDefinitionRows(rows: Array<Record<string, unknown>>): void {
  __setLookupPropertyDefinitionModelForTest('pp1test', {
    Search: async (condition: any) => {
      const and = condition?.And as unknown[] | undefined;
      if (!Array.isArray(and)) return rows;
      return rows.filter(row => {
        for (const clause of and) {
          if (!Array.isArray(clause) || clause.length < 3) continue;
          const [field, , expected] = clause;
          const actual = (row as any)[field as string];
          if (expected === null || expected === undefined) {
            if (actual != null && actual !== '') return false;
          } else if (String(actual) !== String(expected)) {
            return false;
          }
        }
        return true;
      });
    },
  });
}

async function expectRejects(promise: Promise<unknown>, codeOrMsg: string | RegExp) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    if (codeOrMsg instanceof RegExp) {
      const msg = err instanceof Error ? err.message : String(err);
      expect(codeOrMsg.test(msg)).toBe(true);
      return;
    }
    expect(err instanceof ValidationPipelineError).toBe(true);
    expect((err as ValidationPipelineError).issues?.[0]?.code).toBe(codeOrMsg);
  }
}

test('properties PP6: definition must be ManyToOne / ManyToOneRef', () => {
  const badMeta = {
    fields: new Map([
      ['Name', { name: 'Name', type: 'varchar' }],
      ['BadProperties', { name: 'BadProperties', type: 'properties', definition: 'Name' }],
    ]),
  } as any;
  expect(() => validateModelPropertiesDefinitionFields(badMeta)).toThrow(/ManyToOne or ManyToOneRef/);

  const okMeta = MetadataStorage.instance.getModelMetadata(Pp1Task as any);
  expect(() => validateModelPropertiesDefinitionFields(okMeta)).not.toThrow();
});

test('properties resolve: App-level merges schema ⊕ value', async () => {
  installDefinitionRows([
    {
      TargetModel: 'Pp1Partner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [
        { name: 'tax_id', type: 'char', string: 'Tax Id' },
        { name: 'vip', type: 'boolean', default: false },
      ],
    },
  ]);
  try {
    const items = await resolveProperties(Pp1Partner as any, { PartnerProperties: { tax_id: 'T1' } }, 'PartnerProperties');
    expect(items.length).toBe(2);
    expect(items[0]?.name).toBe('tax_id');
    expect(items[0]?.value).toBe('T1');
    expect(items[1]?.name).toBe('vip');
    expect(items[1]?.value).toBeUndefined();
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('properties resolve: empty parent id → [] without App-level fallback (PP2)', async () => {
  installDefinitionRows([
    {
      TargetModel: 'Pp1Task',
      PropertiesField: 'TaskProperties',
      ContainerId: null,
      Definition: [{ name: 'should_not_appear', type: 'char' }],
    },
    {
      TargetModel: 'Pp1Task',
      PropertiesField: 'TaskProperties',
      ContainerModel: 'Pp1Project',
      ContainerId: 'proj-1',
      Definition: [{ name: 'acceptance', type: 'char' }],
    },
  ]);
  try {
    const emptyParent = await resolveProperties(
      Pp1Task as any,
      { ProjectId: null, TaskProperties: { acceptance: 'x' } },
      'TaskProperties'
    );
    expect(emptyParent).toEqual([]);

    const withParent = await resolveProperties(
      Pp1Task as any,
      { ProjectId: 'proj-1', TaskProperties: { acceptance: 'ok' } },
      'TaskProperties'
    );
    expect(withParent.length).toBe(1);
    expect(withParent[0]?.value).toBe('ok');
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('properties resolve: Search failure propagates (not empty schema)', async () => {
  __setLookupPropertyDefinitionModelForTest('pp1test', {
    Search: async () => {
      throw new Error('db down');
    },
  });
  try {
    await expectRejects(resolveProperties(Pp1Partner as any, { PartnerProperties: {} }, 'PartnerProperties'), /db down/);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('properties write: reject array / unknown name / empty schema non-empty (PP3)', async () => {
  installDefinitionRows([
    {
      TargetModel: 'Pp1Partner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [
        { name: 'tax_id', type: 'char' },
        { name: 'note', type: 'text', readonly: true },
      ],
    },
  ]);
  try {
    await expectRejects(
      validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', [{ name: 'tax_id', value: 'x' }], {}),
      'PROPERTIES_WRITE_SHAPE'
    );
    await expectRejects(
      validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', { unknown: 1 }, {}),
      'PROPERTIES_WRITE_UNKNOWN_NAME'
    );
    await expectRejects(
      validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', new Date() as any, {}),
      'PROPERTIES_WRITE_SHAPE'
    );

    const normalized = await validatePropertiesWrite(
      Pp1Partner as any,
      'PartnerProperties',
      { tax_id: 'A', note: 'ignored' },
      {},
      { note: 'keep-me' }
    );
    expect(normalized).toEqual({ tax_id: 'A', note: 'keep-me' });

    await expectRejects(
      validatePropertiesWrite(Pp1Task as any, 'TaskProperties', { acceptance: 'x' }, { ProjectId: null }),
      'PROPERTIES_WRITE_NO_SCHEMA'
    );
    const emptyOk = await validatePropertiesWrite(Pp1Task as any, 'TaskProperties', {}, { ProjectId: null });
    expect(emptyOk).toEqual({});

    expect(await validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', null, {})).toBeNull();
    expect(await validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', undefined, {})).toBeUndefined();
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('properties write: date/datetime reject non-string; replace omits unsubmitted writable (PP4)', async () => {
  installDefinitionRows([
    {
      TargetModel: 'Pp1Partner',
      PropertiesField: 'PartnerProperties',
      ContainerId: null,
      Definition: [
        { name: 'a', type: 'char' },
        { name: 'b', type: 'char' },
        { name: 'due', type: 'date' },
      ],
    },
  ]);
  try {
    const next = await validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', { a: '1' }, {});
    expect(next).toEqual({ a: '1' });
    expect(Object.prototype.hasOwnProperty.call(next, 'b')).toBe(false);

    await expectRejects(
      validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', { due: true }, {}),
      'PROPERTIES_WRITE_TYPE'
    );
    const okDate = await validatePropertiesWrite(Pp1Partner as any, 'PartnerProperties', { due: '2026-01-01' }, {});
    expect(okDate).toEqual({ due: '2026-01-01' });
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
  }
});

test('isPlainPropertiesMap rejects Date/Map', () => {
  expect(isPlainPropertiesMap({})).toBe(true);
  expect(isPlainPropertiesMap(new Date())).toBe(false);
  expect(isPlainPropertiesMap(new Map())).toBe(false);
  expect(isPlainPropertiesMap([])).toBe(false);
});

test('PropertyDefinition Definition rejects PP7-outside types; dirty read skips', () => {
  expect(() => assertValidPropertyDefinitionItems([{ name: 'rel', type: 'many2one' }])).toThrow(/unsupported type/);
  expect(() => assertValidPropertyDefinitionItems([{ name: 's', type: 'selection' }])).toThrow(/non-empty selection/);
  expect(() =>
    assertValidPropertyDefinitionItems([{ name: 'n', type: 'integer', default: 'x' }])
  ).toThrow(/default does not match type/);
  const ok = assertValidPropertyDefinitionItems([
    { name: 'n', type: 'char', default: 'x' },
    { name: 's', type: 'selection', selection: [{ value: 'a', label: 'A' }], default: 'a' },
  ]);
  expect(ok[0]?.name).toBe('n');
  expect(ok[1]?.name).toBe('s');

  const skipped: string[] = [];
  const readable = filterReadablePropertyDefinitionItems(
    [
      { name: 'ok', type: 'char' },
      { name: 'bad', type: 'many2one' },
    ],
    item => skipped.push(item.name)
  );
  expect(readable.map(i => i.name)).toEqual(['ok']);
  expect(skipped).toEqual(['bad']);
});

test('changing parent relation alone does not mutate properties map (PP4 B1)', () => {
  const map = { acceptance: 'keep-me' };
  const row: { ProjectId: string; TaskProperties: Record<string, unknown> } = {
    ProjectId: 'proj-1',
    TaskProperties: map,
  };
  row.ProjectId = 'proj-2';
  expect(row.TaskProperties).toBe(map);
  expect(row.TaskProperties).toEqual({ acceptance: 'keep-me' });
});

test('PropertyDefinitionBaseModel registers logical model name', () => {
  expect(isRegisteredLogicalModelName('PropertyDefinition')).toBe(true);
  expect(Pp1PropertyDefinition).toBeTruthy();
});
