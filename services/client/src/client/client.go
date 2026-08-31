package client

import (
	"io"
	"net"
	"strconv"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 500 // TODO: Change to an appropiate back-off algorithm

const ACTION_PROCESS_BETS = "process-bets"
const ACTION_SEND_BETS = "process-bets"
const ACTION_RECEIVE_WINNERS = "process-bets"

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	client := &Client{conn: conn, config: config}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
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

	betsProtocol := NewBetsProtocol(newSocketConnection(client.conn))
	betsIOHandler, err := NewBetsIOHandler(agencyId, client.config.InputFile, client.config.OutputFile)
	if err != nil {
		logger.Error("open-I/O-files", logger.Fail, "err", err, "agency-id", agencyIdStr)
		return err
	}
	defer betsIOHandler.Close()

	logger.Info(mainAction, logger.InProgress, "agency-id", agencyIdStr)

	if err := sendBets(betsIOHandler, betsProtocol, agencyIdStr); err != nil {
		return err
	}

	if err := receiveWinners(betsIOHandler, betsProtocol, agencyIdStr); err != nil {
		return err
	}

	logger.Info(mainAction, logger.Success, "agency-id", agencyIdStr)
	return nil
}

func sendBets(betsReader *BetsIOHandler, betsProtocol *BetsProtocol, agencyId string) error {
	const action = ACTION_SEND_BETS
	logger.Info(action, logger.InProgress, "agency-id", agencyId)

	betsSent := 0
	for {
		bet, err := betsReader.ReadNextBet()
		if err == io.EOF {
			break // No hay más apuestas para enviar
		}
		if err != nil {
			logger.Error("read-bet", logger.Fail, "err", err, "agency-id", agencyId)
			return err
		}

		if err := betsProtocol.SendBet(bet); err != nil {
			logger.Error("send-bet", logger.Fail, "err", err, "agency-id", agencyId)
			return err
		}
		betsSent++
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
