package services

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/FreeCodeUserJack/Parley/pkg/domain"
	"github.com/FreeCodeUserJack/Parley/pkg/repository"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/context_utils"
	"github.com/FreeCodeUserJack/Parley/pkg/utils/rest_errors"
	"github.com/FreeCodeUserJack/Parley/tools/logger"
	"github.com/google/uuid"
)

type AgreementServiceInterface interface {
	NewAgreement(context.Context, domain.Agreement) (*domain.Agreement, rest_errors.RestError)
	CloseAgreement(context.Context, string, string, string, string, string) (string, rest_errors.RestError)
	UpdateAgreement(context.Context, domain.Agreement, string, string) (*domain.Agreement, rest_errors.RestError)
	GetAgreement(context.Context, string) (*domain.Agreement, rest_errors.RestError)
	SearchAgreements(context.Context, string, string) ([]domain.Agreement, rest_errors.RestError)
	AddUserToAgreement(context.Context, string, string) (string, rest_errors.RestError)
	RemoveUserFromAgreement(context.Context, string, string) (string, rest_errors.RestError)
	SetDeadline(context.Context, string, domain.Deadline, string, string) (*domain.Agreement, rest_errors.RestError)
	DeleteDeadline(context.Context, string, string, string) (*domain.Agreement, rest_errors.RestError)
	ActionAndNotification(context.Context, domain.Notification, string, string) (*domain.Notification, rest_errors.RestError)
	RespondAgreementChange(context.Context, domain.Notification) (*domain.Agreement, rest_errors.RestError)
	GetAgreementEventResponses(context.Context, string) ([]domain.EventResponse, rest_errors.RestError)
	InviteUsersToEvent(context.Context, string, []string) (string, rest_errors.RestError)
	RespondEventInvite(context.Context, string, domain.EventResponse) (*domain.EventResponse, rest_errors.RestError)
}

type agreementService struct {
	AgreementRepository        repository.AgreementRepositoryInterface
	AgreementArchiveRepository repository.AgreementArchiveRepositoryInterface
	NotificationRepository     repository.NotificationRepositoryInterface
}

func NewAgreementService(agreementRepo repository.AgreementRepositoryInterface, agreementArchiveRepo repository.AgreementArchiveRepositoryInterface, notificationRepo repository.NotificationRepositoryInterface) AgreementServiceInterface {
	return &agreementService{
		AgreementRepository:        agreementRepo,
		AgreementArchiveRepository: agreementArchiveRepo,
		NotificationRepository:     notificationRepo,
	}
}

func (a agreementService) NewAgreement(ctx context.Context, agreement domain.Agreement) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service NewAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	if !agreement.Validate() {
		logger.Error("agreement failed validation", errors.New("request agreement is not valid"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("req failed validation")
	}

	// Sanitize Data
	agreement.Sanitize()

	// Add UUID
	auuid := uuid.NewString()
	agreement.Id = auuid

	// Add CreateTime/UpdateTime
	currTime := time.Now().UTC()
	agreement.CreateDateTime = currTime
	agreement.LastUpdateDateTime = currTime
	agreement.AgreementDeadline.LastUpdateDatetime = currTime

	if agreement.AgreementDeadline.NotifyDateTime.IsZero() {
		agreement.AgreementDeadline.NotifyDateTime = agreement.AgreementDeadline.DeadlineDateTime.Add(time.Hour * -24).UTC()
	}

	// Initialize slices
	if agreement.Type != "event" {
		agreement.InvitedParticipants = []string{}
	}

	agreement.RequestedParticipants = []string{}
	agreement.PendingRemovalParticipants = []string{}
	agreement.PendingLeaveParticipants = []string{}
	agreement.AgreementAccept = []string{}
	agreement.AgreementDecline = []string{}
	agreement.EventResponses = []string{}

	logger.Info("agreement service NewAgreement end", context_utils.GetTraceAndClientIds(ctx)...)
	if agreement.Type == "event" {
		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.InvitedParticipants); i++ {
			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s invites you to '%s' agreement", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.InvitedParticipants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "notifyEventInvite",
				Action:           "requires_response",
			})
		}
		return a.AgreementRepository.NewEventAgreement(ctx, agreement, notifications)
	}
	return a.AgreementRepository.NewAgreement(ctx, agreement)
}

