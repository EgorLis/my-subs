package subscription

type CreateRequest struct {
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      string    `json:"user_id"`
	StartDate   YearMonth `json:"start_date"`
	EndDate     YearMonth `json:"end_date"`
}

type UpdateRequest struct {
	ID          string    `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      string    `json:"user_id"`
	StartDate   YearMonth `json:"start_date"`
	EndDate     YearMonth `json:"end_date"`
}
