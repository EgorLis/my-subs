package subscription

type SubscriptionDTO struct {
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      string    `json:"user_id"`
	StartDate   YearMonth `json:"start_date"`
	EndDate     YearMonth `json:"end_date"`
}

// ответ для CREATE, UPDATE, DELETE,
type CUDResponse struct {
	SubID  string `json:"subscription_id"`
	Status string `json:"status"`
}

type ListResponse struct {
	Subs []SubscriptionDTO `json:"subscriptions"`
}
