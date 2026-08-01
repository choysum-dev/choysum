// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gots

import "testing"

func TestProtoFileToTSFile(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "error.proto", want: "error_pb.ts"},
		{in: "auth/user.proto", want: "auth/user_pb.ts"},
		{in: "google/protobuf/struct.proto", want: "google/protobuf/struct_pb.ts"},
		{in: "auth.proto.backup", want: "auth.proto.backup_pb.ts"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := ProtoFileToTSFile(tc.in); got != tc.want {
				t.Fatalf("ProtoFileToTSFile(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestProtoFileToFileConst(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "error.proto", want: "file_error"},
		{in: "auth_service.proto", want: "file_auth_service"},
		{in: "auth/session.proto", want: "file_auth_session"},
		{in: "user-profile.proto", want: "file_user_profile"},
		{in: "---.proto", want: "file_file"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := ProtoFileToFileConst(tc.in); got != tc.want {
				t.Fatalf("ProtoFileToFileConst(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestProtoFieldToTSField(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "_"},
		{in: "___", want: "_"},
		{in: "error_id", want: "errorId"},
		{in: "grpc_code", want: "grpcCode"},
		{in: "return_fields", want: "returnFields"},
		{in: "role_data", want: "roleData"},
		{in: "userId", want: "userId"},
		{in: "default", want: "default_"},
		{in: "1st_value", want: "_1stValue"},
		{in: " _a _b ", want: "aB"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := ProtoFieldToTSField(tc.in); got != tc.want {
				t.Fatalf("ProtoFieldToTSField(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNestedNamesToExport(t *testing.T) {
	cases := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "metadata entry", parts: []string{"ErrorInfo", "MetadataEntry"}, want: "ErrorInfo_MetadataEntry"},
		{name: "simple nested", parts: []string{"User", "Profile"}, want: "User_Profile"},
		{name: "all blank", parts: []string{"", "___"}, want: "_"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NestedNamesToExport(tc.parts); got != tc.want {
				t.Fatalf("NestedNamesToExport(%v) = %q, want %q", tc.parts, got, tc.want)
			}
		})
	}
}

func TestProtoNameToExport(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "_"},
		{in: "___", want: "_"},
		{in: "@@@", want: "_"},
		{in: "_bad_", want: "Bad"},
		{in: "Application_Browse_Req", want: "ApplicationBrowseReq"},
		{in: "null_value", want: "NullValue"},
		{in: "sessionService", want: "SessionService"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := ProtoNameToExport(tc.in); got != tc.want {
				t.Fatalf("ProtoNameToExport(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestToScreamingSnakeCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "   ", want: ""},
		{in: "InitializationState", want: "INITIALIZATION_STATE"},
		{in: "HTTPServer", want: "HTTP_SERVER"},
		{in: "\tÅngströmValue\n", want: "ÅNGSTRÖM_VALUE"},
		{in: "MyHTTP", want: "MY_HTTP"},
		{in: "MyHTMLSomething", want: "MY_HTML_SOMETHING"},
		{in: "v2HTTPServer", want: "V2_HTTP_SERVER"},
		{in: "alllowercase", want: "ALLLOWERCASE"},
		{in: "XMLParser3", want: "XML_PARSER3"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := toScreamingSnakeCase(tc.in); got != tc.want {
				t.Fatalf("toScreamingSnakeCase(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestEscapeTSIdentifier(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "hello", want: "hello"},
		{in: "", want: "_"},
		{in: "1abc", want: "_1abc"},
		{in: "let", want: "let_"},
		{in: "break", want: "break_"},
		{in: "__", want: "_"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := EscapeTSIdentifier(tc.in); got != tc.want {
				t.Fatalf("EscapeTSIdentifier(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestUpperFirst(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "hello", want: "Hello"},
		{in: "", want: ""},
		{in: "h", want: "H"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := upperFirst(tc.in); got != tc.want {
				t.Fatalf("upperFirst(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSplitIdentifierParts(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{in: "a_b", want: []string{"a", "b"}},
		{in: "", want: nil},
		{in: "_ a _ b ", want: []string{"a", "b"}},
		{in: "___", want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := splitIdentifierParts(tc.in)
			if tc.want == nil && got != nil {
				t.Fatalf("splitIdentifierParts(%q) = %v, want nil", tc.in, got)
				return
			}
			if tc.want != nil && got == nil {
				t.Fatalf("splitIdentifierParts(%q) = nil, want %v", tc.in, tc.want)
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("splitIdentifierParts(%q) = %v, want %v", tc.in, got, tc.want)
				return
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("splitIdentifierParts(%q) = %v, want %v", tc.in, got, tc.want)
					return
				}
			}
		})
	}
}

func TestHasTargetTS(t *testing.T) {
	if !hasTargetTS("target=ts") {
		t.Fatalf("expected target=ts to be recognized")
	}
	if !hasTargetTS("opt=a,target=ts,import_extension=none") {
		t.Fatalf("expected mixed params to include target=ts")
	}
	if hasTargetTS("target=js") {
		t.Fatalf("expected target=js to be rejected")
	}
}

func TestIsSupportedTargetTSParameter(t *testing.T) {
	if !isSupportedTargetTSParameter("target=ts") {
		t.Fatalf("expected target=ts to be supported")
	}
	if !isSupportedTargetTSParameter("target=ts,import_extension=none") {
		t.Fatalf("expected import_extension=none to remain supported")
	}
	if !isSupportedTargetTSParameter("import_extension=none,target=ts") {
		t.Fatalf("expected supported parts in any order to remain supported")
	}

	rejectCases := []string{
		"target=js",
		"target=ts,target=dts",
		"target=ts,opt=a",
		"target=ts,js_import_style=legacy_commonjs",
		"target=ts,import_extension=js",
		"target=ts,import_extension=ts",
		"target=ts,json_types=true",
		"target=ts,valid_types=foo.bar.Baz",
	}
	for _, parameter := range rejectCases {
		if isSupportedTargetTSParameter(parameter) {
			t.Fatalf("expected parameter to be rejected: %q", parameter)
		}
	}
}
