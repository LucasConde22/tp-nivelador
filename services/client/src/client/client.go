package client

import (
	"io"
	"net"
	"strconv"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

const (
	CONNECTION_ATTEMPTS_MAX     = 3
	CONNECTION_ATTEMPS_DELAY_MS = 500 // TODO: Change to an appropiate back-off algorithm
	NETWORK_PROTOCOL            = "tcp"

	ACTION_CONNECT_TO_SERVER = "connect-to-server"
	ACTION_PROCESS_BETS      = "process-bets"
	ACTION_SEND_BETS         = "send-bets"
	ACTION_RECEIVE_WINNERS   = "receive-winners"
)

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn(ACTION_CONNECT_TO_SERVER, logger.Fail)
		return nil, err
	}

	client := &Client{conn: conn, config: config}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	var err error
	var conn net.Conn

	logger.Info(ACTION_CONNECT_TO_SERVER, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial(NETWORK_PROTOCOL, host+":"+port)
		if err != nil {
			logger.Warn(ACTION_CONNECT_TO_SERVER, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(ACTION_CONNECT_TO_SERVER, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	defer client.conn.Close()

	if err := processBets(client); err != nil {
		return err
	}

	return nil
}

func processBets(client *Client) error {
	const mainAction = ACTION_PROCESS_BETS
	agencyIdStr := client.config.AgencyId

	agencyId, err := strconv.Atoi(agencyIdStr)
	if err != nil {
		logger.Error("agencyId", logger.Fail, "err", err, "agency-id", agencyIdStr)
		return err
	}

	batchSize, err := strconv.Atoi(client.config.BatchSize)
	if err != nil {
		logger.Error("batchSize", logger.Fail, "err", err, "agency-id", agencyIdStr)
		return err
	}

	betsProtocol := NewBetsProtocol(newSocketConnection(client.conn))
	betsIOHandler, err := NewBetsIOHandler(agencyId, client.config.InputFile, client.config.OutputFile)
	if err != nil {
		logger.Error("open-I/O-files", logger.Fail, "err", err, "agency-id", agencyIdStr)
		return err
	}
	defer betsIOHandler.Close()

	logger.Info(mainAction, logger.InProgress, "agency-id", agencyIdStr)

	if err := sendBets(betsIOHandler, betsProtocol, agencyIdStr, batchSize); err != nil {
		return err
	}

	if err := receiveWinners(betsIOHandler, betsProtocol, agencyIdStr); err != nil {
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", agencyIdStr)
	return nil
}

func sendBets(betsReader *BetsIOHandler, betsProtocol *BetsProtocol, agencyId string, batchSize int) error {
	const action = ACTION_SEND_BETS
	logger.Info(action, logger.InProgress, "agency-id", agencyId)

	betsSent := 0
	betsToSend := make([]*Bet, 0, batchSize)
	allSent := false

	for {
		betsToSend = betsToSend[:0] // Resets lenght to 0 but keeping its capacity

		for len(betsToSend) < batchSize {
			bet, err := betsReader.ReadNextBet()
			if err == io.EOF {
				allSent = true
				break // No more bets to read
			}
			if err != nil {
				logger.Error("read-bet", logger.Fail, "err", err, "agency-id", agencyId)
				return err
			}

			betsToSend = append(betsToSend, bet)
		}

		if len(betsToSend) > 0 {
			if err := betsProtocol.SendBets(betsToSend); err != nil {
				logger.Error("send-bet", logger.Fail, "err", err, "agency-id", agencyId)
				return err
			}
			betsSent += len(betsToSend)
		}

		if allSent {
			break
		}
	}

	logger.Info(action, logger.Success, "agency-id", agencyId, "bets-amount", betsSent)
	return nil
}

func receiveWinners(betsWriter *BetsIOHandler, betsProtocol *BetsProtocol, agencyId string) error {
	const action = ACTION_RECEIVE_WINNERS
	logger.Info(action, logger.InProgress, "agency-id", agencyId)

	winners, err := betsProtocol.ReceiveWinners()
	if err != nil {
		logger.Error("receive-winners", logger.Fail, "err", err, "agency-id", agencyId)
		return err
	}

	for _, winner := range winners {
		if err := betsWriter.WriteBet(winner); err != nil {
			logger.Error("write-winner", logger.Fail, "err", err, "agency-id", agencyId)
			return err
		}
	}

	logger.Info(action, logger.Success, "agency-id", agencyId, "winners-amount", len(winners))
	return nil
}
