package client

import (
	"bufio"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 500 // TODO: Change to an appropiate back-off algorithm

const ECHO_CLIENT_BUFFER_SIZE = 512
const ECHO_CLIENT_MESSAGE_AMOUNT = 3
const ECHO_CLIENT_MESSAGE_DELAY_MS = 1000

const FILE_MESSAGE_BUFFER_SIZE = 2048

const ACTION_TEST_ECHO_SERVER = "test-echo-server"
const ACTION_PROCESS_FILE_MESSAGE = "process_file_message"

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

	/*
		if err := test_echo_server(client); err != nil {
			// The error has already been logged within the function
			return err
		}

		if err := process_file_messages(client); err != nil {
			return err
		}
	*/

	if err := processBets(client); err != nil {
		return err
	}

	if err := receiveWinners(client); err != nil {
		return err
	}

	return nil
}

func test_echo_server(client *Client) error {
	const mainAction = ACTION_TEST_ECHO_SERVER

	for messageId := range ECHO_CLIENT_MESSAGE_AMOUNT {
		messageArgs := []any{"agency-id", client.config.AgencyId, "message-id", messageId}
		logger.Info(mainAction, logger.InProgress, messageArgs...)

		clientMessage := client.config.AgencyId

		if err := send_client_message(client.conn, []byte(clientMessage), messageArgs...); err != nil {
			return err
		}

		responseBuffer, err := receive_client_message(client.conn, len(clientMessage), messageArgs...)
		if err != nil {
			return err
		}

		if string(responseBuffer) != clientMessage {
			logger.Error("check-response", logger.Fail, messageArgs...)
			return err
		}

		time.Sleep(ECHO_CLIENT_MESSAGE_DELAY_MS * time.Millisecond)
	}
	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)

	return nil
}

func processBets(client *Client) error {
	agencyId, err := strconv.Atoi(client.config.AgencyId)
	if err != nil {
		return err
	}

	betsProtocol := newBetsProtocol(client.conn)
	betsReader, err := newBetsReader(agencyId, client.config.InputFile)
	if err != nil {
		return err
	}
	defer betsReader.Close()

	for {
		bet, err := betsReader.NextBet()
		if err == io.EOF {
			break // No hay más apuestas para enviar
		}
		if err != nil {
			return err
		}

		if err := betsProtocol.SendBet(bet); err != nil {
			return err
		}
	}

	return nil
}

func receiveWinners(client *Client) error {
	betsProtocol := newBetsProtocol(client.conn)
	betsProtocol.ReceiveWinners()

	return nil
}

func process_file_messages(client *Client) error { // Exercise 3
	const mainAction = ACTION_PROCESS_FILE_MESSAGE

	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		logger.Error("open-input-file", logger.Fail, "err", err)
		return err
	}

	defer inputFile.Close()

	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		logger.Error("open-output-file", logger.Fail, "err", err)
		return err
	}

	defer outputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	messageId := ECHO_CLIENT_MESSAGE_AMOUNT
	for scanner.Scan() {
		clientMessage := scanner.Text()

		if err := scanner.Err(); err != nil {
			logger.Error("scan-message-text", logger.Fail, "err", err)
			return err
		}

		messageArgs := []any{"agency-id", client.config.AgencyId, "message-id", messageId}
		logger.Info(mainAction, logger.InProgress, messageArgs...)

		if err := send_client_message(client.conn, []byte(clientMessage), messageArgs...); err != nil {
			return err
		}

		responseBuffer, err := receive_client_message(client.conn, len(clientMessage), messageArgs...)
		if err != nil {
			return err
		}

		_, err = outputFile.Write(responseBuffer)
		if err != nil {
			logger.Error("write-response", logger.Fail, messageArgs...)
			return err
		}

		_, err = outputFile.WriteString("\n")
		if err != nil {
			logger.Error("write-response", logger.Fail, messageArgs...)
			return err
		}

		messageId++
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)
	return nil
}

func send_client_message(socket io.Writer, message []byte, args ...any) error {
	if err := safe_socket.SendAll(socket, message); err != nil {
		logger.Error("send-message", logger.Fail, args)
		return err
	}

	return nil
}

func receive_client_message(socket io.Reader, buffer_size int, args ...any) ([]byte, error) {
	responseBuffer, err := safe_socket.RecvAll(socket, buffer_size)
	if err != nil {
		logger.Error("recv-response", logger.Fail, args...)
		return nil, err
	}
	return responseBuffer, nil
}
