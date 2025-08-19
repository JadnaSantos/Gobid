package api

import (
	"errors"
	"net/http"

	"github.com/JadnaSantos/Gobid.git/internal/jsonutils"
	"github.com/JadnaSantos/Gobid.git/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (api *Api) handleSubscribeUserToAuction(w http.ResponseWriter, r *http.Request) {
	rawProductId := chi.URLParam(r, "product_id")

	product_id, err := uuid.Parse(rawProductId)

	if err != nil {
		jsonutils.EncodeJson(w, r, http.StatusBadRequest, map[string]any{
			"message": "Invalid product id - must be a valid uuid",
		})

		return
	}

	_, err = api.ProductService.GetProductById(r.Context(), product_id)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			jsonutils.EncodeJson(w, r, http.StatusNotFound, map[string]any{
				"message": "no product with given id",
			})
			return
		}
		jsonutils.EncodeJson(w, r, http.StatusInternalServerError, map[string]any{
			"message": "unexpected error, try again later.",
		})
		return
	}

	userId, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserId").(uuid.UUID)
	if !ok {
		jsonutils.EncodeJson(w, r, http.StatusInternalServerError, map[string]any{
			"message": "unexpected error, try again later.",
		})
		return
	}

	api.AuctionLobby.Lock()
	room, ok := api.AuctionLobby.Rooms[product_id]
	api.AuctionLobby.Unlock()

	if !ok {
		jsonutils.EncodeJson(w, r, http.StatusBadRequest, map[string]any{
			"message": "the auction has ended",
		})
	}

	conn, err := api.WsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		jsonutils.EncodeJson(w, r, http.StatusInternalServerError, map[string]any{
			"message": "could not upgrade connection to a websocket protocol",
		})
		return
	}

	client := services.NewClient(room, conn, userId)

	room.Register <- client

	go client.ReadEventLoop()
	go client.WriteEventLoop()
}
