package model

import (
	"github.com/gorilla/websocket"
)

// ActiveDiscussion - container for websocket to manage active discussion
type ActiveDiscussion struct {
	DiscussionID int
	MessageChan  chan PrevMessage
	Clients      map[*websocket.Conn]int //value = User.ID
}
