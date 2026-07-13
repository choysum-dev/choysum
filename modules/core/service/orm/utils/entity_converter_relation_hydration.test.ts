// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { REL_ALIAS_PREFIX } from '../relation/relation_alias';
import { EntityConverter } from './converter';

@Model('test.EntityConverterHydratedUser')
class EntityConverterHydratedUser extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.EntityConverterHydratedTeam')
class EntityConverterHydratedTeam extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => EntityConverterHydratedUser }, notNull: false })
  Owner?: EntityConverterHydratedUser;

  @Field({ type: 'OneToMany', relation: { targetModel: () => EntityConverterHydratedMember, inverseField: 'TeamId' } })
  Members?: EntityConverterHydratedMember[];
}

@Model('test.EntityConverterHydratedMember')
class EntityConverterHydratedMember extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => EntityConverterHydratedTeam }, notNull: false })
  TeamId?: EntityConverterHydratedTeam;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

function createModelInstance<T extends BaseModel>(ModelCtor: { new (...args: any[]): T } & typeof BaseModel, entity: Record<string, any>): T {
  const factoryToken = (ModelCtor as any).FACTORY_TOKEN;
  return new ModelCtor(factoryToken, entity, undefined as any);
}

test('EntityConverter direct many2one relation hydration does not call public Model.Hydrate', () => {
  const originalHydrate = EntityConverterHydratedUser.Hydrate;
  let hydrateCalls = 0;

  try {
    EntityConverterHydratedUser.Hydrate = ((_entity: Record<string, any>) => {
      hydrateCalls += 1;
      throw new Error('public hydrate should not be called');
    }) as any;

    const team = createModelInstance(EntityConverterHydratedTeam, {
      Id: 'TEAM-1',
      Owner: { Id: 'USER-1', Name: 'Alice' },
    });

    expect(hydrateCalls).toBe(0);
    expect(team.Owner instanceof EntityConverterHydratedUser).toBe(true);
    expect(team.Owner?.Id).toBe('USER-1');
    expect(team.Owner?.Name).toBe('Alice');
  } finally {
    EntityConverterHydratedUser.Hydrate = originalHydrate;
  }
});

test('EntityConverter preloaded to-many relation hydration does not call public Model.Hydrate', () => {
  const originalHydrate = EntityConverterHydratedMember.Hydrate;
  let hydrateCalls = 0;

  try {
    EntityConverterHydratedMember.Hydrate = ((_entity: Record<string, any>) => {
      hydrateCalls += 1;
      throw new Error('public hydrate should not be called');
    }) as any;

    const team = createModelInstance(EntityConverterHydratedTeam, {
      Id: 'TEAM-2',
      [`${REL_ALIAS_PREFIX}Members`]: [
        { Id: 'MEM-1', Name: 'alpha' },
        { Id: 'MEM-2', Name: 'beta' },
      ],
    });

    expect(hydrateCalls).toBe(0);
    expect(Array.isArray(team.Members)).toBe(true);
    expect(team.Members?.length).toBe(2);
    expect(team.Members?.[0] instanceof EntityConverterHydratedMember).toBe(true);
    expect(team.Members?.[0]?.Id).toBe('MEM-1');
    expect(team.Members?.[1]?.Name).toBe('beta');
  } finally {
    EntityConverterHydratedMember.Hydrate = originalHydrate;
  }
});
