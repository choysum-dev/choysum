// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guard: collection APIs infer the row type from the calling ctor.
 * Assertions live in a never-invoked function so module evaluation does not run ORM calls.
 */
import BaseModel from './model';
import type { Projected } from '../repository/types';
import { fields } from '../repository/types';

class Probe extends BaseModel {
  Name!: string;
}

function typecheckStaticThis(): void {
  const created: Promise<Probe> = Probe.Create({ Name: 'x' });
  void created;

  const searched: Promise<Probe[]> = Probe.Search(['Name', '=', 'x']);
  void searched;

  const browsedProjected: Promise<Array<Projected<Probe, ['Name']>>> = Probe.BrowseMany(['id'], ['Name'] as const);
  void browsedProjected;

  const browsedFull: Promise<Probe[]> = Probe.BrowseMany(['id']);
  void browsedFull;

  const searchedProjected: Promise<Array<Projected<Probe, ['Name']>>> = Probe.Search([], {
    fields: fields<Probe>()('Name'),
  });
  void searchedProjected;

  const createdProjected: Promise<Projected<Probe, ['Name']>> = Probe.Create({ Name: 'x' }, fields<Probe>()('Name'));
  void createdProjected;

  const updatedProjected: Promise<Array<Projected<Probe, ['Name']>>> = Probe.Update(
    ['Id', '=', 'id'],
    { Name: 'x' },
    fields<Probe>()('Name')
  );
  void updatedProjected;

  const browsedOneProjected: Promise<Projected<Probe, ['Name']>> = Probe.Browse('id', fields<Probe>()('Name'));
  void browsedOneProjected;

  const createdManyProjected: Promise<Array<Projected<Probe, ['Name']>>> = Probe.CreateMany(
    [{ Name: 'x' }],
    fields<Probe>()('Name')
  );
  void createdManyProjected;

  const updatedByIdProjected: Promise<Projected<Probe, ['Name']>> = Probe.UpdateById(
    'id',
    { Name: 'x' },
    fields<Probe>()('Name')
  );
  void updatedByIdProjected;

  // @ts-expect-error unknown field is not a QueryCondition path
  const invalidSearch = Probe.Search(['NoSuch', '=', 1]);
  void invalidSearch;
}
void typecheckStaticThis;