func (a agreementService) CloseAgreement(ctx context.Context, id, completionKey, completionVal, typeKey, typeVal string) (string, rest_errors.RestError) {
	logger.Info("agreement service CloseAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	//Sanitize the id string
	id = html.EscapeString(id)
	completionKey = strings.TrimSpace(html.EscapeString(completionKey))
	completionVal = strings.TrimSpace(html.EscapeString(completionVal))
	typeKey = strings.TrimSpace(html.EscapeString(typeKey))
	typeVal = strings.TrimSpace(html.EscapeString(typeVal))

	if completionKey != "completion" || completionVal != "completed" && completionVal != "retired" {
		logger.Error(fmt.Sprintf("agreement service CloseAgreement - improper completion key/val: %s %s %s", id, completionKey, completionVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("improper completion key/val: " + completionKey + "/" + completionVal)
	}

	if typeKey != "type" || typeVal != "solo" && typeVal != "directed" && typeVal != "collaborative" {
		logger.Error(fmt.Sprintf("agreement service CloseAgreement - improper type key/val: %s %s %s", id, typeKey, typeVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("improper type key/val: " + typeKey + "/" + typeVal)
	}

	// Get current agreement
	agreement, getErr := a.AgreementRepository.GetAgreement(ctx, id)
	if getErr != nil {
		logger.Error("agreement service CloseAgreement - could not get agreement: "+id, getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewInternalServerError("error trying to retrieve agreement: "+id, errors.New("database error"))
	}

	// Check if agreement already closed
	if agreement.Status == "retired" || agreement.Status == "completed" {
		logger.Error(fmt.Sprintf("agreement service CloseAgreement - agreement already closed: %v", agreement), getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("agreement already closed: " + id)
	}

	// Archive Agreement
	if typeVal != "collaborative" {
		agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, id, "deleted", "agreement was closed", nil)
		if archiveErr == nil {
			go func() {
				a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
			}()
		}
	}

	logger.Info("agreement service CloseAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	if typeVal == "solo" || len(agreement.Participants) < 2 {
		return a.AgreementRepository.CloseAgreement(ctx, id, completionVal)
	} else if typeVal == "directed" {
		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s %s '%s' agreement", agreement.CreatorName, completionVal, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "notifyFinish",
				Action:           "updateFinish",
			})
		}
		return a.AgreementRepository.CloseAgreementDirected(ctx, id, completionVal, notifications)
	} else {
		// TODO for collborative notification / put in awaiting collaboration + new agreement state
		updatedAgreement := *agreement
		updatedAgreement.Status = completionVal
		agreement.Status = "awaitingConfirmation"
		agreement.UpdatedAgreement = &updatedAgreement
		agreement.UpdatedAgreement.UpdatedAgreement = nil

		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s wants to %s '%s' agreement, please respond", agreement.CreatorName, completionVal, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "requires_response",
				Action:           "updateClose",
			})
		}
		return a.AgreementRepository.CollaborativeUpdateAgreementNotifications(ctx, *agreement, notifications)
	}
}

