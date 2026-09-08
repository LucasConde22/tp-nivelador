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
	MSG_TYPE_MULTI_BETS      = 1
	MSG_TYPE_REQUEST_WINNERS = 2
	MSG_TYPE_WINNER          = 3
	MSG_TYPE_ACK             = 4
	DELIMITER                = "|"

	HEADER_PAYLOAD_LEN_SIZE = 4
	HEADER_TYPE_SIZE        = 1
	HEADER_SIZE             = HEADER_PAYLOAD_LEN_SIZE + HEADER_TYPE_SIZE

	HEADER_NO_BETS_SIZE    = 4                                 // Bytes indicating the number of bets
	HEADER_MULTI_BETS_SIZE = HEADER_SIZE + HEADER_NO_BETS_SIZE // Header size when multiple bets are sent

	BET_PAYLOAD_PARTS_AMOUNT = 6

	MSG_ERROR_COULD_NOT_BUILD_MSG = "Message could not be built"
	MSG_ERROR_DID_NOT_RECEIVE_ACK = "Didn't receive ack message"
)

// BetsProtocol handles the communication protocol for sending and receiving bets over a connection.
type BetsProtocol struct {
	conn Connection
}

// NewBetsProtocol creates a new BetsProtocol instance with the given connection.
func NewBetsProtocol(conn Connection) *BetsProtocol {
	return &BetsProtocol{conn}
}

// SendBet sends a single bet over the connection.
func (betsProtocol *BetsProtocol) SendBet(bet *Bet) error { // Used for exercise 5
	message := betsProtocol.buildBetMessage(bet)
	if message == nil {
		return errors.New(MSG_ERROR_COULD_NOT_BUILD_MSG)
	}

	return betsProtocol.conn.SendAll(message)
}

// SendBets sends multiple bets over the connection.
func (betsProtocol *BetsProtocol) SendBets(bets []*Bet) error {
	message := betsProtocol.buildBetsMessage(bets)
	if message == nil {
		return errors.New(MSG_ERROR_COULD_NOT_BUILD_MSG)
	}

	if err := betsProtocol.conn.SendAll(message); err != nil {
		return err
	}

	return betsProtocol.receiveAck()
}

// ReceiveWinners requests and receives the list of winning bets from the server.
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
			return nil, err // Discards "partial" winners.
		}

		winners = append(winners, winner)
	}

	return winners, nil
}

// requestWinners sends a request to the server to get the list of winning bets.
func (betsProtocol *BetsProtocol) requestWinners() error {
	message := betsProtocol.buildRequestWinnersMessage()
	return betsProtocol.conn.SendAll(message)
}

// receiveWinner receives a single winning bet from the server.
func (betsProtocol *BetsProtocol) receiveWinner() (*Bet, error) {
	header, err := betsProtocol.receiveHeader()
	if err != nil {
		return nil, err
	}

	payloadLen := extractUint32FromByteArray(header[:HEADER_PAYLOAD_LEN_SIZE])

	payloadBytes, err := betsProtocol.conn.RecvAll(int(payloadLen))
	if err != nil {
		return nil, err
	}

	return betsProtocol.parseBetFromPayload(string(payloadBytes))
}

// receiveAck waits for an ack message from the server after sending bets.
func (betsProtocol *BetsProtocol) receiveAck() error {
	header, err := betsProtocol.receiveHeader()
	if err != nil {
		return errors.New(MSG_ERROR_DID_NOT_RECEIVE_ACK)
	}

	msgType := header[HEADER_PAYLOAD_LEN_SIZE]

	if msgType == MSG_TYPE_ACK {
		return nil
	}

	return errors.New(MSG_ERROR_DID_NOT_RECEIVE_ACK)
}

// receiveHeader receives the header of a message from the server, which includes the payload length and message type.
func (betsProtocol *BetsProtocol) receiveHeader() ([]byte, error) {
	return betsProtocol.conn.RecvAll(HEADER_SIZE)
}

// parseBetFromPayload parses a Bet struct from a payload string received from the server.
func (BetsProtocol) parseBetFromPayload(payload string) (*Bet, error) {
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

// formatBetFields formats the fields of a Bet struct into a string to send.
func (BetsProtocol) formatBetFields(bet *Bet) string {
	return fmt.Sprintf("%s%s%s%s%d%s%s%s%d",
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
}

// buildBetMessage constructs a message to send a single bet over the connection, including the header and payload.
func (betsProtocol BetsProtocol) buildBetMessage(bet *Bet) []byte {
	if bet == nil {
		return nil
	}

	payload := fmt.Sprintf("%d%s%s", bet.agency_id, DELIMITER, betsProtocol.formatBetFields(bet))
	payloadBytes := []byte(payload)
	payloadLen := uint32(len(payloadBytes))

	// [4 bytes: Payload lenght|1 byte: Type|N Bytes: Payload]
	message := make([]byte, HEADER_SIZE+len(payloadBytes))
	insertUint32IntoByteArray(message[0:HEADER_PAYLOAD_LEN_SIZE], payloadLen)
	message[HEADER_PAYLOAD_LEN_SIZE] = MSG_TYPE_BET
	copy(message[HEADER_SIZE:], payloadBytes)

	return message
}

// buildBetsMessage constructs a message to send multiple bets over the connection, including the header and payload.
func (betsProtocol BetsProtocol) buildBetsMessage(bets []*Bet) []byte {
	if len(bets) == 0 {
		return nil
	}

	betStrings := make([]string, len(bets))
	for i, bet := range bets {
		betStrings[i] = betsProtocol.formatBetFields(bet)
	}

	payload := fmt.Sprintf("%d%s%s", bets[0].agency_id, DELIMITER, strings.Join(betStrings, DELIMITER))
	payloadBytes := []byte(payload)
	payloadLen := uint32(len(payloadBytes))
	numberOfBets := uint32(len(bets))

	// [4 bytes: Payload lenght|1 byte: Type|4 bytes: Number of bets|N Bytes: Payload]
	message := make([]byte, HEADER_MULTI_BETS_SIZE+len(payloadBytes))
	insertUint32IntoByteArray(message[0:HEADER_PAYLOAD_LEN_SIZE], payloadLen)
	message[HEADER_PAYLOAD_LEN_SIZE] = MSG_TYPE_MULTI_BETS
	insertUint32IntoByteArray(message[HEADER_SIZE:HEADER_MULTI_BETS_SIZE], numberOfBets)
	copy(message[HEADER_MULTI_BETS_SIZE:], payloadBytes)

	return message
}

// buildRequestWinnersMessage constructs a message to request the list of winning bets from the server.
func (BetsProtocol) buildRequestWinnersMessage() []byte {
	message := make([]byte, HEADER_SIZE)
	insertUint32IntoByteArray(message[0:HEADER_PAYLOAD_LEN_SIZE], 0) // payloadLen = 0 (there's no payload)
	message[HEADER_PAYLOAD_LEN_SIZE] = MSG_TYPE_REQUEST_WINNERS      // Type
	return message
}

// extractUint32FromByteArray extracts a uint32 integer from a byte array.
func extractUint32FromByteArray(array []byte) uint32 {
	return uint32(array[0])<<24 | uint32(array[1])<<16 | uint32(array[2])<<8 | uint32(array[3])
}

// insertUint32IntoByteArray inserts a uint32 integer into a byte array.
func insertUint32IntoByteArray(array []byte, integer uint32) {
	array[0] = byte(integer >> 24)
	array[1] = byte(integer >> 16)
	array[2] = byte(integer >> 8)
	array[3] = byte(integer)
}
