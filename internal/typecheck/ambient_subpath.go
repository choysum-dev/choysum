// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"fmt"
	"strings"
)

// SubpathStubOverlay returns the relative ambient path and declarations for
// subpath imports that lack individual .d.ts coverage (locales, assets, etc.).
func SubpathStubOverlay() (relPath, content string) {
	return "subpath-stubs.d.ts", buildSubpathStubs()
}

func buildSubpathStubs() string {
	stubs := []string{
		"dayjs/locale/*",
		"dayjs/plugin/*",
		"element-plus/es/locale/lang/*",
		"@element-plus/icons-vue",
		"nprogress",
		"vitest",
		"@vue/test-utils",
	}
	var b strings.Builder
	b.WriteString("// Ambient declarations for subpath imports without individual types.\n")
	for _, mod := range stubs {
		fmt.Fprintf(&b, "declare module %q;\n", mod)
	}
	// ImportMeta / CSS/SVG asset modules live in the vite/client ambient only —
	// declaring them here would duplicate global index signatures and module shapes.
	b.WriteString(`
declare module "@bufbuild/protobuf/codegenv2" {
  export type Message = any;
  export type GenFile = any;
  export type GenMessage<T = any> = any;
  export type GenService<T = any> = any;
  export const fileDesc: any;
  export const messageDesc: any;
  export const enumDesc: any;
  export const serviceDesc: any;
}

declare module "@bufbuild/protobuf/wkt" {
  export type Value = any;
  export const EmptySchema: any;
  export const ListValueSchema: any;
  export const NullValue: any;
  export const StructSchema: any;
  export const ValueSchema: any;
}

declare module "kysely/helpers/postgres" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "kysely/helpers/mysql" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "kysely/helpers/sqlite" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "kysely/helpers/mssql" {
  export const jsonArrayFrom: any;
  export const jsonObjectFrom: any;
}

declare module "echarts/core" {
  export const use: (...args: any[]) => void;
}

declare module "echarts/charts" {
  export const BarChart: any;
  export const LineChart: any;
  export const PieChart: any;
}

declare module "echarts/components" {
  export const TitleComponent: any;
  export const TooltipComponent: any;
  export const LegendComponent: any;
  export const GridComponent: any;
}

declare module "echarts/renderers" {
  export const SVGRenderer: any;
}

declare module "element-plus/es/components/table-v2/src/row" {
	export type RowEventHandlerParams = any;
}

declare module "element-plus/es/components/table-v2/src/types" {
	export type RowEventHandlerParams = import("element-plus/es/components/table-v2/src/row").RowEventHandlerParams;
	export type KeyType = string | number;
}

declare module "fast-deep-equal" {
	export default function equal(a: any, b: any): boolean;
}

declare module "node:fs" {
	export function readFileSync(path: string, encoding?: string): string;
}

declare module "node:path" {
	export function resolve(...paths: string[]): string;
}
`)
	return b.String()
}
