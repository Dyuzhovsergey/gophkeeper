package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Dyuzhovsergey/gophkeeper/internal/domain"
	vaultservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/vault"
	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/dto"
	httpmiddleware "github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/middleware"
	"github.com/Dyuzhovsergey/gophkeeper/internal/transport/http/response"
)

// VaultService описывает бизнес-логику работы с секретами,
// которая нужна HTTP-обработчикам.
type VaultService interface {
	// Create создаёт новый секрет.
	Create(ctx context.Context, input vaultservice.CreateInput) (*domain.SecretItem, error)

	// GetByID возвращает один секрет пользователя.
	GetByID(ctx context.Context, ownerID, secretID string) (*domain.SecretItem, error)

	// ListByOwner возвращает список секретов пользователя.
	ListByOwner(ctx context.Context, ownerID string) ([]*domain.SecretItem, error)

	// Update обновляет существующий секрет пользователя.
	Update(ctx context.Context, input vaultservice.UpdateInput) (*domain.SecretItem, error)

	// Delete выполняет мягкое удаление секрета пользователя.
	Delete(ctx context.Context, ownerID, secretID string) error
}

// SecretsHandler обрабатывает HTTP-запросы к секретам.
type SecretsHandler struct {
	service VaultService
}

// NewSecretsHandler создаёт новый SecretsHandler.
func NewSecretsHandler(service VaultService) *SecretsHandler {
	return &SecretsHandler{service: service}
}

