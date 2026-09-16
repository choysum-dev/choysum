// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guard: collection APIs infer the row type from the calling ctor.
 * Assertions live in a never-invoked function so module evaluation does not run ORM calls.
 */
import BaseModel from './model';
import type { Projected } from '../repository/types';

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

  // @ts-expect-error unknown field is not a QueryCondition path
  const invalidSearch = Probe.Search(['NoSuch', '=', 1]);
  void invalidSearch;
}
void typecheckStaticThis;
