// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Compile-only guard: collection APIs infer the row type from the calling ctor.
 */
import BaseModel from './model';

class Probe extends BaseModel {
  Name!: string;
}

const created: Promise<Probe> = Probe.Create({ Name: 'x' });
void created;

const searched: Promise<Probe[]> = Probe.Search(['Name', '=', 'x']);
void searched;

const browsed: Promise<Probe[]> = Probe.BrowseMany(['id'], ['Name']);
void browsed;

// @ts-expect-error unknown field is not a QueryCondition path
const invalidSearch = Probe.Search(['NoSuch', '=', 1]);
void invalidSearch;
