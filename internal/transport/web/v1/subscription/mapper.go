package subscription

import "github.com/EgorLis/my-subs/internal/domain"

// --- запросы -> домен ---

func MapCreateReqToDomain(req CreateRequest) domain.Subscription {
	return domain.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate.ToTime(),
		EndDate:     req.EndDate.ToTime(),
	}
}

func MapUpdateReqToDomain(req UpdateRequest) domain.Subscription {
	return domain.Subscription{
		ID:          req.ID,
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate.ToTime(),
		EndDate:     req.EndDate.ToTime(),
	}
}

// --- домен -> DTO/Response---

func MapDomainToDTO(sub domain.Subscription) SubscriptionDTO {
	return SubscriptionDTO{
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID,
		StartDate:   YearMonth(sub.StartDate),
		EndDate:     YearMonth(sub.EndDate),
	}
}

func MapDomainListToDTO(subs []domain.Subscription) []SubscriptionDTO {
	out := make([]SubscriptionDTO, 0, len(subs))
	for _, s := range subs {
		out = append(out, MapDomainToDTO(s))
	}
	return out
}
