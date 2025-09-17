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
	Log  *log.Logger
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
		h.Log.Printf("unmarshal error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	defer r.Body.Close()

	if err := ValidateCreateRequest(req); err != nil {
		h.Log.Printf("validation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub := MapCreateReqToDomain(req)

	subWithID, err := h.Repo.AddSub(ctx, sub)

	if err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			h.Log.Printf("request timed out: %v", err)
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		h.Log.Printf("repo error while add value: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &CUDResponse{SubID: subWithID.ID, Status: CREATED}

	h.Log.Printf("subscription created, id: %s", subWithID.ID)

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
		h.Log.Printf("validation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	sub, err := h.Repo.GetSub(ctx, id)
	if err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			h.Log.Printf("request timed out: %v", err)
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			h.Log.Printf("row with id: %s, not found", id)
			v1.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		h.Log.Printf("repo error while geting value: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := MapDomainToDTO(sub)

	h.Log.Printf("subscription returned, id: %s", id)

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
		h.Log.Printf("validation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub := MapUpdateReqToDomain(req)

	if err := h.Repo.UpdateSub(ctx, sub); err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			h.Log.Printf("request timed out: %v", err)
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			h.Log.Printf("row with id: %s, not found", req.ID)
			v1.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		h.Log.Printf("repo error while updating: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &CUDResponse{SubID: req.ID, Status: UPDATED}

	h.Log.Printf("subscription updated, id: %s", req.ID)

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
		h.Log.Printf("validation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	if err := h.Repo.DeleteSub(ctx, id); err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			h.Log.Printf("request timed out: %v", err)
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		if errors.Is(err, domain.ErrNotFound) {
			h.Log.Printf("row with id: %s, not found", id)
			v1.WriteError(w, http.StatusNotFound, "not found")
			return
		}

		h.Log.Printf("repo error while deleting: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &CUDResponse{SubID: id, Status: DELETED}

	h.Log.Printf("subscription deleted, id: %s", id)

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
			h.Log.Printf("request timed out: %v", err)
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		h.Log.Printf("repo error while get values: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &ListResponse{Subs: MapDomainListToDTO(subs)}

	h.Log.Println("subscriptions list returned")

	v1.WriteJSON(w, http.StatusOK, resp)
}

// TotalCost godoc
// @Summary      Calculate total subscriptions cost
// @Description  Получить суммарную стоимость подписок за период, с фильтрацией по пользователю и названию подписки
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        user_id     query  string  true   "ID пользователя"
// @Param        service_name        query  string  true   "Название подписки"
// @Param        from        query  string  true   "Начало периода (MM-YYYY)"
// @Param        to          query  string  true   "Конец периода (MM-YYYY)"
// @Success      200  {object}  subscription.TotalCostResponse
// @Failure      400  {object}  map[string]string
// @Failure      504  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /v1/subscriptions/totalcost [get]
func (h *Handler) TotalCost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	userIDStr := q.Get("user_id")
	serviceName := q.Get("service_name")
	fromStr := q.Get("from")
	toStr := q.Get("to")

	fromYM, err := YMFromStr(fromStr)
	if err != nil {
		h.Log.Printf("convertation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	toYM, err := YMFromStr(toStr)
	if err != nil {
		h.Log.Printf("convertation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := ValidateTotalCostQuery(userIDStr, serviceName, fromYM, toYM); err != nil {
		h.Log.Printf("validation error: %v", err)
		v1.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5000*time.Millisecond)
	defer cancel()

	totalCost, err :=
		h.Repo.TotalCost(ctx, serviceName, userIDStr, fromYM.ToTime(), toYM.ToTime())

	if err != nil {
		// Проверка на timeout / отмену контекста
		if v1.IsTimeout(err) {
			h.Log.Printf("request timed out: %v", err)
			v1.WriteError(w, http.StatusGatewayTimeout, "request timed out")
			return
		}

		h.Log.Printf("repo error while get total cost: %v", err)
		v1.WriteError(w, http.StatusInternalServerError, "")
		return
	}

	resp := &TotalCostResponse{UserID: userIDStr, ServiceName: serviceName,
		From: fromYM, To: toYM, TotalCost: totalCost}

	h.Log.Printf("user_id = %s, service_name = %s, from  %s, to %s,  total cost =  %d",
		userIDStr, serviceName, fromStr, toStr, totalCost)

	v1.WriteJSON(w, http.StatusOK, resp)
}
