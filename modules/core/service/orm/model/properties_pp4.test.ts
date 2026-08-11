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
import {
  __setParentWritableProbeForTest,
  assertPropertyDefinitionParentWritable,
} from './properties_definition_acl';
import { purgePropertyDefinitionsForContainers } from './properties_definition_purge';
import { resolveProperties } from './properties_resolve';
import { validatePropertiesWrite } from './properties_write';

@Model('Pp4Project', { application: 'pp4test' })
class Pp4Project extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('Pp4Task', { application: 'pp4test' })
class Pp4Task extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Pp4Project },
  })
  ProjectId!: Pp4Project;

  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field<Pp4Task>({
    type: 'properties',
    definition: 'ProjectId',
  })
  TaskProperties!: Record<string, unknown>;
}

@Model('Pp4Partner', { application: 'pp4test' })
class Pp4Partner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'properties' })
  PartnerProperties!: Record<string, unknown>;
}

@Model('PropertyDefinition', { application: 'pp4test' })
class Pp4PropertyDefinition extends PropertyDefinitionBaseModel {}

type DefRow = Record<string, unknown> & { Id?: string };

function installDefinitionStore(rows: DefRow[]) {
  const store = {
    Search: async (condition: any) => {
      const and = condition?.And as unknown[] | undefined;
      if (!Array.isArray(and)) return [...rows];
      return rows.filter(row => {
        for (const clause of and) {
          if (!Array.isArray(clause) || clause.length < 3) continue;
          const [field, op, expected] = clause;
          const actual = (row as any)[field as string];
          if (op === 'in') {
            const list = Array.isArray(expected) ? expected.map(String) : [];
            if (!list.includes(String(actual ?? ''))) return false;
            continue;
          }
          if (op === '!=') {
            if (String(actual) === String(expected)) return false;
            continue;
          }
          if (expected === null || expected === undefined) {
            if (actual != null && actual !== '') return false;
          } else if (String(actual) !== String(expected)) {
            return false;
          }
        }
        return true;
      });
    },
    Delete: async (condition: any) => {
      const matched = await store.Search(condition);
      const ids = new Set(matched.map((r: any) => String(r.Id)));
      const before = rows.length;
      const next = rows.filter(r => !ids.has(String(r.Id)));
      rows.length = 0;
      rows.push(...next);
      return before - rows.length;
    },
    DeleteById: async (id: string) => {
      const before = rows.length;
      const next = rows.filter(r => String(r.Id) !== String(id));
      rows.length = 0;
      rows.push(...next);
      return before - rows.length;
    },
  };
  __setLookupPropertyDefinitionModelForTest('pp4test', store as any);
  return { rows, store };
}

async function expectRejects(promise: Promise<unknown>, codeOrMsg: string | RegExp) {
  try {
    await promise;
    throw new Error(`expected rejection matching ${codeOrMsg}`);
  } catch (e: any) {
    if (e?.message === `expected rejection matching ${codeOrMsg}`) throw e;
    const hay = `${e?.code || ''} ${e?.message || e}`;
    if (codeOrMsg instanceof RegExp) expect(hay).toMatch(codeOrMsg);
    else expect(hay.includes(codeOrMsg)).toBe(true);
  }
}

test('PP4: App-level definition edit updates resolve; deleting a key does not scrub child JSON', async () => {
  const childRecord = {
    Id: 'partner-1',
    PartnerProperties: { tax_id: 'T1', vip: true, orphan: 'keep' } as Record<string, unknown>,
  };
  const { rows } = installDefinitionStore([
    {
      Id: 'app-1',
      TargetModel: 'Pp4Partner',
      PropertiesField: 'PartnerProperties',
      ContainerModel: null,
      ContainerId: null,
      Definition: [
        { name: 'tax_id', type: 'char' },
        { name: 'vip', type: 'boolean', default: false },
      ],
    },
  ]);
  try {
    const before = await resolveProperties(
      Pp4Partner as any,
      childRecord,
      'PartnerProperties'
    );
    expect(before.map(i => i.name)).toEqual(['tax_id', 'vip']);

    // Simulate definition edit (definition-table Update) without cascading to child JSON.
    rows[0]!.Definition = [{ name: 'tax_id', type: 'char' }];

    const after = await resolveProperties(
      Pp4Partner as any,
      childRecord,
      'PartnerProperties'
    );
    expect(after.map(i => i.name)).toEqual(['tax_id']);

    // Child JSON is untouched by definition edits (resolve only filters for display).
    expect(childRecord.PartnerProperties).toHaveProperty('vip');
    expect(childRecord.PartnerProperties).toHaveProperty('orphan');

    const written = await validatePropertiesWrite(Pp4Partner as any, 'PartnerProperties', { tax_id: 'T2' }, {});
    expect(written).toEqual({ tax_id: 'T2' });
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
    __setParentWritableProbeForTest(undefined);
  }
});

