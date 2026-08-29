// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	recordwriter "github.com/choysum-dev/choysum/internal/import/writer/record"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func describeImportFields(ctx context.Context, runtimeScope scope.Scope, req *importpb.DescribeImportFieldsRequest) (*importpb.DescribeImportFieldsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	modelName := strings.TrimSpace(req.GetModel())
	if modelName == "" {
		return nil, status.Error(codes.InvalidArgument, "model is required")
	}

	session, ok := scope.SessionForScope(ctx, runtimeScope)
	if !ok || session == nil || session.DB == nil {
		return nil, status.Error(codes.Unavailable, "database session unavailable")
	}

	model, err := recordwriter.LookupModel(session.DB, modelName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "model lookup failed: %v", err)
	}

	defaults, err := describeDefaultImportFields(modelName)
	if err != nil {
		if impErr, ok := importpkg.AsError(err); ok && impErr.Code == importpkg.CodeModelNotFound {
			return nil, status.Error(codes.FailedPrecondition, impErr.Error())
		}
		return nil, status.Errorf(codes.Internal, "default import fields: %v", err)
	}

	fields, err := listDescribeImportModelFields(session.DB, model)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list fields: %v", err)
	}

	nodes := make([]*importpb.ImportFieldNode, 0, len(fields))
	for _, field := range fields {
		node, err := importFieldNode(session.DB, field)
		if err != nil || node == nil {
			continue
		}
		nodes = append(nodes, node)
	}

	return &importpb.DescribeImportFieldsResponse{
		Fields:        nodes,
		DefaultFields: append([]string(nil), defaults...),
	}, nil
}

var describeDefaultImportFields = recordwriter.DefaultImportFields

var listDescribeImportModelFields = recordwriter.ListFields

func importFieldNode(db *gorm.DB, field meta.Field) (*importpb.ImportFieldNode, error) {
	if shouldSkipImportField(&field) {
		return nil, nil
	}

	label := importFieldLabel(&field)
	node := &importpb.ImportFieldNode{
		Path:  field.Name,
		Label: label,
	}

	if fieldIsManyToOne(&field) {
		target, err := fieldRelationTarget(&field)
		if err != nil {
			return node, nil
		}
		relModel, err := recordwriter.LookupModel(db, target)
		if err != nil {
			return node, nil
		}
		relFields, err := recordwriter.ListFields(db, relModel)
		if err != nil {
			return node, nil
		}
		for _, relField := range relFields {
			if relField.Name != "Code" && relField.Name != "Name" {
				continue
			}
			if shouldSkipImportField(&relField) {
				continue
			}
			node.Children = append(node.Children, &importpb.ImportFieldNode{
				Path:  field.Name + "/" + relField.Name,
				Label: fmt.Sprintf("%s / %s", label, importFieldLabel(&relField)),
			})
		}
	}
	return node, nil
}

func shouldSkipImportField(field *meta.Field) bool {
	if field == nil {
		return true
	}
	name := strings.TrimSpace(field.Name)
	if name == "" || name == "Id" {
		return true
	}
	if field.IsReadonly {
		return true
	}
	ft := strings.ToLower(strings.TrimSpace(field.FieldType))
	switch ft {
	case "one2many", "one2manyref", "many2many", "many2manyref", "binary", "json", "jsonb":
		return true
	}
	if strings.EqualFold(strings.TrimSpace(field.Relation), "One2Many") ||
		strings.EqualFold(strings.TrimSpace(field.Relation), "Many2Many") {
		return true
	}
	return false
}

func importFieldLabel(field *meta.Field) string {
	if field == nil {
		return ""
	}
	if label := strings.TrimSpace(field.FieldString); label != "" {
		return label
	}
	return field.Name
}

func fieldIsManyToOne(field *meta.Field) bool {
	if field == nil {
		return false
	}
	ft := strings.TrimSpace(field.FieldType)
	return ft == "ManyToOne" || ft == "ManyToOneRef" || strings.EqualFold(field.Relation, "ManyToOne")
}

func fieldRelationTarget(field *meta.Field) (string, error) {
	target := strings.TrimSpace(field.RelationModel)
	if target == "" {
		if spec, err := field.GetResolvedSpec(); err == nil && spec != nil && spec.Structural.Relation != nil {
			if v, ok := spec.Structural.Relation["targetModel"].(string); ok {
				target = strings.TrimSpace(v)
			}
		}
	}
	if target == "" {
		return "", fmt.Errorf("relation target missing")
	}
	if !strings.Contains(target, ".") {
		return "", fmt.Errorf("relation target must be app.Model")
	}
	return target, nil
}
