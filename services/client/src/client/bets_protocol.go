package client

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const (
	MSG_TYPE_BET             = 0
	MSG_TYPE_REQUEST_WINNERS = 1
	DELIMITER                = "|"
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
	message := betsProtocol.buildBetMessage(bet)
	return safe_socket.SendAll(betsProtocol.conn, message)
}

func (betsProtocol *BetsProtocol) ReceiveWinners() ([]*Bet, error) {
	if err := betsProtocol.requestWinners(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (betsProtocol *BetsProtocol) requestWinners() error {
	message := make([]byte, 5)
	binary.BigEndian.PutUint32(message[0:4], 0) // payloadLen = 0 (there's no payload)
	message[4] = MSG_TYPE_REQUEST_WINNERS       // Type
	return safe_socket.SendAll(betsProtocol.conn, message)
}

func (BetsProtocol) buildBetMessage(bet *Bet) []byte {
	payload := fmt.Sprintf("%d%s%s%s%s%s%d%s%s%s%d",
		bet.agency_id,
		DELIMITER,
		bet.first_name,
		DELIMITER,
		bet.last_name,
		DELIMITER,
		bet.document,
		DELIMITER,
		bet.birthdate,
		DELIMITER,
		bet.number,
	)

	payloadBytes := []byte(payload)
	payloadLen := uint32(len(payloadBytes))

	// 4 bytes: Payload lenght|1 byte: Type|Payload
	message := make([]byte, 4+1+len(payloadBytes))
	binary.BigEndian.PutUint32(message[0:4], payloadLen)
	message[4] = MSG_TYPE_BET
	copy(message[5:], payloadBytes)

	return message
}
