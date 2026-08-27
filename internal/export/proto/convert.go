// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

import (
	"github.com/choysum-dev/choysum/internal/export/proto/exportpb"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// ReportToProto converts a pkg/import Report to the export wire shape.
func ReportToProto(report importpkg.Report) *exportpb.ExportReport {
	out := &exportpb.ExportReport{
		Profile:     string(report.Profile),
		DryRun:      report.DryRun,
		Stats:       statsToProto(report.Stats),
		Messages:    messagesToProto(report.Messages),
		ArtifactRef: report.ArtifactRef,
	}
	if report.Meta != nil {
		out.Meta = &exportpb.ExportReportMeta{
			Lang:        report.Meta.Lang,
			TargetModel: report.Meta.TargetModel,
		}
	}
	return out
}

// ReportFromProto converts exportpb.ExportReport to pkg/import Report.
func ReportFromProto(pb *exportpb.ExportReport) importpkg.Report {
	if pb == nil {
		return importpkg.Report{}
	}
	var meta *importpkg.ReportMeta
	if pb.GetMeta() != nil {
		meta = &importpkg.ReportMeta{
			Lang:        pb.GetMeta().GetLang(),
			TargetModel: pb.GetMeta().GetTargetModel(),
		}
	}
	return importpkg.Report{
		Profile:     importpkg.Profile(pb.GetProfile()),
		Policy:      importpkg.PolicyUnspecified,
		DryRun:      pb.GetDryRun(),
		Stats:       statsFromProto(pb.GetStats()),
		Messages:    messagesFromProto(pb.GetMessages()),
		ArtifactRef: pb.GetArtifactRef(),
		Meta:        meta,
	}
}

func statsToProto(stats importpkg.Stats) *exportpb.ExportStats {
	return &exportpb.ExportStats{
		Total:   int32(stats.Total),
		Ok:      int32(stats.Ok),
		Error:   int32(stats.Error),
		Skip:    int32(stats.Skip),
		Warning: int32(stats.Warning),
	}
}

func statsFromProto(stats *exportpb.ExportStats) importpkg.Stats {
	if stats == nil {
		return importpkg.Stats{}
	}
	return importpkg.Stats{
		Total:   int(stats.GetTotal()),
		Ok:      int(stats.GetOk()),
		Error:   int(stats.GetError()),
		Skip:    int(stats.GetSkip()),
		Warning: int(stats.GetWarning()),
	}
}

func messagesToProto(messages []importpkg.Message) []*exportpb.ExportMessage {
	if len(messages) == 0 {
		return nil
	}
	out := make([]*exportpb.ExportMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, &exportpb.ExportMessage{
			Type:      messageTypeToProto(msg.Type),
			Row:       int32(msg.Row),
			Field:     msg.Field,
			Code:      msg.Code,
			Text:      msg.Text,
			RecordRef: msg.RecordRef,
		})
	}
	return out
}

func messagesFromProto(messages []*exportpb.ExportMessage) []importpkg.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]importpkg.Message, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		out = append(out, importpkg.Message{
			Type:      messageTypeFromProto(msg.GetType()),
			Row:       int(msg.GetRow()),
			Field:     msg.GetField(),
			Code:      msg.GetCode(),
			Text:      msg.GetText(),
			RecordRef: msg.GetRecordRef(),
		})
	}
	return out
}

func messageTypeToProto(t importpkg.MessageType) exportpb.ExportMessageType {
	switch t {
	case importpkg.MessageError:
		return exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_ERROR
	case importpkg.MessageWarning:
		return exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_WARNING
	case importpkg.MessageSkip:
		return exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_SKIP
	default:
		return exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_UNSPECIFIED
	}
}

func messageTypeFromProto(t exportpb.ExportMessageType) importpkg.MessageType {
	switch t {
	case exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_ERROR:
		return importpkg.MessageError
	case exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_WARNING:
		return importpkg.MessageWarning
	case exportpb.ExportMessageType_EXPORT_MESSAGE_TYPE_SKIP:
		return importpkg.MessageSkip
	default:
		return importpkg.MessageType("")
	}
}
