package client

import (
	"fmt"
	"net"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

type BetsProtocol struct {
	conn net.Conn
}

func newBetsProtocol(conn net.Conn) *BetsProtocol {
	return &BetsProtocol{conn}
}

func (betsProtocol *BetsProtocol) SendBet(bet *Bet) error {
	if bet == nil {
		return nil
	}
	msg := betsProtocol.serializeBet(bet)
	return safe_socket.SendAll(betsProtocol.conn, msg)
}

func (BetsProtocol) serializeBet(bet *Bet) []byte {
	const msgType = 0
	const delimiter = "|"

	payload := fmt.Sprintf("%d%s%d%s%s%s%s%s%d%s%s%s%d",
		msgType,
		delimiter,
		bet.agency_id,
		delimiter,
		bet.first_name,
		delimiter,
		bet.last_name,
		delimiter,
		bet.document,
		delimiter,
		bet.birthdate,
		delimiter,
		bet.number,
	)

	message := fmt.Sprintf("%d%s%s", len(payload), delimiter, payload)
	return []byte(message)
}
