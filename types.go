package main

type Deposit struct {
	UserID int `json:"user_id" binding:"required"`
	Amount int `json:"amount" binding:"required,gt=0" `
}

type Payment struct {
	UserID    int `json:"user_id" binding:"required"`
	ServiceID int `json:"service_id" binding:"required"`
	OrderID   int `json:"order_id" binding:"required"`
	Amount    int `json:"amount" binding:"required,gt=0"`
}

type MoneySend struct {
	From   int `json:"from" binding:"required"`
	To     int `json:"to" binding:"required"`
	Amount int `json:"amount" binding:"required,gt=0"`
}

type Transaction struct {
	UserID       int    `json:"user_id`
	AmountSortBy string `json:"amount_sort_by"`
	DateSortBy   string `json:"amount_sort_by"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
}
