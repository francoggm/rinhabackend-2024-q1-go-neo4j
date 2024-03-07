package controller

import (
	"context"
	"crebito/database"
	"crebito/models"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func HandleTransaction(w http.ResponseWriter, r *http.Request, s neo4j.SessionWithContext) {
	defer r.Body.Close()

	idParam := r.URL.Query().Get("id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if id < 1 || id > 5 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var req models.TransactionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if req.Value < 1 || (req.Type != "d" && req.Type != "c") || (len(req.Description) < 1 || len(req.Description) > 10) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	if req.Type == "d" {
		req.Value = -1 * req.Value
	}

	ctx := context.Background()

	result, err := s.ExecuteWrite(ctx,
		func(tx neo4j.ManagedTransaction) (any, error) {
			result, err := tx.Run(ctx, database.TransactionQuery,
				map[string]any{
					"id":        id,
					"tipo":      req.Type,
					"valor":     req.Value,
					"descricao": req.Description,
				})
			if err != nil {
				return nil, err
			}

			record, err := result.Single(ctx)
			if err != nil {
				return nil, models.ErrUserNotFound
			}

			if record.AsMap()["transacao"] == nil {
				return nil, models.ErrInsufficientLimit
			}

			var res models.TransactionResponse

			res.Balance = record.AsMap()["saldo"].(int64)
			res.Limit = record.AsMap()["limite"].(int64)

			return res, nil
		})

	if err != nil {
		switch {
		case errors.Is(err, models.ErrUserNotFound):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusUnprocessableEntity)
		}

		return
	}

	res, _ := json.Marshal(result)
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}