func (a agreementService) UpdateAgreement(ctx context.Context, agreement domain.Agreement, typeKey, typeVal string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service UpdateAgreement called", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize fields
	agreement.Sanitize()

	typeKey = strings.TrimSpace(html.EscapeString(typeKey))
	typeVal = strings.TrimSpace(html.EscapeString(typeVal))

	if typeKey != "type" || typeVal != "solo" && typeVal != "directed" && typeVal != "collaborative" {
		logger.Error(fmt.Sprintf("agreement service UpdateAgreement - improper type key/val: %s %s", typeKey, typeVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("improper type key/val: " + typeKey + "/" + typeVal)
	}

	// Get Existing Agreement and update fields that are different
	currTime := time.Now().UTC()
	agreement.LastUpdateDateTime = currTime
	agreement.AgreementDeadline.LastUpdateDatetime = currTime

	savedAgreement, getErr := a.GetAgreement(ctx, agreement.Id)
	if getErr != nil {
		return nil, getErr
	}

	// Check if agreement already closed
	if agreement.Status == "retired" || agreement.Status == "completed" {
		logger.Error(fmt.Sprintf("agreement service UpdateAgreement - agreement already closed: %v", agreement), getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("agreement already closed: " + agreement.Id)
	}

	// Check which fields need to be set
	if agreement.Title == "" {
		agreement.Title = savedAgreement.Title
	}
	if agreement.Description == "" {
		agreement.Description = savedAgreement.Description
	}
	if len(agreement.Participants) == 0 {
		agreement.Participants = savedAgreement.Participants
	}
	if agreement.AgreementDeadline.DeadlineDateTime.IsZero() {
		agreement.AgreementDeadline = savedAgreement.AgreementDeadline
	}
	if agreement.Status == "" {
		agreement.Status = savedAgreement.Status
	}
	if agreement.Public == "" {
		agreement.Public = savedAgreement.Public
	}
	if len(agreement.Tags) == 0 {
		agreement.Tags = savedAgreement.Tags
	}
	if agreement.Location == "" {
		agreement.Location = savedAgreement.Location
	}

	agreement.Type = savedAgreement.Type
	agreement.CreateDateTime = savedAgreement.CreateDateTime
	agreement.CreatedBy = savedAgreement.CreatedBy
	agreement.CreatorName = savedAgreement.CreatorName
	agreement.Participants = savedAgreement.Participants
	agreement.InvitedParticipants = savedAgreement.InvitedParticipants
	agreement.RequestedParticipants = savedAgreement.RequestedParticipants
	agreement.PendingRemovalParticipants = savedAgreement.PendingRemovalParticipants
	agreement.PendingLeaveParticipants = savedAgreement.PendingLeaveParticipants
	agreement.AgreementAccept = savedAgreement.AgreementAccept
	agreement.AgreementDecline = savedAgreement.AgreementDecline
	agreement.UpdatedAgreement = savedAgreement.UpdatedAgreement

	// Archive Agreement Changes
	if typeVal != "collaborative" {
		agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreement.Id, "modified", "agreement was modified", savedAgreement)
		if archiveErr == nil {
			go func() {
				a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
			}()
		}
	}

	logger.Info("agreement service UpdateAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	if typeVal == "solo" || len(agreement.Participants) < 2 {
		return a.AgreementRepository.UpdateAgreement(ctx, agreement)
	} else if typeVal == "directed" {
		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s updated '%s' agreement", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "notifyUpdate",
				Action:           "update",
			})
		}
		return a.AgreementRepository.UpdateAgreementNotifications(ctx, agreement, notifications)
	} else { // TODO collaborative
		savedAgreement.Status = "awaitingConfirmation"
		savedAgreement.UpdatedAgreement = &agreement

		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s updated '%s' agreement, please respond", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "requires_response",
				Action:           "update",
			})
		}
		return a.AgreementRepository.UpdateAgreementNotifications(ctx, *savedAgreement, notifications)
	}
}

// func removeCreator(input []string, id string) []string {
// 	res := []string{}

// 	for _, el := range input {
// 		if el != id {
// 			res = append(res, el)
// 		}
// 	}

// 	return res
// }

