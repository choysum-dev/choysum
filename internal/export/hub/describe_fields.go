// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package hub

import (
	"context"
	"fmt"
	"strings"

	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	recordreader "github.com/choysum-dev/choysum/internal/export/reader/record"
	importwriter "github.com/choysum-dev/choysum/internal/import/writer/record"
	exportpkg "github.com/choysum-dev/choysum/pkg/export"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func describeFields(ctx context.Context, runtimeScope scope.Scope, req *exportpb.DescribeFieldsRequest) (*exportpb.DescribeFieldsResponse, error) {
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

	model, err := importwriter.LookupModel(session.DB, modelName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "model lookup failed: %v", err)
	}

	defaults, err := describeDefaultExportFields(modelName)
	if err != nil {
		if expErr, ok := exportpkg.AsError(err); ok && expErr.Code == exportpkg.CodeModelNotFound {
			return nil, status.Error(codes.FailedPrecondition, expErr.Error())
		}
		return nil, status.Errorf(codes.Internal, "default export fields: %v", err)
	}

	fields, err := listDescribeModelFields(session.DB, model)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list fields: %v", err)
	}

	nodes := make([]*exportpb.ExportFieldNode, 0, len(fields))
	for _, field := range fields {
		node, err := exportFieldNode(session.DB, field)
		if err != nil || node == nil {
			continue
		}
		nodes = append(nodes, node)
	}

	return &exportpb.DescribeFieldsResponse{
		Fields:        nodes,
		DefaultFields: append([]string(nil), defaults...),
	}, nil
}

var describeDefaultExportFields = recordreader.DefaultExportFields

var listDescribeModelFields = importwriter.ListFields

func exportFieldNode(db *gorm.DB, field meta.Field) (*exportpb.ExportFieldNode, error) {
	if shouldSkipExportField(&field) {
		return nil, nil
	}

	label := fieldLabel(&field)
	node := &exportpb.ExportFieldNode{
		Path:  field.Name,
		Label: label,
	}

	if importwriterFieldIsManyToOne(&field) {
		target, err := importwriterFieldRelationTarget(&field)
		if err != nil {
			return node, nil
		}
		relModel, err := importwriter.LookupModel(db, target)
		if err != nil {
			return node, nil
		}
		relFields, err := importwriter.ListFields(db, relModel)
		if err != nil {
			return node, nil
		}
		for _, relField := range relFields {
			if relField.Name != "Code" && relField.Name != "Name" {
				continue
			}
			node.Children = append(node.Children, &exportpb.ExportFieldNode{
				Path:  field.Name + "/" + relField.Name,
				Label: fmt.Sprintf("%s / %s", label, fieldLabel(&relField)),
			})
		}
	}
	return node, nil
}

func shouldSkipExportField(field *meta.Field) bool {
	if field == nil {
		return true
	}
	name := strings.TrimSpace(field.Name)
	if name == "" || name == "Id" {
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

func fieldLabel(field *meta.Field) string {
	if field == nil {
		return ""
	}
	if label := strings.TrimSpace(field.FieldString); label != "" {
		return label
	}
	return field.Name
}

func importwriterFieldIsManyToOne(field *meta.Field) bool {
	if field == nil {
		return false
	}
	ft := strings.TrimSpace(field.FieldType)
	return ft == "ManyToOne" || ft == "ManyToOneRef" || strings.EqualFold(field.Relation, "ManyToOne")
}

func importwriterFieldRelationTarget(field *meta.Field) (string, error) {
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
