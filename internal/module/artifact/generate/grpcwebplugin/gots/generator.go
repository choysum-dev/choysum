// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gots

import (
	"encoding/base64"
	"path/filepath"
	"strconv"
	"strings"

	xfmt "golang.org/x/exp/errors/fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Generator is the minimal pure-Go generator skeleton for target=ts.
// It supports top-level message, enum, and service schemas plus minimal cross-file imports.
type Generator struct{}

type symbolKind string

const (
	symbolMessage  symbolKind = "message"
	symbolEnum     symbolKind = "enum"
	symbolMapEntry symbolKind = "map_entry"
)

type symbol struct {
	Kind      symbolKind
	FullName  string
	TSName    string
	FileName  string
	Message   *descriptorpb.DescriptorProto
	Enum      *descriptorpb.EnumDescriptorProto
	MapFields map[string]*descriptorpb.FieldDescriptorProto
}

type renderContext struct {
	currentFile string
	symbols     map[string]*symbol
	imports     map[string]map[string]struct{}
	descFns     map[string]struct{}
}

type wktEntry struct {
	TypeName   string
	SchemaName string
	FileConst  string
}

var wktTypes = map[string]wktEntry{
	".google.protobuf.Empty":       {TypeName: "Empty", SchemaName: "EmptySchema", FileConst: "file_google_protobuf_empty"},
	".google.protobuf.Any":         {TypeName: "Any", SchemaName: "AnySchema", FileConst: "file_google_protobuf_any"},
	".google.protobuf.Timestamp":   {TypeName: "Timestamp", SchemaName: "TimestampSchema", FileConst: "file_google_protobuf_timestamp"},
	".google.protobuf.Duration":    {TypeName: "Duration", SchemaName: "DurationSchema", FileConst: "file_google_protobuf_duration"},
	".google.protobuf.FieldMask":   {TypeName: "FieldMask", SchemaName: "FieldMaskSchema", FileConst: "file_google_protobuf_field_mask"},
	".google.protobuf.DoubleValue": {TypeName: "DoubleValue", SchemaName: "DoubleValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.FloatValue":  {TypeName: "FloatValue", SchemaName: "FloatValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.Int64Value":  {TypeName: "Int64Value", SchemaName: "Int64ValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.UInt64Value": {TypeName: "UInt64Value", SchemaName: "UInt64ValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.Int32Value":  {TypeName: "Int32Value", SchemaName: "Int32ValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.UInt32Value": {TypeName: "UInt32Value", SchemaName: "UInt32ValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.BoolValue":   {TypeName: "BoolValue", SchemaName: "BoolValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.StringValue": {TypeName: "StringValue", SchemaName: "StringValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.BytesValue":  {TypeName: "BytesValue", SchemaName: "BytesValueSchema", FileConst: "file_google_protobuf_wrappers"},
	".google.protobuf.Value":       {TypeName: "Value", SchemaName: "ValueSchema", FileConst: "file_google_protobuf_struct"},
	".google.protobuf.Struct":      {TypeName: "Struct", SchemaName: "StructSchema", FileConst: "file_google_protobuf_struct"},
	".google.protobuf.ListValue":   {TypeName: "ListValue", SchemaName: "ListValueSchema", FileConst: "file_google_protobuf_struct"},
	".google.protobuf.NullValue":   {TypeName: "NullValue", SchemaName: "NullValueSchema", FileConst: "file_google_protobuf_struct"},
}

var wktFileByProtoName = map[string]string{
	"google/protobuf/empty.proto":      "file_google_protobuf_empty",
	"google/protobuf/any.proto":        "file_google_protobuf_any",
	"google/protobuf/timestamp.proto":  "file_google_protobuf_timestamp",
	"google/protobuf/duration.proto":   "file_google_protobuf_duration",
	"google/protobuf/field_mask.proto": "file_google_protobuf_field_mask",
	"google/protobuf/wrappers.proto":   "file_google_protobuf_wrappers",
	"google/protobuf/struct.proto":     "file_google_protobuf_struct",
}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	if req == nil {
		return nil, xfmt.Errorf("grpc target=ts generator: nil request")
	}
	if !isSupportedTargetTSParameter(req.GetParameter()) {
		return nil, xfmt.Errorf("grpc target=ts generator: unsupported parameter %q", req.GetParameter())
	}

	protoFiles := make(map[string]*descriptorpb.FileDescriptorProto, len(req.GetProtoFile()))
	symbols := make(map[string]*symbol)
	for _, fd := range req.GetProtoFile() {
		if fd == nil || strings.TrimSpace(fd.GetName()) == "" {
			continue
		}
		protoFiles[fd.GetName()] = fd
		collectSymbols(fd, symbols)
	}

	resp := &pluginpb.CodeGeneratorResponse{}
	for _, inName := range req.GetFileToGenerate() {
		fd, ok := protoFiles[inName]
		if !ok {
			return nil, xfmt.Errorf("grpc target=ts generator: missing descriptor for %q", inName)
		}
		content, err := renderFile(fd, symbols, protoFiles)
		if err != nil {
			return nil, err
		}
		outName := ProtoFileToTSFile(inName)
		resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    &outName,
			Content: &content,
		})
	}

	return resp, nil
}

func renderFile(fd *descriptorpb.FileDescriptorProto, symbols map[string]*symbol, protoFiles map[string]*descriptorpb.FileDescriptorProto) (string, error) {
	b, err := proto.Marshal(fd)
	if err != nil {
		return "", xfmt.Errorf("grpc target=ts generator: marshal descriptor %q: %w", fd.GetName(), err)
	}
	b64 := base64.StdEncoding.EncodeToString(b)

	fileConst := ProtoFileToFileConst(fd.GetName())
	ctx := &renderContext{
		currentFile: fd.GetName(),
		symbols:     symbols,
		imports:     make(map[string]map[string]struct{}),
		descFns:     make(map[string]struct{}),
	}
	ctx.descFns["fileDesc"] = struct{}{}

	var out strings.Builder
	out.WriteString("// @generated by choysum gots with parameter \"target=ts\"\n\n")
	out.WriteString("__CHOYSUM_IMPORTS_PLACEHOLDER__\n")
	out.WriteString("// File descriptor for ")
	out.WriteString(fd.GetName())
	out.WriteString("\n")
	out.WriteString("export const ")
	out.WriteString(fileConst)
	out.WriteString(": GenFile = fileDesc('")
	out.WriteString(b64)
	out.WriteString("'")
	depFileConsts := renderFileDependencies(ctx, fd, protoFiles)
	if len(depFileConsts) > 0 {
		sortStrings(depFileConsts)
		out.WriteString(", [")
		out.WriteString(strings.Join(depFileConsts, ", "))
		out.WriteString("]")
	}
	out.WriteString(");\n\n")

	for i, e := range fd.GetEnumType() {
		enumName := ProtoNameToExport(e.GetName())
		ctx.descFns["enumDesc"] = struct{}{}
		renderEnumDecl(&out, enumName, e)
		renderSchemaConst(&out, enumName+"Schema", "enumDesc", fileConst, []int{i})
	}

	for i, m := range fd.GetMessageType() {
		if m.GetOptions().GetMapEntry() {
			continue
		}
		renderMessageDecl(&out, ctx, fileConst, strings.TrimSpace(fd.GetPackage()), fd.GetSyntax(), nil, []int{i}, m)
	}

	for i, s := range fd.GetService() {
		ctx.descFns["serviceDesc"] = struct{}{}
		for _, m := range s.GetMethod() {
			_ = resolveSymbolType(ctx, m.GetInputType())
			_ = resolveSymbolType(ctx, m.GetOutputType())
		}
		serviceName := EscapeTSIdentifier(s.GetName())
		out.WriteString("// Service ")
		out.WriteString(serviceName)
		out.WriteString("\n")
		out.WriteString("export const ")
		out.WriteString(serviceName)
		out.WriteString(" = serviceDesc(")
		out.WriteString(fileConst)
		out.WriteString(", ")
		out.WriteString(strconv.Itoa(i))
		out.WriteString(");\n\n")
	}

	ctx.addImportType("@bufbuild/protobuf/codegenv2", "GenFile")
	descNames := make([]string, 0, len(ctx.descFns))
	for name := range ctx.descFns {
		descNames = append(descNames, name)
	}
	sortStrings(descNames)
	for _, name := range descNames {
		ctx.addImportValue("@bufbuild/protobuf/codegenv2", name)
	}

	imports := renderImports(ctx)
	body := strings.Replace(out.String(), "__CHOYSUM_IMPORTS_PLACEHOLDER__\n", imports+"\n", 1)
	return body, nil
}

func renderFileDependencies(ctx *renderContext, fd *descriptorpb.FileDescriptorProto, protoFiles map[string]*descriptorpb.FileDescriptorProto) []string {
	depSet := map[string]struct{}{}
	for _, dep := range fd.GetDependency() {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		if wktFileConst, ok := wktFileByProtoName[dep]; ok {
			ctx.addImportValue("@bufbuild/protobuf/wkt", wktFileConst)
			depSet[wktFileConst] = struct{}{}
			continue
		}
		if _, ok := protoFiles[dep]; !ok {
			continue
		}
		depConst := ProtoFileToFileConst(dep)
		ctx.addImportValue(relativeImportPath(fd.GetName(), dep), depConst)
		depSet[depConst] = struct{}{}
	}
	deps := make([]string, 0, len(depSet))
	for dep := range depSet {
		deps = append(deps, dep)
	}
	return deps
}

func renderEnumDecl(out *strings.Builder, enumTypeName string, e *descriptorpb.EnumDescriptorProto) {
	prefix := toScreamingSnakeCase(enumTypeName) + "_"
	out.WriteString("// Enum ")
	out.WriteString(enumTypeName)
	out.WriteString("\n")
	out.WriteString("export enum ")
	out.WriteString(enumTypeName)
	out.WriteString(" {\n")
	for _, v := range e.GetValue() {
		name := v.GetName()
		if strings.HasPrefix(name, prefix) {
			stripped := name[len(prefix):]
			if len(stripped) > 0 && stripped[0] >= 'A' && stripped[0] <= 'Z' {
				name = stripped
			}
		}
		out.WriteString("  ")
		out.WriteString(name)
		out.WriteString(" = ")
		out.WriteString(strconv.Itoa(int(v.GetNumber())))
		out.WriteString(",\n")
	}
	out.WriteString("}\n\n")
}

func renderSchemaConst(out *strings.Builder, constName, descFn, fileConst string, path []int) {
	out.WriteString("export const ")
	out.WriteString(constName)
	out.WriteString(" = ")
	out.WriteString(descFn)
	out.WriteString("(")
	out.WriteString(fileConst)
	for _, idx := range path {
		out.WriteString(", ")
		out.WriteString(strconv.Itoa(idx))
	}
	out.WriteString(");\n\n")
}

func renderMessageDecl(out *strings.Builder, ctx *renderContext, fileConst, pkg, fileSyntax string, parents []string, msgPath []int, m *descriptorpb.DescriptorProto) {
	if m.GetOptions().GetMapEntry() {
		return
	}

	ctx.descFns["messageDesc"] = struct{}{}
	ctx.addImportType("@bufbuild/protobuf", "Message")
	ctx.addImportType("@bufbuild/protobuf/codegenv2", "GenMessage")
	names := append(append([]string{}, parents...), m.GetName())
	msgName := NestedNamesToExport(names)
	fullName := msgName
	if pkg != "" {
		fullName = pkg + "." + strings.Join(names, ".")
	}

	out.WriteString("// Message ")
	out.WriteString(msgName)
	out.WriteString("\n")
	out.WriteString("export type ")
	out.WriteString(msgName)
	out.WriteString(" = Message<'")
	out.WriteString(fullName)
	out.WriteString("'> & {\n")
	for _, f := range m.GetField() {
		out.WriteString("  ")
		out.WriteString(ProtoFieldToTSField(f.GetName()))
		if isOptionalField(f, fileSyntax) {
			out.WriteString("?: ")
		} else {
			out.WriteString(": ")
		}
		out.WriteString(fieldTypeFor(ctx, f))
		out.WriteString(";\n")
	}
	out.WriteString("};\n\n")

	out.WriteString("export const ")
	out.WriteString(msgName)
	out.WriteString("Schema: GenMessage<")
	out.WriteString(msgName)
	out.WriteString("> = messageDesc(")
	out.WriteString(fileConst)
	for _, idx := range msgPath {
		out.WriteString(", ")
		out.WriteString(strconv.Itoa(idx))
	}
	out.WriteString(");\n\n")

	for enumIdx, e := range m.GetEnumType() {
		enumName := NestedNamesToExport(append(append([]string{}, names...), e.GetName()))
		ctx.descFns["enumDesc"] = struct{}{}
		renderEnumDecl(out, enumName, e)
		renderSchemaConst(out, enumName+"Schema", "enumDesc", fileConst, append(append([]int{}, msgPath...), enumIdx))
	}

	for nestedIdx, nested := range m.GetNestedType() {
		renderMessageDecl(out, ctx, fileConst, pkg, fileSyntax, names, append(append([]int{}, msgPath...), nestedIdx), nested)
	}
}

func isOptionalField(f *descriptorpb.FieldDescriptorProto, fileSyntax string) bool {
	if f == nil {
		return false
	}
	if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return false
	}
	if f.OneofIndex != nil {
		return true
	}
	if f.GetProto3Optional() {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(fileSyntax), "proto2") &&
		f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL {
		return true
	}
	return false
}

func fieldTypeFor(ctx *renderContext, f *descriptorpb.FieldDescriptorProto) string {
	if f == nil {
		return "unknown"
	}

	if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED && f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		if mapType, ok := mapFieldType(ctx, f.GetTypeName()); ok {
			return mapType
		}
	}

	base := scalarOrSymbolType(ctx, f)
	if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return base + "[]"
	}
	return base
}

func scalarOrSymbolType(ctx *renderContext, f *descriptorpb.FieldDescriptorProto) string {
	switch f.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "boolean"
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "Uint8Array"
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
		descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		return "number"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return "bigint"
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		return resolveSymbolType(ctx, f.GetTypeName())
	default:
		return "unknown"
	}
}

func mapFieldType(ctx *renderContext, typeName string) (string, bool) {
	s, ok := ctx.symbols[typeName]
	if !ok || s.Kind != symbolMapEntry {
		return "", false
	}
	key := s.MapFields["key"]
	val := s.MapFields["value"]
	if key == nil || val == nil {
		return "", false
	}
	return "Record<" + mapKeyTypeForRecordKey(key) + ", " + scalarOrSymbolType(ctx, val) + ">", true
}

func mapKeyTypeForRecordKey(key *descriptorpb.FieldDescriptorProto) string {
	if key == nil {
		return "string"
	}

	switch key.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		// JS object map keys are strings at runtime.
		return "string"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		// Record keys cannot be bigint; keep TS type valid and runtime-accurate.
		return "string"
	default:
		return "number"
	}
}

func resolveSymbolType(ctx *renderContext, typeName string) string {
	if s, ok := ctx.symbols[typeName]; ok && s.FileName == ctx.currentFile {
		return s.TSName
	}

	if w, ok := wktTypes[typeName]; ok {
		ctx.addImportType("@bufbuild/protobuf/wkt", w.TypeName)
		return w.TypeName
	}
	s, ok := ctx.symbols[typeName]
	if !ok {
		return "unknown"
	}
	if s.FileName == ctx.currentFile {
		return s.TSName
	}
	ctx.addImportType(relativeImportPath(ctx.currentFile, s.FileName), s.TSName)
	return s.TSName
}

func collectSymbols(fd *descriptorpb.FileDescriptorProto, symbols map[string]*symbol) {
	pkgPrefix := ""
	if pkg := strings.TrimSpace(fd.GetPackage()); pkg != "" {
		pkgPrefix = "." + pkg
	}
	for _, e := range fd.GetEnumType() {
		full := pkgPrefix + "." + e.GetName()
		symbols[full] = &symbol{Kind: symbolEnum, FullName: full, TSName: ProtoNameToExport(e.GetName()), FileName: fd.GetName(), Enum: e}
	}
	for _, m := range fd.GetMessageType() {
		collectMessageSymbol(fd.GetName(), pkgPrefix, nil, m, symbols)
	}
}

func collectMessageSymbol(fileName, pkgPrefix string, parents []string, m *descriptorpb.DescriptorProto, symbols map[string]*symbol) {
	names := append(append([]string{}, parents...), m.GetName())
	full := pkgPrefix + "." + strings.Join(names, ".")
	kind := symbolMessage
	mapFields := map[string]*descriptorpb.FieldDescriptorProto{}
	if m.GetOptions().GetMapEntry() {
		kind = symbolMapEntry
		for _, f := range m.GetField() {
			if f.GetName() == "key" || f.GetName() == "value" {
				mapFields[f.GetName()] = f
			}
		}
	}
	symbols[full] = &symbol{Kind: kind, FullName: full, TSName: NestedNamesToExport(names), FileName: fileName, Message: m, MapFields: mapFields}

	for _, e := range m.GetEnumType() {
		enumNames := append(append([]string{}, names...), e.GetName())
		enumFull := pkgPrefix + "." + strings.Join(enumNames, ".")
		symbols[enumFull] = &symbol{Kind: symbolEnum, FullName: enumFull, TSName: NestedNamesToExport(enumNames), FileName: fileName, Enum: e}
	}
	for _, nested := range m.GetNestedType() {
		collectMessageSymbol(fileName, pkgPrefix, names, nested, symbols)
	}
}

func relativeImportPath(currentProto, targetProto string) string {
	fromTS := ProtoFileToTSFile(currentProto)
	toTS := ProtoFileToTSFile(targetProto)
	rel, err := filepath.Rel(filepath.Dir(fromTS), toTS)
	if err != nil {
		rel = toTS
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, ".ts")
	if !strings.HasPrefix(rel, "./") && !strings.HasPrefix(rel, "../") {
		rel = "./" + rel
	}
	return rel
}

func (ctx *renderContext) addImportType(source, name string) {
	ctx.addImport(source, "type:"+name)
}

func (ctx *renderContext) addImportValue(source, name string) {
	ctx.addImport(source, "value:"+name)
}

func (ctx *renderContext) addImport(source, tagName string) {
	if source == "" || tagName == "" {
		return
	}
	if _, ok := ctx.imports[source]; !ok {
		ctx.imports[source] = map[string]struct{}{}
	}
	ctx.imports[source][tagName] = struct{}{}
}

func renderImports(ctx *renderContext) string {
	sources := make([]string, 0, len(ctx.imports))
	for s := range ctx.imports {
		sources = append(sources, s)
	}
	sortStrings(sources)

	var b strings.Builder
	for _, src := range sources {
		valueNames := make([]string, 0)
		typeNames := make([]string, 0)
		for tag := range ctx.imports[src] {
			if strings.HasPrefix(tag, "value:") {
				valueNames = append(valueNames, strings.TrimPrefix(tag, "value:"))
			}
			if strings.HasPrefix(tag, "type:") {
				typeNames = append(typeNames, strings.TrimPrefix(tag, "type:"))
			}
		}
		sortStrings(valueNames)
		sortStrings(typeNames)

		if len(typeNames) > 0 {
			b.WriteString("import type { ")
			b.WriteString(strings.Join(typeNames, ", "))
			b.WriteString(" } from '")
			b.WriteString(src)
			b.WriteString("';\n")
		}
		if len(valueNames) > 0 {
			b.WriteString("import { ")
			b.WriteString(strings.Join(valueNames, ", "))
			b.WriteString(" } from '")
			b.WriteString(src)
			b.WriteString("';\n")
		}
	}
	return b.String()
}

func sortStrings(items []string) {
	if len(items) < 2 {
		return
	}
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func hasTargetTS(parameter string) bool {
	for _, part := range strings.Split(parameter, ",") {
		part = strings.TrimSpace(part)
		if part == "target=ts" {
			return true
		}
	}
	return false
}

func isSupportedTargetTSParameter(parameter string) bool {
	if !hasTargetTS(parameter) {
		return false
	}

	for _, part := range strings.Split(parameter, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !isWhitelistedTargetTSParameterPart(part) {
			return false
		}
	}

	return true
}

func isWhitelistedTargetTSParameterPart(part string) bool {
	switch part {
	case "target=ts", "import_extension=none":
		return true
	default:
		return false
	}
}