func (a agreementService) GetAgreement(ctx context.Context, id string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service GetAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize the id string
	id = html.EscapeString(id)

	logger.Info("agreement service GetAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.GetAgreement(ctx, id)
}

func (a agreementService) SearchAgreements(ctx context.Context, key string, val string) ([]domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service SearchAgreements start", context_utils.GetTraceAndClientIds(ctx)...)

	// Cannot search for all agreements
	if key == "" || val == "" {
		return nil, rest_errors.NewBadRequestError("key/val cannot be empty")
	}

	// Sanitize key + val
	key = strings.TrimSpace(html.EscapeString(key))
	val = strings.TrimSpace(html.EscapeString(val))

	logger.Info("agreement service SearchAgreements finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.SearchAgreements(ctx, key, val)
}

func (a agreementService) AddUserToAgreement(ctx context.Context, agreementId string, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement service AddUserToAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and friendId
	agreementId = html.EscapeString(agreementId)
	friendId = html.EscapeString(friendId)

	// Archive Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service AddUserToAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.AddUserToAgreement(ctx, agreementId, friendId)
}

func (a agreementService) RemoveUserFromAgreement(ctx context.Context, agreementId, friendId string) (string, rest_errors.RestError) {
	logger.Info("agreement service RemoveUserFromAgreement start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and friendId
	agreementId = html.EscapeString(agreementId)
	friendId = html.EscapeString(friendId)

	// Archive Changes
	agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
	if archiveErr == nil {
		go func() {
			a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
		}()
	}

	logger.Info("agreement service RemoveUserFromAgreement finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.RemoveUserFromAgreement(ctx, agreementId, friendId)
}

func (a agreementService) SetDeadline(ctx context.Context, agreementId string, deadline domain.Deadline, typeKey, typeVal string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service SetDeadline start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and deadline instance
	agreementId = html.EscapeString(agreementId)
	deadline.Sanitize()
	typeKey = strings.TrimSpace(html.EscapeString(typeKey))
	typeVal = strings.TrimSpace(html.EscapeString(typeVal))

	if typeKey != "type" || typeVal != "solo" && typeVal != "directed" && typeVal != "collaborative" {
		logger.Error(fmt.Sprintf("agreement service SetDeadline - improper type key/val: %s %s", typeKey, typeVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("improper type key/val: " + typeKey + "/" + typeVal)
	}

	// Get Existing Agreement
	agreement, getErr := a.GetAgreement(ctx, agreementId)
	if getErr != nil {
		return nil, getErr
	}

	// Check if agreement already closed
	if agreement.Status == "retired" || agreement.Status == "completed" {
		logger.Error(fmt.Sprintf("agreement service SetDeadline - agreement already closed: %v", agreement), getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("agreement already closed: " + agreementId)
	}

	// Archive Changes
	if typeVal != "collaborative" {
		agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
		if archiveErr == nil {
			go func() {
				a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
			}()
		}
	}

	// Check Nullable fields
	if deadline.NotifyDateTime.IsZero() {
		deadline.NotifyDateTime = deadline.DeadlineDateTime.Add(time.Hour * -24).UTC()
	}

	deadline.LastUpdateDatetime = time.Now().UTC()

	// Status must be passed in request
	if deadline.Status == "" {
		logger.Error("agreement service SetDeadline - no status in request", errors.New("missing status in request"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("missing status field of deadline instance")
	}

	logger.Info("agreement service SetDeadline finish", context_utils.GetTraceAndClientIds(ctx)...)
	if typeVal == "solo" || len(agreement.Participants) < 2 {
		return a.AgreementRepository.SetDeadline(ctx, agreementId, deadline)
	} else if typeVal == "directed" {
		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s updated deadline of '%s' agreement", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "notifyUpdate",
				Action:           "update",
			})
		}
		return a.AgreementRepository.SetDeadlineDirected(ctx, agreementId, deadline, notifications)
	} else { // collaborative
		agreement.AgreementDeadline = deadline
		updatedAgreement := *agreement
		updatedAgreement.Status = "active"
		agreement.Status = "awaitingConfirmation"
		agreement.UpdatedAgreement = &updatedAgreement
		agreement.UpdatedAgreement.UpdatedAgreement = nil

		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s wants to change '%s' agreement deadline, please respond", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "requires_response",
				Action:           "update",
			})
		}
		_, err := a.AgreementRepository.CollaborativeUpdateAgreementNotifications(ctx, *agreement, notifications)
		if err != nil {
			return nil, err
		} else {
			return nil, nil
		}
	}
}

func (a agreementService) DeleteDeadline(ctx context.Context, agreementId string, typeKey string, typeVal string) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service DeleteDeadlineDirected start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize agreementId and query param
	agreementId = html.EscapeString(agreementId)
	typeKey = strings.TrimSpace(html.EscapeString(typeKey))
	typeVal = strings.TrimSpace(html.EscapeString(typeVal))

	if typeKey != "type" || typeVal != "solo" && typeVal != "directed" && typeVal != "collaborative" {
		logger.Error(fmt.Sprintf("agreement service DeleteDeadlineDirected - improper type key/val: %s %s", typeKey, typeVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("improper type key/val: " + typeKey + "/" + typeVal)
	}

	// Get Existing Agreement
	agreement, getErr := a.GetAgreement(ctx, agreementId)
	if getErr != nil {
		return nil, getErr
	}

	// Check if agreement already closed
	if agreement.Status == "retired" || agreement.Status == "completed" {
		logger.Error(fmt.Sprintf("agreement service DeleteDeadline - agreement already closed: %v", agreement), getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("agreement already closed: " + agreementId)
	}

	// Archive Changes
	if typeVal != "collaborative" {
		agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, agreementId, "modified", "agreement was modified", nil)
		if archiveErr == nil {
			go func() {
				a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
			}()
		}
	}

	logger.Info("agreement service DeleteDeadlineDirected finish", context_utils.GetTraceAndClientIds(ctx)...)
	if typeVal == "solo" || len(agreement.Participants) < 2 {
		return a.AgreementRepository.DeleteDeadline(ctx, agreementId)
	} else if typeVal == "directed" {
		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s removed deadline of '%s' agreement", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "notifyUpdate",
				Action:           "update",
			})
		}
		return a.AgreementRepository.DeleteDeadlineDirected(ctx, agreementId, notifications)
	} else { // TODO collaborative
		agreement.AgreementDeadline.Status = "deleted"
		updatedAgreement := *agreement
		updatedAgreement.Status = "active"
		agreement.Status = "awaitingConfirmation"
		agreement.UpdatedAgreement = &updatedAgreement
		agreement.UpdatedAgreement.UpdatedAgreement = nil

		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s wants to delete '%s' agreement deadline, please respond", agreement.CreatorName, agreement.Title),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "requires_response",
				Action:           "update",
			})
		}
		_, err := a.AgreementRepository.CollaborativeUpdateAgreementNotifications(ctx, *agreement, notifications)
		if err != nil {
			return nil, err
		} else {
			return nil, nil
		}
	}
}

