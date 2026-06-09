// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Model relation processing module.
 * Exposes processors and helpers for handling relations between entities.
 */

// Export the abstract base class.
export { RelationProcessor } from './processor';

// Export concrete relation processors.
export { ManyToOneProcessor } from './many-to-one';
export { OneToManyProcessor } from './one-to-many';
export { ManyToManyProcessor } from './many-to-many';

// Export the relation factory.
export { RelationFactory } from './factory';

// Export all relation type definitions.
export type {
  ManyToOneOperation,
  OneToManyOperation,
  ManyToManyOperation,
  RelationOperation,
  ExtractedRelations,
  PrepareResult,
  RelationProcessingResult,
  BatchProcessingResult,
  RelationChangeOperation,
  RelationChangesCollection,
} from './types';
export { RelationArrayMethod } from './types';
