// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package choysumtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runNodeWithScript(t *testing.T, source string) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not found in PATH")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "choysumtest_runtime_check.js")
	if err := os.WriteFile(scriptPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write node script: %v", err)
	}

	cmd := exec.Command("node", scriptPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node script failed: %v\n%s", err, string(out))
	}
}

func TestChoysumTestScriptExportsExpectRejects(t *testing.T) {
	if !strings.Contains(ChoysumTestScript, "globalThis.expectRejects = expectRejects") {
		t.Fatalf("expected script to export expectRejects")
	}
}

func TestChoysumTestScriptMatchersAndExpectRejectsRuntime(t *testing.T) {
	script := ChoysumTestScript + `
function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

(async () => {
  expect('hello world').toContain('world');
  expect([1, 2, 3]).toContain(2);

  let containFailed = false;
  try {
    expect([1, 2]).toContain(3);
  } catch (e) {
    containFailed = String((e && e.message) || e).indexOf('to contain') >= 0;
  }
  assert(containFailed, 'toContain should fail when value is missing');

  expect(() => { throw new Error('boom message'); }).toThrow();
  expect(() => { throw new Error('boom message'); }).toThrow('boom');
  expect(() => { throw new Error('boom message'); }).toThrow(/message/);

  let toThrowFailed = false;
  try {
    expect(() => {}).toThrow();
  } catch (e) {
    toThrowFailed = String((e && e.message) || e).indexOf('Expected function to throw') >= 0;
  }
  assert(toThrowFailed, 'toThrow should fail when function does not throw');

  await expectRejects(Promise.reject(new Error('reject message')), 'reject');
  await expectRejects(async () => { throw new Error('async reject'); }, /async/);

  let rejectsFailed = false;
  try {
    await expectRejects(Promise.resolve('ok'));
  } catch (e) {
    rejectsFailed = String((e && e.message) || e).indexOf('Expected promise to reject') >= 0;
  }
  assert(rejectsFailed, 'expectRejects should fail when promise resolves');

  expect(undefined).toBeUndefined();
  expect('value').toBeDefined();
  expect(null).toBeNull();

  let undefinedFailed = false;
  try {
    expect('not-undefined').toBeUndefined();
  } catch (e) {
    undefinedFailed = String((e && e.message) || e).indexOf('to be undefined') >= 0;
  }
  assert(undefinedFailed, 'toBeUndefined should fail for non-undefined values');

  let definedFailed = false;
  try {
    expect(undefined).toBeDefined();
  } catch (e) {
    definedFailed = String((e && e.message) || e).indexOf('to be defined') >= 0;
  }
  assert(definedFailed, 'toBeDefined should fail for undefined values');

  let nullFailed = false;
  try {
    expect('not-null').toBeNull();
  } catch (e) {
    nullFailed = String((e && e.message) || e).indexOf('to be null') >= 0;
  }
  assert(nullFailed, 'toBeNull should fail for non-null values');

  expect('alpha-beta').toMatch('beta');
  expect('alpha-beta').toMatch(/alpha/);

  let matchFailed = false;
  try {
    expect('alpha-beta').toMatch(/gamma/);
  } catch (e) {
    matchFailed = String((e && e.message) || e).indexOf('to match') >= 0;
  }
  assert(matchFailed, 'toMatch should fail on unmatched regex');

  expect([1, 2, 3]).toHaveLength(3);
  expect('hello').toHaveLength(5);

  let lengthFailed = false;
  try {
    expect([1, 2, 3]).toHaveLength(4);
  } catch (e) {
    lengthFailed = String((e && e.message) || e).indexOf('Expected length') >= 0;
  }
  assert(lengthFailed, 'toHaveLength should fail on mismatched length');

  expect(10).toBeGreaterThan(9);
  expect(9).toBeLessThan(10);
  expect(10).toBeGreaterThanOrEqual(10);
  expect(10).toBeLessThanOrEqual(10);

  let gtFailed = false;
  try {
    expect(5).toBeGreaterThan(6);
  } catch (e) {
    gtFailed = String((e && e.message) || e).indexOf('greater than') >= 0;
  }
  assert(gtFailed, 'toBeGreaterThan should fail when comparison is false');

  let ltFailed = false;
  try {
    expect(7).toBeLessThan(6);
  } catch (e) {
    ltFailed = String((e && e.message) || e).indexOf('less than') >= 0;
  }
  assert(ltFailed, 'toBeLessThan should fail when comparison is false');

  let gteFailed = false;
  try {
    expect(7).toBeGreaterThanOrEqual(8);
  } catch (e) {
    gteFailed = String((e && e.message) || e).indexOf('greater than or equal') >= 0;
  }
  assert(gteFailed, 'toBeGreaterThanOrEqual should fail when comparison is false');

  let lteFailed = false;
  try {
    expect(9).toBeLessThanOrEqual(8);
  } catch (e) {
    lteFailed = String((e && e.message) || e).indexOf('less than or equal') >= 0;
  }
  assert(lteFailed, 'toBeLessThanOrEqual should fail when comparison is false');
})();
`

	runNodeWithScript(t, script)
}
