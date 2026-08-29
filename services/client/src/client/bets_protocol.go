package client

import (
	"fmt"
	"io"
)

const (
	MSG_TYPE_BET             = 0
	MSG_TYPE_REQUEST_WINNERS = 1
	MSG_TYPE_WINNER          = 2
	DELIMITER                = "|"
)

type BetsProtocol struct {
	conn Connection
}

func NewBetsProtocol(conn Connection) *BetsProtocol {
	return &BetsProtocol{conn}
}

func (betsProtocol *BetsProtocol) SendBet(bet *Bet) error {
	if bet == nil {
		return nil
	}
	message := betsProtocol.buildBetMessage(bet)
	return betsProtocol.conn.SendAll(message)
}

func (betsProtocol *BetsProtocol) ReceiveWinners() ([]*Bet, error) {
	if err := betsProtocol.requestWinners(); err != nil {
		return nil, err
	}

	winners := make([]*Bet, 0)
	for {
		break
		winner, err := betsProtocol.receiveWinner()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err // O devolver los ganadores hasta ahora, revisar!!!
		}

		winners = append(winners, winner)
	}

	return winners, nil
}

func (betsProtocol *BetsProtocol) requestWinners() error {
	message := betsProtocol.buildRequestWinnersMessage()
	return betsProtocol.conn.SendAll(message)
}

func (betsProtocol *BetsProtocol) receiveWinner() (*Bet, error) {
	return nil, nil
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
	insertUint32IntoByteArray(message[0:4], payloadLen)
	message[4] = MSG_TYPE_BET
	copy(message[5:], payloadBytes)

	return message
}

func (BetsProtocol) buildRequestWinnersMessage() []byte {
	message := make([]byte, 5)
	insertUint32IntoByteArray(message[0:4], 0) // payloadLen = 0 (there's no payload)
	message[4] = MSG_TYPE_REQUEST_WINNERS      // Type
	return message
}

func insertUint32IntoByteArray(array []byte, integer uint32) {
	array[0] = byte(integer >> 24)
	array[1] = byte(integer >> 16)
	array[2] = byte(integer >> 8)
	array[3] = byte(integer)
}
