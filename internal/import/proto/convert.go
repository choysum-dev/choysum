// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

import (
	"github.com/choysum-dev/choysum/internal/import/proto/importpb"
	importpkg "github.com/choysum-dev/choysum/pkg/import"
)

// ReportToProto converts a pkg/import Report to the wire shape.
func ReportToProto(report importpkg.Report) *importpb.ImportReport {
	out := &importpb.ImportReport{
		Profile:     string(report.Profile),
		Policy:      policyToProto(report.Policy),
		DryRun:      report.DryRun,
		Stats:       statsToProto(report.Stats),
		Messages:    messagesToProto(report.Messages),
		ArtifactRef: report.ArtifactRef,
	}
	if report.Meta != nil {
		out.Meta = &importpb.ImportReportMeta{
			Lang:        report.Meta.Lang,
			SourceRef:   report.Meta.SourceRef,
			TargetModel: report.Meta.TargetModel,
		}
	}
	return out
}

// ReportFromProto converts importpb.ImportReport to pkg/import Report.
func ReportFromProto(pb *importpb.ImportReport) importpkg.Report {
	if pb == nil {
		return importpkg.Report{}
	}
	var meta *importpkg.ReportMeta
	if pb.GetMeta() != nil {
		meta = &importpkg.ReportMeta{
			Lang:        pb.GetMeta().GetLang(),
			SourceRef:   pb.GetMeta().GetSourceRef(),
			TargetModel: pb.GetMeta().GetTargetModel(),
		}
	}
	return importpkg.Report{
		Profile:     importpkg.Profile(pb.GetProfile()),
		Policy:      policyFromProto(pb.GetPolicy()),
		DryRun:      pb.GetDryRun(),
		Stats:       statsFromProto(pb.GetStats()),
		Messages:    messagesFromProto(pb.GetMessages()),
		ArtifactRef: pb.GetArtifactRef(),
		Meta:        meta,
	}
}

func policyToProto(policy importpkg.Policy) importpb.ImportPolicy {
	switch policy {
	case importpkg.PolicyAtomic:
		return importpb.ImportPolicy_IMPORT_POLICY_ATOMIC
	case importpkg.PolicyStopKeep:
		return importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP
	case importpkg.PolicyBestEffort:
		return importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT
	default:
		return importpb.ImportPolicy_IMPORT_POLICY_UNSPECIFIED
	}
}

func policyFromProto(policy importpb.ImportPolicy) importpkg.Policy {
	switch policy {
	case importpb.ImportPolicy_IMPORT_POLICY_ATOMIC:
		return importpkg.PolicyAtomic
	case importpb.ImportPolicy_IMPORT_POLICY_STOP_KEEP:
		return importpkg.PolicyStopKeep
	case importpb.ImportPolicy_IMPORT_POLICY_BEST_EFFORT:
		return importpkg.PolicyBestEffort
	default:
		return importpkg.PolicyUnspecified
	}
}

func statsToProto(stats importpkg.Stats) *importpb.ImportStats {
	return &importpb.ImportStats{
		Total:   int32(stats.Total),
		Ok:      int32(stats.Ok),
		Error:   int32(stats.Error),
		Skip:    int32(stats.Skip),
		Warning: int32(stats.Warning),
	}
}

func statsFromProto(stats *importpb.ImportStats) importpkg.Stats {
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

func messagesToProto(messages []importpkg.Message) []*importpb.ImportMessage {
	if len(messages) == 0 {
		return nil
	}
	out := make([]*importpb.ImportMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, &importpb.ImportMessage{
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

func messagesFromProto(messages []*importpb.ImportMessage) []importpkg.Message {
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

func messageTypeToProto(t importpkg.MessageType) importpb.ImportMessageType {
	switch t {
	case importpkg.MessageError:
		return importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_ERROR
	case importpkg.MessageWarning:
		return importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_WARNING
	case importpkg.MessageSkip:
		return importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_SKIP
	default:
		return importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_UNSPECIFIED
	}
}

func messageTypeFromProto(t importpb.ImportMessageType) importpkg.MessageType {
	switch t {
	case importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_ERROR:
		return importpkg.MessageError
	case importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_WARNING:
		return importpkg.MessageWarning
	case importpb.ImportMessageType_IMPORT_MESSAGE_TYPE_SKIP:
		return importpkg.MessageSkip
	default:
		return importpkg.MessageType("")
	}
}