test('PP4: parent-container write denied when parent probe rejects; App-level skips probe', async () => {
  installDefinitionStore([]);
  __setParentWritableProbeForTest(async () => {
    throw Object.assign(new Error('record rule denied'), { code: 'record_rule_denied' });
  });
  try {
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4PropertyDefinition as any, {
        TargetModel: 'Pp4Task',
        PropertiesField: 'TaskProperties',
        ContainerModel: 'Pp4Project',
        ContainerId: 'proj-1',
      }),
      'PROPERTY_DEFINITION_PARENT_WRITE_DENIED'
    );

    // App-level: probe must not run.
    let probed = false;
    __setParentWritableProbeForTest(async () => {
      probed = true;
    });
    await assertPropertyDefinitionParentWritable(Pp4PropertyDefinition as any, {
      TargetModel: 'Pp4Partner',
      PropertiesField: 'PartnerProperties',
      ContainerModel: null,
      ContainerId: null,
    });
    expect(probed).toBe(false);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
    __setParentWritableProbeForTest(undefined);
  }
});

test('PP4: parent writable probe required fields and missing parent model', async () => {
  try {
    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4PropertyDefinition as any, {
        ContainerId: 'x',
        ContainerModel: null,
      }),
      'PROPERTY_DEFINITION_PARENT_SCOPE'
    );

    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4PropertyDefinition as any, {
        ContainerId: null,
        ContainerModel: 'Pp4Project',
      }),
      'PROPERTY_DEFINITION_PARENT_SCOPE'
    );

    await expectRejects(
      assertPropertyDefinitionParentWritable(Pp4PropertyDefinition as any, {
        ContainerId: 'x',
        ContainerModel: 'DoesNotExistModel',
      }),
      'PROPERTY_DEFINITION_PARENT_MODEL'
    );
  } finally {
    __setParentWritableProbeForTest(undefined);
  }
});

test('PP4: purge container definitions leaves App-level and other parents', async () => {
  const { rows } = installDefinitionStore([
    {
      Id: 'p1',
      TargetModel: 'Pp4Task',
      PropertiesField: 'TaskProperties',
      ContainerModel: 'Pp4Project',
      ContainerId: 'proj-keep',
      Definition: [{ name: 'a', type: 'char' }],
    },
    {
      Id: 'p2',
      TargetModel: 'Pp4Task',
      PropertiesField: 'TaskProperties',
      ContainerModel: 'Pp4Project',
      ContainerId: 'proj-gone',
      Definition: [{ name: 'b', type: 'char' }],
    },
    {
      Id: 'p2q',
      TargetModel: 'Pp4Task',
      PropertiesField: 'TaskProperties',
      ContainerModel: 'pp4test.Pp4Project',
      ContainerId: 'proj-gone',
      Definition: [{ name: 'bq', type: 'char' }],
    },
    {
      Id: 'app',
      TargetModel: 'Pp4Partner',
      PropertiesField: 'PartnerProperties',
      ContainerModel: null,
      ContainerId: null,
      Definition: [{ name: 'c', type: 'char' }],
    },
  ]);
  try {
    const n = await purgePropertyDefinitionsForContainers('pp4test', 'Pp4Project', ['proj-gone']);
    expect(n).toBe(2);
    expect(rows.map(r => r.Id).sort()).toEqual(['app', 'p1']);
  } finally {
    __clearLookupPropertyDefinitionModelForTest();
    __setParentWritableProbeForTest(undefined);
  }
});
