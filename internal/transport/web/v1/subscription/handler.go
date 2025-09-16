package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/EgorLis/my-subs/internal/domain"
	v1 "github.com/EgorLis/my-subs/internal/transport/web/v1"
)

const (
	CREATED = "subscription created"
	UPDATED = "subscription updated"
	DELETED = "subscription deleted"
)

type Handler struct {
	Repo domain.SubscriptionRepository
}

// Create godoc
// @Summary      Create subscription
// @Description  Создать новую подписку
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        request  body      subscription.CreateRequest  true  "Subscription payload"
// @Success      200      {object}  subscription.CUDResponse
// @Failure      400      {object}  map[string]string
// @Failure      504      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /v1/subscriptions [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	var req CreateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		v1.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	defer r.Body.Close()

	if err := ValidateCreateRequest(req); err != nil {
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub := MapCreateReqToDomain(req)

	subWithID, err := h.Repo.AddSub(ctx, sub)

	if err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		log.Printf("repo error while add value: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &CUDResponse{SubID: subWithID.ID, Status: CREATED}

	v1.WriteJSON(w, http.StatusOK, resp)
}

// Get godoc
// @Summary      Get subscription by ID
// @Description  Получить подписку по её идентификатору
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "Subscription ID (GUID)"
// @Success      200  {object}  subscription.SubscriptionDTO
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      504  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /v1/subscriptions/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := ValidateGUID(id); err != nil {
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	sub, err := h.Repo.GetSub(ctx, id)
	if err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			v1.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		log.Printf("repo error while geting value: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := MapDomainToDTO(sub)

	v1.WriteJSON(w, http.StatusOK, resp)
}

// Update godoc
// @Summary      Update subscription
// @Description  Обновить данные существующей подписки
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        request  body      subscription.UpdateRequest  true  "Subscription payload"
// @Success      200      {object}  subscription.CUDResponse
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      504      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /v1/subscriptions/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	var req UpdateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		v1.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	defer r.Body.Close()

	if err := ValidateUpdateRequest(req); err != nil {
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub := MapUpdateReqToDomain(req)

	if err := h.Repo.UpdateSub(ctx, sub); err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			v1.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		log.Printf("repo error while updating: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &CUDResponse{SubID: req.ID, Status: UPDATED}

	v1.WriteJSON(w, http.StatusOK, resp)
}

// Delete godoc
// @Summary      Delete subscription
// @Description  Удалить подписку по её идентификатору
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "Subscription ID (GUID)"
// @Success      200  {object}  subscription.CUDResponse
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      504  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /v1/subscriptions/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := ValidateGUID(id); err != nil {
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	if err := h.Repo.DeleteSub(ctx, id); err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			v1.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		log.Printf("repo error while deleting: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &CUDResponse{SubID: id, Status: DELETED}

	v1.WriteJSON(w, http.StatusOK, resp)
}

// List godoc
// @Summary      List subscriptions
// @Description  Получить список всех подписок
// @Tags         subscriptions
// @Produce      json
// @Success      200  {object}  subscription.ListResponse
// @Failure      504  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /v1/subscriptions [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	subs, err := h.Repo.ListSubs(ctx)
	if err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		log.Printf("repo error while get values: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &ListResponse{Subs: MapDomainListToDTO(subs)}

	v1.WriteJSON(w, http.StatusOK, resp)
}
