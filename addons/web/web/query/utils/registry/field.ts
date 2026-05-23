// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Field selection registry with ref counting per store
import { ref, readonly } from 'vue';

type StoreId = string;

interface FieldNode {
  name: string;
  children?: Map<string, FieldNode>;
  refCount: number;
}

const storeRoots: Map<StoreId, FieldNode> = new Map();

// A reactive version counter to notify listeners when registry changes
const registryVersion = ref(0);
const bumpVersion = () => {
  registryVersion.value += 1;
};
export const useFieldRegistryVersion = () => readonly(registryVersion);

function getOrCreateRoot(storeId: StoreId): FieldNode {
  let root = storeRoots.get(storeId);
  if (!root) {
    root = { name: '__root__', children: new Map(), refCount: 0 };
    storeRoots.set(storeId, root);
  }
  return root;
}

function mergePath(root: FieldNode, path: string[]): void {
  let node = root;
  for (const seg of path) {
    if (!node.children) node.children = new Map();
    let child = node.children.get(seg);
    if (!child) {
      child = { name: seg, children: new Map(), refCount: 0 };
      node.children.set(seg, child);
    }
    child.refCount++;
    node = child;
  }
}

function removePath(root: FieldNode, path: string[]): void {
  const stack: { node: FieldNode; seg: string }[] = [];
  let node = root;
  for (const seg of path) {
    if (!node.children) return;
    const child = node.children.get(seg);
    if (!child) return;
    stack.push({ node, seg });
    node = child;
  }
  node.refCount = Math.max(0, node.refCount - 1);
  for (let i = stack.length - 1; i >= 0; i--) {
    const { node: parent, seg } = stack[i];
    const child = parent.children!.get(seg)!;
    if (child.refCount <= 0 && (!child.children || child.children.size === 0)) {
      parent.children!.delete(seg);
    }
  }
}

export function registerFieldPath(storeId: StoreId, path: string | string[]): void {
  const root = getOrCreateRoot(storeId);
  const parts = Array.isArray(path) ? path : path.split('.').filter(Boolean);
  if (parts.length === 0) return;
  mergePath(root, parts);
  bumpVersion();
}

export function unregisterFieldPath(storeId: StoreId, path: string | string[]): void {
  const root = getOrCreateRoot(storeId);
  const parts = Array.isArray(path) ? path : path.split('.').filter(Boolean);
  if (parts.length === 0) return;
  removePath(root, parts);
  bumpVersion();
}

function collectPaths(node: FieldNode, prefix: string[] = []): string[] {
  const out: string[] = [];
  if (!node.children || node.children.size === 0) return out;
  for (const [name, child] of node.children) {
    const next = [...prefix, name];
    if (!child.children || child.children.size === 0) {
      if (child.refCount > 0) out.push(next.join('.'));
    } else {
      out.push(...collectPaths(child, next));
    }
  }
  return out;
}

export function exportFieldSelection(storeId: StoreId): string[] {
  const root = getOrCreateRoot(storeId);
  const paths = collectPaths(root);
  // Always include Id so create and update responses carry the primary key.
  if (!paths.includes('Id')) {
    paths.push('Id');
  }
  return paths;
}

export function clearFieldsByStore(storeId: StoreId): void {
  storeRoots.delete(storeId);
  bumpVersion();
}

// Ensure root-level Id is selected when performing search/browse
export function ensureRootId(fields: any[] | undefined): any[] | undefined {
  if (!fields) return undefined;
  const hasId = fields.some(f => typeof f === 'string' && f === 'Id');
  if (hasId) return fields;
  return ['Id', ...fields];
}

// Convert dot-path selections (e.g., 'UserId.Username') into nested FieldSelection format
// Result example: ['Id', { UserId: ['Username'] }]
export function pathsToFieldSelection(paths: string[] | undefined): any[] | undefined {
  if (!paths || paths.length === 0) return undefined;

  type Node = { scalars: Set<string>; relations: Map<string, Node> };
  const makeNode = (): Node => ({ scalars: new Set(), relations: new Map() });
  const root: Node = makeNode();

  const addPath = (node: Node, parts: string[]) => {
    if (parts.length === 0) return;
    if (parts.length === 1) {
      node.scalars.add(parts[0]!);
      return;
    }
    const [head, ...rest] = parts;
    let child = node.relations.get(head);
    if (!child) {
      child = makeNode();
      node.relations.set(head, child);
    }
    addPath(child, rest);
  };

  for (const p of paths) {
    if (!p || typeof p !== 'string') continue;
    const parts = p.split('.').filter(Boolean);
    if (parts.length === 0) continue;
    addPath(root, parts);
  }

  const toSelection = (node: Node, isRoot = true): any[] => {
    const arr: any[] = [];
    const scalars = Array.from(node.scalars);
    // Automatically include Id on relation nodes; the root Id is handled by ensureRootId.
    if (!isRoot && !scalars.includes('Id')) scalars.unshift('Id');
    for (const s of scalars) arr.push(s);
    for (const [rel, child] of node.relations) {
      arr.push({ [rel]: toSelection(child, false) });
    }
    return arr;
  };

  return toSelection(root, true);
}

// For debugging/testing
export function _debugDump(storeId: StoreId): any {
  return storeRoots.get(storeId);
}
