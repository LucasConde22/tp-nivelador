package client

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	MSG_TYPE_BET             = 0
	MSG_TYPE_REQUEST_WINNERS = 1
	MSG_TYPE_WINNER          = 2
	DELIMITER                = "|"

	HEADER_PAYLOAD_LEN_SIZE  = 4
	HEADER_TYPE_SIZE         = 1
	HEADER_SIZE              = HEADER_PAYLOAD_LEN_SIZE + HEADER_TYPE_SIZE
	BET_PAYLOAD_PARTS_AMOUNT = 6
)

type BetsProtocol struct {
	conn Connection
}

func NewBetsProtocol(conn Connection) *BetsProtocol {
	return &BetsProtocol{conn}
}

func (betsProtocol *BetsProtocol) SendBet(bet *Bet) error {
	message := betsProtocol.buildBetMessage(bet)
	return betsProtocol.conn.SendAll(message)
}

func (betsProtocol *BetsProtocol) ReceiveWinners() ([]*Bet, error) {
	if err := betsProtocol.requestWinners(); err != nil {
		return nil, err
	}

	winners := make([]*Bet, 0)
	for {
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
	header, err := betsProtocol.conn.RecvAll(HEADER_SIZE)
	if err != nil {
		return nil, err
	}

	payloadLen := extractUint32FromByteArray(header[:HEADER_PAYLOAD_LEN_SIZE])

	payloadBytes, err := betsProtocol.conn.RecvAll(int(payloadLen))
	if err != nil {
		return nil, err
	}

	return parseBetFromPayload(string(payloadBytes))
}

func parseBetFromPayload(payload string) (*Bet, error) {
	parts := strings.Split(payload, DELIMITER)
	if len(parts) != BET_PAYLOAD_PARTS_AMOUNT {
		return nil, errors.New(MSG_ERROR_INVALID_LINE)
	}

	agencyId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	firstName := parts[1]
	lastName := parts[2]

	document, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, err
	}

	birthdate := parts[4]

	number, err := strconv.Atoi(parts[5])
	if err != nil {
		return nil, err
	}

	return &Bet{
		agency_id:  agencyId,
		first_name: firstName,
		last_name:  lastName,
		document:   document,
		birthdate:  birthdate,
		number:     number,
	}, nil
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
	message := make([]byte, HEADER_SIZE+len(payloadBytes))
	insertUint32IntoByteArray(message[0:HEADER_PAYLOAD_LEN_SIZE], payloadLen)
	message[HEADER_PAYLOAD_LEN_SIZE] = MSG_TYPE_BET
	copy(message[HEADER_SIZE:], payloadBytes)

	return message
}

func (BetsProtocol) buildRequestWinnersMessage() []byte {
	message := make([]byte, HEADER_SIZE)
	insertUint32IntoByteArray(message[0:HEADER_PAYLOAD_LEN_SIZE], 0) // payloadLen = 0 (there's no payload)
	message[HEADER_PAYLOAD_LEN_SIZE] = MSG_TYPE_REQUEST_WINNERS      // Type
	return message
}

func extractUint32FromByteArray(array []byte) uint32 {
	return uint32(array[0])<<24 | uint32(array[1])<<16 | uint32(array[2])<<8 | uint32(array[3])
}

func insertUint32IntoByteArray(array []byte, integer uint32) {
	array[0] = byte(integer >> 24)
	array[1] = byte(integer >> 16)
	array[2] = byte(integer >> 8)
	array[3] = byte(integer)
}
