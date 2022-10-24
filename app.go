package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func balance(c *gin.Context, store MoneyStore) {
	userID := c.Param("userID")

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "user_id should be integer",
		})
		return
	}

	hasBalance, err := store.hasBalance(context.Background(), userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	if !hasBalance {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "user don't have balance",
		})
		return
	}

	balance, err := store.GetBalance(context.Background(), userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance": balance,
	})
}

func deposit(c *gin.Context, store MoneyStore) {
	var deposit Deposit
	if err := c.BindJSON(&deposit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad json",
		})
		return
	}

	if err := store.Deposit(context.Background(), deposit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "can't deposit",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func reserve(c *gin.Context, store MoneyStore) {
	var payment Payment
	if err := c.BindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad request",
		})
		return
	}

	hasBalance, err := store.hasBalance(context.Background(), payment.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	if !hasBalance {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "user don't have balance",
		})
		return
	}

	balance, err := store.GetBalance(context.Background(), payment.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	if payment.Amount > balance {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "balance is less then reserve",
		})
		return
	}

	_, _, _, err = store.GetPayment(context.Background(), payment)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "this payment has already reserved",
		})
		return
	} else if err.Error() != "payment not found" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "can't check payment",
		})
		return
	}

	if err := store.Reserve(context.Background(), payment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "can't reserve payment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func paymentPreprocess(c *gin.Context, store MoneyStore) (int, error) {
	var payment Payment
	if err := c.BindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad json",
		})
		return 0, err
	}

	var paymentID int
	var paymentConfirmedAt pgtype.Timestamptz
	var paymentCanceledAt pgtype.Timestamptz

	paymentID, paymentConfirmedAt, paymentCanceledAt, err := store.GetPayment(context.Background(), payment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "can't find payment",
		})
		return 0, err
	}

	if paymentConfirmedAt.Status != pgtype.Null {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "payment already confirmed",
		})
		return 0, errors.New("payment already confirmed")
	}

	if paymentCanceledAt.Status != pgtype.Null {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "payment already canceled",
		})
		return 0, errors.New("payment already canceled")
	}

	return paymentID, nil
}

func confirm(c *gin.Context, store MoneyStore) {
	paymentID, err := paymentPreprocess(c, store)
	if err != nil {
		return
	}

	if err := store.Confirm(context.Background(), paymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "can't confirm payment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func cancel(c *gin.Context, store MoneyStore) {
	paymentID, err := paymentPreprocess(c, store)
	if err != nil {
		return
	}

	if err := store.Cancel(context.Background(), paymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "can't cancel payment",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func send(c *gin.Context, store MoneyStore) {
	var moneySend MoneySend
	if err := c.BindJSON(&moneySend); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "bad json",
		})
		return
	}

	if moneySend.From == moneySend.To {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "you can't send money to yourself",
		})
		return
	}

	hasBalance, err := store.hasBalance(context.Background(), moneySend.From)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	if !hasBalance {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "user don't have balance",
		})
		return
	}

	balance, err := store.GetBalance(context.Background(), moneySend.From)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	if moneySend.Amount > balance {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "amount greater than balance",
		})
		return
	}

	if err := store.SendMoney(context.Background(), moneySend); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "can't send money",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}

func transactions(c *gin.Context, store MoneyStore) {
	userID := c.Param("userID")

	var transaction Transaction

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "user_id should be integer",
		})
		return
	}

	amountSortBy, amountSortByExists := c.GetQuery("amountSortBy")
	if amountSortByExists {
		if !(amountSortBy == "ASC" || amountSortBy == "DESC") {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "amountSortBy should be ASC or DESC",
			})
			return
		}
		transaction.AmountSortBy = amountSortBy
	}

	dateSortBy, dateSortByExists := c.GetQuery("dateSortBy")
	if dateSortByExists {
		if !(dateSortBy == "ASC" || dateSortBy == "DESC") {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "dateSortBy should be ASC or DESC",
			})
			return
		}
		transaction.DateSortBy = dateSortBy
	}

	limit, limitExists := c.GetQuery("limit")
	if limitExists {
		limitInt, err := strconv.Atoi(limit)
		if err != nil || limitInt < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "limit should be integer and more than zero",
			})
			return
		}
		transaction.Limit = limitInt
	}

	offset, offsetExists := c.GetQuery("offset")
	if offsetExists {
		offsetInt, err := strconv.Atoi(offset)
		if err != nil || offsetInt < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "offset should be integer and more than zero",
			})
			return
		}
		transaction.Offset = offsetInt
	}

	hasBalance, err := store.hasBalance(context.Background(), userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "error",
		})
		return
	}

	if !hasBalance {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "user don't have balance",
		})
		return
	}

	transaction.UserID = userIDInt

	result, err := store.Transactions(context.Background(), transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "can't get transactions",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func main() {
	dbPool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		fmt.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	var store MoneyStore
	store.pool = dbPool

	r := gin.Default()

	r.GET("/balance/:userID", func(c *gin.Context) {
		balance(c, store)
	})

	r.POST("/deposit", func(c *gin.Context) {
		deposit(c, store)
	})

	r.POST("/reserve", func(c *gin.Context) {
		reserve(c, store)
	})

	r.POST("/confirm", func(c *gin.Context) {
		confirm(c, store)
	})

	r.POST("/cancel", func(c *gin.Context) {
		cancel(c, store)
	})

	r.POST("/send", func(c *gin.Context) {
		send(c, store)
	})

	r.GET("/transactions/:userID", func(c *gin.Context) {
		transactions(c, store)
	})

	r.Run()
}

/*
TODO:
1. Вынести методы к бд в отдельный объект +
2. Написать тесты к этому объекту?
3. Проверить валидацию данных

4. Отчет для бухгалтера: SQL и CSV
5. Детализация +

6. Swagger
7. Оформление github, база данных, swagger, дальнейшие улучшения

Улучшение: делать временные отчеты за предыдующие периоды, чтобы каждый раз не обращаться к предыдующим данным
*/