func archiveAgreementHelper(ctx context.Context, agreementRepo repository.AgreementRepositoryInterface, agreementArchiveRepo repository.AgreementArchiveRepositoryInterface, id, status, info string, agreement *domain.Agreement) (*domain.AgreementArchive, rest_errors.RestError) {
	if agreement == nil {
		var err rest_errors.RestError
		agreement, err = agreementRepo.GetAgreement(ctx, id)
		if err != nil {
			logger.Error("agreement service DeleteAgreement could not get current agreement", err, context_utils.GetTraceAndClientIds(ctx)...)
			return nil, err
		}
	}

	// agreement.Status = status
	currTime := time.Now().UTC()
	agreement.LastUpdateDateTime = currTime
	agreementArchive := domain.AgreementArchive{
		Id:             uuid.NewString(),
		AgreementData:  *agreement,
		CreateDateTime: time.Now().UTC(),
		Info:           info,
	}

	return &agreementArchive, nil
}

func (a agreementService) ActionAndNotification(ctx context.Context, notification domain.Notification, typeKey, typeVal string) (*domain.Notification, rest_errors.RestError) {
	logger.Info("agreement service ActionAndNotification start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize action string and notification instance
	notification.Sanitize()
	typeKey = strings.TrimSpace(html.EscapeString(typeKey))
	typeVal = strings.TrimSpace(html.EscapeString(typeVal))

	if typeKey != "type" || typeVal != "solo" && typeVal != "directed" && typeVal != "collaborative" {
		logger.Error(fmt.Sprintf("agreement service ActionAndNotification - improper type key/val: %s %s", typeKey, typeVal), errors.New("key/value are incorrect"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("improper type key/val: " + typeKey + "/" + typeVal)
	}

	// Get Existing Agreement
	agreement, getErr := a.GetAgreement(ctx, notification.AgreementId)
	if getErr != nil {
		return nil, getErr
	}

	// Archive Changes
	if typeVal != "collaborative" {
		agreementArchive, archiveErr := archiveAgreementHelper(ctx, a.AgreementRepository, a.AgreementArchiveRepository, notification.AgreementId, "modified", "agreement was modified", nil)
		if archiveErr == nil {
			go func() {
				a.AgreementArchiveRepository.ArchiveAgreement(ctx, *agreementArchive)
			}()
		}
	}

	// check if request to join agreement is public
	if notification.Action == "request" && agreement.Public == "false" {
		logger.Error("agreement service ActionAndNotification - request to join on a non public agreement", fmt.Errorf("agreement to request is not public: %+v", agreement), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("agreement is not public: " + notification.AgreementId)
	}

	// Set uuid
	notification.Id = uuid.NewString()
	notification.CreateDateTime = time.Now().UTC()

	// Get appropriate inputs for repository
	actionInputs := getActionAndNotificationInputs(notification.Action)
	if actionInputs == nil {
		return nil, rest_errors.NewBadRequestError("action not supported")
	}

	notification.Title = fmt.Sprintf(actionInputs[0], notification.ContactFirstName, notification.AgreementTitle)

	// doneChan := make(chan int)
	// go func(c chan int) {
	// 	defer close(c)
	// 	a.AgreementRepository.ActionAndNotification(ctx, actionInputs, notification)
	// 	doneChan <- 1
	// }(doneChan)

	// notificationResult, repoErr := a.NotificationRepository.SaveNotification(ctx, notification)

	// select {
	// case done := <-doneChan:
	// 	logger.Info(fmt.Sprintf("notification created: %v, chan output: %d", notification, done), context_utils.GetTraceAndClientIds(ctx)...)
	// case <-time.After(5 * time.Second):
	// 	logger.Error("agreement service ActionAndNotification - couldn't save notification", fmt.Errorf("could not update agreement/user for notification: %v", notification), context_utils.GetTraceAndClientIds(ctx)...)
	// 	a.NotificationRepository.DeleteNotification(ctx, notification.Id) // if this fails then complete failure... - maybe put in Fatal Inconsistency DB
	// 	return nil, rest_errors.NewInternalServerError(fmt.Sprintf("could not ActionAndNotification for %v", notification), errors.New("database error"))
	// }

	logger.Info("agreement service ActionAndNotification finish", context_utils.GetTraceAndClientIds(ctx)...)
	// if typeVal == "solo" || typeVal == "directed" {
	return a.AgreementRepository.ActionAndNotification(ctx, actionInputs[2:], notification)
	// } else if typeVal == "collaborative" { // collaborative
	// 	if notification.Action = ""

	// 	notifications := make([]domain.Notification, 0)
	// 	for i := 0; i < len(agreement.Participants); i++ {
	// 		if agreement.Participants[i] == notification.ContactId {
	// 			continue
	// 		}

	// 		notifications = append(notifications, domain.Notification{
	// 			Id:               uuid.NewString(),
	// 			Title:            fmt.Sprintf(actionInputs[1], notification.ContactFirstName, notification.UserFirstName, notification.AgreementTitle),
	// 			Message:          "",
	// 			CreateDateTime:   time.Now().UTC(),
	// 			Status:           "new",
	// 			UserId:           agreement.Participants[i],
	// 			ContactId:        agreement.CreatedBy,
	// 			ContactFirstName: agreement.CreatorName,
	// 			AgreementId:      agreement.Id,
	// 			AgreementTitle:   agreement.Title,
	// 			Response:         "",
	// 			Type:             "notifyUpdate",
	// 			Action:           "update",
	// 		})
	// 	}

	// 	return a.AgreementRepository.ActionAndNotifyAll(ctx, actionInputs[2:], notifications)
	// }
}

func getActionAndNotificationInputs(action string) []string {
	// Up to 2 Actions (for agreements), 2 inputs per action () - the data is passed via obj
	res, ok := actionCodes[action]
	if !ok {
		goto INVALIDACTION
	}

	return res

INVALIDACTION:
	return nil
}

var actionCodes map[string][]string

func init() {
	actionCodes = map[string][]string{
		"invite": {"%s invites you to agreement: %s", "%s invited %s to '%s' agreement", "$push", "invited_participants"},
		// "uninvite": {"%s uninvites you to agreement: %s", "$pull", "invited_participants"},
		"acceptInvite":     {"%s accepted your invite to agreement: %s", "%s accepted %s's invite to agreement %s", "$pull", "invited_participants", "$push", "participants"},
		"declineInvite":    {"%s declined your invite to agreement: %s", "%s declined %s's invite to agreement %s", "$pull", "invited_participants"},
		"requestAgreement": {"%s requests you join agreement: %s", "%s requested %s to join '%s' agreement", "$push", "requested_participants"},
		// "unrequestAgreement": {"%s unrequested you join agreement: %s", "$pull", "requested_participants"},
		"acceptRequest":  {"%s accepted your request to agreement: %s", "%s accepted %s's request to '%s' agreement", "$pull", "requested_participants", "$push", "participants"},
		"declineRequest": {"%s declined your request to agreement: %s", "%s declined %s's request to '%s' agreement", "$pull", "requested_participants"},
		"remove":         {"%s requests to remove you from agreement: %s", "%s requests to remove %s from '%s' agreement", "$push", "pending_removal_participants"},
		// "unremove": {"%s unrequests to remove you from agreement: %s", "$pull", "pending_removal_participants"},
		"acceptRemove":  {"%s accepts your removal request for agreement: %s", "%s accepted %s's removal request for '%s' agreement", "$pull", "pending_removal_participants", "$pull", "participants"},
		"declineRemove": {"%s declines your removal request for agreement: %s", "%s declined %s's removal request for '%s' agreement", "$pull", "pending_removal_participants"},
		"leave":         {"%s wants to leave your agreement: %s", "%s wants to leave %s's '%s' agreement", "$push", "pending_leave_participants"},
		// "unleave": {"%s unwants to leave your to agreement: %s", "$pull", "pending_leave_participants"},
		"acceptLeave":  {"%s accepts your request to leave agreement: %s", "%s accepted %s's request to leave '%s' agreement", "$pull", "pending_leave_participants", "$pull", "participants"},
		"declineLeave": {"%s declined your request to leave agreement: %s", "%s accepted %s's request to leave '%s' agreement", "$pull", "pending_leave_participants"},
	}

	// fmt.Printf("%v\n", actionCodes)
}

func (a agreementService) RespondAgreementChange(ctx context.Context, notification domain.Notification) (*domain.Agreement, rest_errors.RestError) {
	logger.Info("agreement service RespondAgreementChange start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize notification
	notification.Sanitize()

	if valid := notification.Validate(); !valid {
		validateErr := rest_errors.NewBadRequestError("req (notification) failed validation")
		logger.Error("agreement service RespondAgreementChange - could not get agreement", validateErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, validateErr
	}

	// Get Current Agreement
	agreement, getErr := a.GetAgreement(ctx, notification.AgreementId)
	if getErr != nil {
		logger.Error("agreement service RespondAgreementChange - could not get agreement", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, getErr
	}

	// check if agreement is awaiting confirmation
	if agreement.Status != "awaitingConfirmation" {
		logger.Error("agreement service RespondAgreementChange - agreement is not waiting for responses", fmt.Errorf("%+v", agreement), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("agreement is not waiting for a response")
	}

	agreement.LastUpdateDateTime = time.Now().UTC()
	if agreement.UpdatedAgreement != nil {
		agreement.UpdatedAgreement.LastUpdateDateTime = time.Now().UTC()
	}

	if notification.Response == "accept" {
		agreement.AgreementAccept = append(agreement.AgreementAccept, notification.UserId)
		return a.AgreementRepository.UpdateAgreementRead(ctx, *agreement, notification.Id)
	} else if notification.Response == "decline" {
		agreement.AgreementDecline = append(agreement.AgreementDecline, notification.UserId)
		return a.AgreementRepository.UpdateAgreementRead(ctx, *agreement, notification.Id)
	} else if notification.Response == "commit" {
		agreement.UpdatedAgreement.AgreementAccept = []string{}
		agreement.UpdatedAgreement.AgreementDecline = []string{}
		agreement.UpdatedAgreement.UpdatedAgreement = nil

		var msg string
		var typestr string

		if notification.Action == "change" {
			if notification.Type == "complete" || notification.Type == "retire" {
				msg = "agreement was " + notification.Action + "d"
			} else {
				msg = "agreement was changed"
			}
			typestr = "Change"
		} else {
			msg = "agreement changes reverted"
			typestr = "Revert"
		}

		notifications := make([]domain.Notification, 0)
		for i := 0; i < len(agreement.Participants); i++ {
			if agreement.Participants[i] == agreement.CreatedBy {
				continue
			}

			notifications = append(notifications, domain.Notification{
				Id:               uuid.NewString(),
				Title:            fmt.Sprintf("%s committed '%s' agreement - %s", agreement.CreatorName, agreement.Title, msg),
				Message:          "",
				CreateDateTime:   time.Now().UTC(),
				Status:           "new",
				UserId:           agreement.Participants[i],
				ContactId:        agreement.CreatedBy,
				ContactFirstName: agreement.CreatorName,
				AgreementId:      agreement.Id,
				AgreementTitle:   agreement.Title,
				Response:         "",
				Type:             "notify" + typestr,
				Action:           "update",
			})
		}

		if notification.Action == "change" {
			if notification.Type == "complete" || notification.Type == "retire" {
				fmt.Println("changing the status of the updated agreement to be: " + notification.Type + "d")
				agreement.UpdatedAgreement.Status = notification.Type + "d"
			} else { // just a change so back to active
				agreement.UpdatedAgreement.Status = "active"
			}
			// fmt.Printf("%+v\n", *agreement.UpdatedAgreement)
			fmt.Println(agreement.UpdatedAgreement.Status)
			return a.AgreementRepository.RespondAgreementChange(ctx, *agreement.UpdatedAgreement, notifications)
		} else { // notification.Response == "revert"
			agreement.Status = "active"
			agreement.UpdatedAgreement = nil
			return a.AgreementRepository.RespondAgreementChange(ctx, *agreement, notifications)
		}
	} else { // response not accepted
		logger.Error("agreement service RespondAgreementChange - invalid response value: "+notification.Response, errors.New("respond value for notification not valid"), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("notification response value is invalid: " + notification.Response)
	}
}

func (a agreementService) GetAgreementEventResponses(ctx context.Context, agreementId string) ([]domain.EventResponse, rest_errors.RestError) {
	logger.Info("agreement service GetAgreementEventResponses start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize
	agreementId = strings.TrimSpace(html.EscapeString(agreementId))

	logger.Info("agreement service GetAgreementEventResponses finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.GetAgreementEventResponses(ctx, agreementId)
}

func (a agreementService) InviteUsersToEvent(ctx context.Context, agreementId string, uuids []string) (string, rest_errors.RestError) {
	logger.Info("agreement service InviteUsersToEvent start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize inputs
	agreementId = strings.TrimSpace(html.EscapeString(agreementId))
	uuids = domain.SanitizeStringSlice(uuids)

	// Get Current Agreement
	agreement, getErr := a.GetAgreement(ctx, agreementId)
	if getErr != nil {
		logger.Error("agreement service InviteUsersToEvent - could not get agreement", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return "", getErr
	}

	var newUsers []string

	for _, id := range uuids {
		if isInSlice(id, agreement.InvitedParticipants) {
			continue
		}
		newUsers = append(newUsers, id)
		agreement.InvitedParticipants = append(agreement.InvitedParticipants, id)
	}

	// check if no new userIds to invite
	if len(newUsers) == 0 {
		logger.Error("agreement service InviteUsersToEvent - no new users to invite", fmt.Errorf("incoming userIds: %v does not contain new ids, agreement: %+v", uuids, agreement), context_utils.GetTraceAndClientIds(ctx)...)
		return "", rest_errors.NewBadRequestError("all users in request are already invited")
	}

	notifications := make([]domain.Notification, 0)
	for i := 0; i < len(newUsers); i++ {
		notifications = append(notifications, domain.Notification{
			Id:               uuid.NewString(),
			Title:            fmt.Sprintf("%s invites you to '%s' agreement, please respond", agreement.CreatorName, agreement.Title),
			Message:          "",
			CreateDateTime:   time.Now().UTC(),
			Status:           "new",
			UserId:           uuids[i],
			ContactId:        agreement.CreatedBy,
			ContactFirstName: agreement.CreatorName,
			AgreementId:      agreement.Id,
			AgreementTitle:   agreement.Title,
			Response:         "",
			Type:             "requires_response",
			Action:           "eventInvite",
		})
	}

	logger.Info("agreement service InviteUsersToEvent finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.InviteUsersToEvent(ctx, *agreement, notifications)
}

func isInSlice(input string, arr []string) bool {
	for _, el := range arr {
		if el == input {
			return true
		}
	}

	return false
}

func (a agreementService) RespondEventInvite(ctx context.Context, agreementId string, eventResponse domain.EventResponse) (*domain.EventResponse, rest_errors.RestError) {
	logger.Info("agreement service RespondEventInvite start", context_utils.GetTraceAndClientIds(ctx)...)

	// Sanitize
	agreementId = strings.TrimSpace(html.EscapeString(agreementId))
	eventResponse.Sanitize()

	// Validate
	if ok := eventResponse.Validate(); !ok {
		logger.Error("agreement service RespondEventInvite - eventResponse failed validation", fmt.Errorf("invalidate eventResponse: %+v", eventResponse), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("invalid eventResponse")
	}

	// get current Agreement
	agreement, getErr := a.GetAgreement(ctx, agreementId)
	if getErr != nil {
		logger.Error("agreement service RespondEventInvite - could not get agreement", getErr, context_utils.GetTraceAndClientIds(ctx)...)
		return nil, getErr
	}

	eventResponse.Id = uuid.NewString()
	eventResponse.CreateDateTime = time.Now().UTC()

	if isInSlice(eventResponse.UserId, agreement.InvitedParticipants) {
		agreement.EventResponses = append(agreement.EventResponses, eventResponse.Id)
	} else {
		// the user was not invited, should not respond
		logger.Error("greement service RespondEventInvite - user was not invited", fmt.Errorf("eventResponse: %+v - user not in agreement.InvitedParticipants: %+v", agreement, eventResponse), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("user was not invited, should not respond")
	}

	// check if user already responded
	if isInSlice(eventResponse.UserId, agreement.AgreementAccept) || isInSlice(eventResponse.UserId, agreement.AgreementDecline) {
		logger.Error("greement service RespondEventInvite - user already responded", fmt.Errorf("eventResponse: %+v - user already responded %+v", agreement, eventResponse), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("user already responded")
	}

	// update AgreementAccept or AgreementDecline slice
	if eventResponse.Response == "accept" {
		agreement.AgreementAccept = append(agreement.AgreementAccept, eventResponse.UserId)
	} else if eventResponse.Response == "decline" {
		agreement.AgreementDecline = append(agreement.AgreementDecline, eventResponse.UserId)
	} else {
		logger.Error("greement service RespondEventInvite - invalid response value for eventResponse", fmt.Errorf("eventResponse: %+v - invalid response value for eventResponse %+v", agreement, eventResponse), context_utils.GetTraceAndClientIds(ctx)...)
		return nil, rest_errors.NewBadRequestError("invalid response value")
	}

	logger.Info("agreement service RespondEventInvite finish", context_utils.GetTraceAndClientIds(ctx)...)
	return a.AgreementRepository.RespondEventInvite(ctx, *agreement, eventResponse)
}
