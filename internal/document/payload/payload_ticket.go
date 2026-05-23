// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package payload

import (
	"encoding/json"
	"strings"
)

type payloadWriteTicket struct {
	UploadID        string `json:"uploadId"`
	CompanyID       string `json:"companyId"`
	UserID          string `json:"userId"`
	ActiveCompanyID string `json:"activeCompanyId"`
	Status          string `json:"status"`
	ExpiresAt       string `json:"expiresAt"`
}

type payloadReadTicket struct {
	AttachmentBindingID string `json:"attachmentBindingId"`
	AttachmentContentID string `json:"attachmentContentId"`
	StoredContentID     string `json:"storedContentId"`
}

func parsePayloadWriteTicket(raw string) (payloadWriteTicket, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return payloadWriteTicket{}, InvalidArgument("payload write ticket is required")
	}

	var ticket payloadWriteTicket
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		return payloadWriteTicket{}, InvalidArgumentWrap("parse payload write ticket", err)
	}

	ticket.UploadID = strings.TrimSpace(ticket.UploadID)
	ticket.CompanyID = strings.TrimSpace(ticket.CompanyID)
	ticket.UserID = strings.TrimSpace(ticket.UserID)
	ticket.ActiveCompanyID = strings.TrimSpace(ticket.ActiveCompanyID)
	ticket.Status = strings.TrimSpace(ticket.Status)
	ticket.ExpiresAt = strings.TrimSpace(ticket.ExpiresAt)

	if ticket.UploadID == "" {
		return payloadWriteTicket{}, InvalidArgument("payload write ticket missing uploadId")
	}

	return ticket, nil
}

func parsePayloadReadTicket(raw string) (payloadReadTicket, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return payloadReadTicket{}, InvalidArgument("payload read ticket is required")
	}

	var ticket payloadReadTicket
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		return payloadReadTicket{}, InvalidArgumentWrap("parse payload read ticket", err)
	}

	ticket.AttachmentBindingID = strings.TrimSpace(ticket.AttachmentBindingID)
	ticket.AttachmentContentID = strings.TrimSpace(ticket.AttachmentContentID)
	ticket.StoredContentID = strings.TrimSpace(ticket.StoredContentID)

	if ticket.StoredContentID == "" {
		return payloadReadTicket{}, InvalidArgument("payload read ticket missing storedContentId")
	}

	return ticket, nil
}