// Collection обрабатывает маршруты коллекции секретов:
// POST /api/secrets
// GET  /api/secrets
func (h *SecretsHandler) Collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Item обрабатывает маршруты одного секрета:
// GET    /api/secrets/{id}
// PUT    /api/secrets/{id}
// DELETE /api/secrets/{id}
func (h *SecretsHandler) Item(w http.ResponseWriter, r *http.Request) {
	secretID := strings.TrimPrefix(r.URL.Path, "/api/secrets/")
	secretID = strings.TrimSpace(secretID)

	if secretID == "" || strings.Contains(secretID, "/") {
		response.Error(w, http.StatusNotFound, "secret not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getByID(w, r, secretID)
	case http.MethodPut:
		h.update(w, r, secretID)
	case http.MethodDelete:
		h.delete(w, r, secretID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *SecretsHandler) create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentOwnerID(r.Context())
	if !ok {
		response.Error(w, http.StatusInternalServerError, "identity not found in context")
		return
	}

	var req dto.SecretUpsertRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	secretType, secretData, err := parseSecretUpsertRequest(req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid secret payload")
		return
	}

	item, err := h.service.Create(r.Context(), vaultservice.CreateInput{
		OwnerID: ownerID,
		Type:    secretType,
		Meta:    req.Meta,
		Data:    secretData,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBinaryPayloadTooLarge):
			response.Error(w, http.StatusRequestEntityTooLarge, "binary payload too large")
			return
		case errors.Is(err, domain.ErrInvalidSecretType),
			errors.Is(err, domain.ErrInvalidSecretData):
			response.Error(w, http.StatusBadRequest, "invalid secret payload")
			return
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	response.JSON(w, http.StatusCreated, secretToResponse(item))
}

func (h *SecretsHandler) list(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentOwnerID(r.Context())
	if !ok {
		response.Error(w, http.StatusInternalServerError, "identity not found in context")
		return
	}

	items, err := h.service.ListByOwner(r.Context(), ownerID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := dto.SecretListResponse{
		Items: make([]dto.SecretResponse, 0, len(items)),
	}

	for _, item := range items {
		resp.Items = append(resp.Items, secretToResponse(item))
	}

	response.JSON(w, http.StatusOK, resp)
}

func (h *SecretsHandler) getByID(w http.ResponseWriter, r *http.Request, secretID string) {
	ownerID, ok := currentOwnerID(r.Context())
	if !ok {
		response.Error(w, http.StatusInternalServerError, "identity not found in context")
		return
	}

	item, err := h.service.GetByID(r.Context(), ownerID, secretID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSecretNotFound),
			errors.Is(err, domain.ErrSecretDeleted):
			response.Error(w, http.StatusNotFound, "secret not found")
			return
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	response.JSON(w, http.StatusOK, secretToResponse(item))
}

func (h *SecretsHandler) update(w http.ResponseWriter, r *http.Request, secretID string) {
	ownerID, ok := currentOwnerID(r.Context())
	if !ok {
		response.Error(w, http.StatusInternalServerError, "identity not found in context")
		return
	}

	var req dto.SecretUpsertRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	secretType, secretData, err := parseSecretUpsertRequest(req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid secret payload")
		return
	}

	item, err := h.service.Update(r.Context(), vaultservice.UpdateInput{
		ID:      secretID,
		OwnerID: ownerID,
		Type:    secretType,
		Meta:    req.Meta,
		Data:    secretData,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBinaryPayloadTooLarge):
			response.Error(w, http.StatusRequestEntityTooLarge, "binary payload too large")
			return
		case errors.Is(err, domain.ErrInvalidSecretType),
			errors.Is(err, domain.ErrInvalidSecretData):
			response.Error(w, http.StatusBadRequest, "invalid secret payload")
			return
		case errors.Is(err, domain.ErrSecretNotFound),
			errors.Is(err, domain.ErrSecretDeleted):
			response.Error(w, http.StatusNotFound, "secret not found")
			return
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	response.JSON(w, http.StatusOK, secretToResponse(item))
}

func (h *SecretsHandler) delete(w http.ResponseWriter, r *http.Request, secretID string) {
	ownerID, ok := currentOwnerID(r.Context())
	if !ok {
		response.Error(w, http.StatusInternalServerError, "identity not found in context")
		return
	}

	if err := h.service.Delete(r.Context(), ownerID, secretID); err != nil {
		switch {
		case errors.Is(err, domain.ErrSecretNotFound),
			errors.Is(err, domain.ErrSecretDeleted):
			response.Error(w, http.StatusNotFound, "secret not found")
			return
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func currentOwnerID(ctx context.Context) (string, bool) {
	identity, ok := httpmiddleware.IdentityFromContext(ctx)
	if !ok {
		return "", false
	}

	if strings.TrimSpace(identity.UserID) == "" {
		return "", false
	}

	return identity.UserID, true
}

func parseSecretUpsertRequest(req dto.SecretUpsertRequest) (domain.SecretType, domain.SecretData, error) {
	secretType := domain.SecretType(strings.ToLower(strings.TrimSpace(req.Type)))

	switch secretType {
	case domain.SecretTypeCredentials:
		var payload dto.CredentialsData
		if err := json.Unmarshal(req.Data, &payload); err != nil {
			return "", nil, err
		}

		return secretType, domain.CredentialData{
			Login:    payload.Login,
			Password: payload.Password,
		}, nil

	case domain.SecretTypeText:
		var payload dto.TextData
		if err := json.Unmarshal(req.Data, &payload); err != nil {
			return "", nil, err
		}

		return secretType, domain.TextData{
			Text: payload.Text,
		}, nil
	case domain.SecretTypeCard:
		var payload dto.CardData
		if err := json.Unmarshal(req.Data, &payload); err != nil {
			return "", nil, err
		}

		return secretType, domain.CardData{
			Number:      payload.Number,
			Cardholder:  payload.Cardholder,
			ExpiryMonth: payload.ExpiryMonth,
			ExpiryYear:  payload.ExpiryYear,
			CVV:         payload.CVV,
		}, nil

	case domain.SecretTypeBinary:
		var payload dto.BinaryData
		if err := json.Unmarshal(req.Data, &payload); err != nil {
			return "", nil, err
		}

		content, err := base64.StdEncoding.DecodeString(payload.ContentBase64)
		if err != nil {
			return "", nil, err
		}

		return secretType, domain.BinaryData{
			Filename: payload.Filename,
			MIMEType: payload.MIMEType,
			Content:  content,
		}, nil
	default:
		return "", nil, domain.ErrInvalidSecretType
	}
}

func secretToResponse(item *domain.SecretItem) dto.SecretResponse {
	return dto.SecretResponse{
		ID:        item.ID,
		Type:      string(item.Type),
		Meta:      item.Meta,
		Data:      secretDataToResponse(item.Data),
		Version:   item.Version,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func secretDataToResponse(data domain.SecretData) any {
	switch v := data.(type) {
	case domain.CredentialData:
		return dto.CredentialsData{
			Login:    v.Login,
			Password: v.Password,
		}
	case *domain.CredentialData:
		if v == nil {
			return nil
		}
		return dto.CredentialsData{
			Login:    v.Login,
			Password: v.Password,
		}
	case domain.TextData:
		return dto.TextData{
			Text: v.Text,
		}
	case *domain.TextData:
		if v == nil {
			return nil
		}
		return dto.TextData{
			Text: v.Text,
		}
	case domain.CardData:
		return dto.CardData{
			Number:      v.Number,
			Cardholder:  v.Cardholder,
			ExpiryMonth: v.ExpiryMonth,
			ExpiryYear:  v.ExpiryYear,
			CVV:         v.CVV,
		}
	case *domain.CardData:
		if v == nil {
			return nil
		}
		return dto.CardData{
			Number:      v.Number,
			Cardholder:  v.Cardholder,
			ExpiryMonth: v.ExpiryMonth,
			ExpiryYear:  v.ExpiryYear,
			CVV:         v.CVV,
		}
	case domain.BinaryData:
		return dto.BinaryData{
			Filename:      v.Filename,
			MIMEType:      v.MIMEType,
			ContentBase64: base64.StdEncoding.EncodeToString(v.Content),
		}
	case *domain.BinaryData:
		if v == nil {
			return nil
		}
		return dto.BinaryData{
			Filename:      v.Filename,
			MIMEType:      v.MIMEType,
			ContentBase64: base64.StdEncoding.EncodeToString(v.Content),
		}
	default:
		return nil
	}
}
