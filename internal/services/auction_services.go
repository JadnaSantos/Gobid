package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageKind int

const (
	PlaceBid MessageKind = iota
	FailedToPlaceBid
	SucessfullyPlaceBid
	NewBidPlaced
	AuctionFinished
	InvalidJSON
)

const (
	maxMessageSize = 512
	readDeadLine   = 60 * time.Second
	pingPerid      = (readDeadLine * 9) / 10
	writeWait      = 10 * time.Second
)

type Message struct {
	Message string      `json:"message,omitempty"`
	Amount  float64     `json:"amount,omitempty"`
	Kind    MessageKind `json:"kind"`
	UserID  uuid.UUID   `json:"user_id,omitempty"`
}

type AuctionLobby struct {
	sync.Mutex
	Rooms map[uuid.UUID]*AuctionRoom
}

type AuctionRoom struct {
	Id         uuid.UUID
	Context    context.Context
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Clients    map[uuid.UUID]*Client

	BidsService BidsService
}

func (r *AuctionRoom) registerClient(c *Client) {
	slog.Info("New user connected", "Client", c)
	r.Clients[c.UserId] = c

}

func (r *AuctionRoom) unregisterClient(c *Client) {
	slog.Info("	User disconnected", "Client", c)
	delete(r.Clients, c.UserId)
}

func (r *AuctionRoom) broadscastMessage(m Message) {
	slog.Info("New message recive", "RoomId", r.Id, "message", m.Message, "user_id", m.UserID)
	switch m.Kind {
	case PlaceBid:
		r.BidsService.Placebid(r.Context, r.Id, m.UserID, m.Amount)

		bid, err := r.BidsService.Placebid(r.Context, r.Id, m.UserID, m.Amount)

		if err != nil {
			if errors.Is(err, ErrBidIsTooLow) {
				if client, ok := r.Clients[m.UserID]; ok {
					client.Send <- Message{Kind: FailedToPlaceBid, Message: ErrBidIsTooLow.Error()}
				}
				return
			}
		}

		if client, ok := r.Clients[m.UserID]; ok {
			client.Send <- Message{
				Kind:    SucessfullyPlaceBid,
				Message: "Your bid was successfully placed.",
				UserID:  m.UserID,
			}
		}

		for id, client := range r.Clients {
			newBidMessage := Message{
				Kind:    NewBidPlaced,
				Message: "A new bid was placed",
				Amount:  bid.BidAmount,
				UserID:  m.UserID,
			}
			if id == m.UserID {
				continue
			}
			client.Send <- newBidMessage
		}
	case InvalidJSON:
		client, ok := r.Clients[m.UserID]

		if !ok {
			slog.Info(
				"Client not foud in hashmap",
				"user_id",
				m.UserID,
			)
			return
		}
		client.Send <- m
	}

}

func (r *AuctionRoom) Run() {
	defer func() {
		close(r.Broadcast)
		close(r.Register)
		close(r.Unregister)
	}()

	for {
		select {
		case client := <-r.Register:
			r.registerClient(client)
		case client := <-r.Unregister:
			r.unregisterClient(client)
		case message := <-r.Broadcast:
			r.broadscastMessage(message)
		case <-r.Context.Done():
			slog.Info("Auction has ended", "auctionID", r.Id)

			for _, client := range r.Clients {
				client.Send <- Message{Kind: AuctionFinished, Message: "auction has been finished"}
			}

			return
		}
	}
}

func NewAuctionRoom(ctx context.Context, id uuid.UUID, BidsService BidsService) *AuctionRoom {
	return &AuctionRoom{
		Id:          id,
		Broadcast:   make(chan Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Clients:     make(map[uuid.UUID]*Client),
		Context:     ctx,
		BidsService: BidsService,
	}
}

type Client struct {
	Room   *AuctionRoom
	Conn   *websocket.Conn
	Send   chan Message
	UserId uuid.UUID
}

func NewClient(room *AuctionRoom, conn *websocket.Conn, userId uuid.UUID) *Client {
	return &Client{
		Room:   room,
		Conn:   conn,
		Send:   make(chan Message, 512),
		UserId: userId,
	}
}

func (c *Client) ReadEventLoop() {
	defer func() {
		c.Room.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(readDeadLine))
	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(readDeadLine))
		return nil
	})

	for {
		var m Message
		m.UserID = c.UserId
		err := c.Conn.ReadJSON(&m)

		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err, websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Error("Unexpeced Close error", "error", err)
			}

			c.Room.Broadcast <- Message{
				Kind:    InvalidJSON,
				Message: "this messege should be a valid json",
				UserID:  m.UserID,
			}
			continue
		}

		c.Room.Broadcast <- m
	}
}

func (c *Client) WriteEventLoop() {
	ticker := time.NewTicker(pingPerid)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteJSON(Message{
					Kind:    websocket.CloseMessage,
					Message: "closing websocket conn",
				})
				return
			}

			if message.Kind == AuctionFinished {
				close(c.Send)
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			err := c.Conn.WriteJSON(message)

			if err != nil {
				c.Room.Unregister <- c
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("Unexpected write error", "error", err)
				return
			}
		}
	}
}
