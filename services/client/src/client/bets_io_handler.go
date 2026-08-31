package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	CSV_SEPARATOR          = ","
	BET_FILE_FIELDS_AMOUNT = 5
	MSG_ERROR_INVALID_LINE = "The line does not have enough parts to represent a bet"
)

type BetsIOHandler struct {
	agency_id   int
	input_file  *os.File
	output_file *os.File
	scanner     *bufio.Scanner
}

func NewBetsIOHandler(agency_id int, InputFile string, OutputFile string) (*BetsIOHandler, error) {
	inputFile, err := os.Open(InputFile)
	if err != nil {
		return nil, err
	}

	outputFile, err := os.Create(OutputFile)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(inputFile)

	return &BetsIOHandler{
		agency_id,
		inputFile,
		outputFile,
		scanner,
	}, nil
}

func (reader BetsIOHandler) ReadNextBet() (*Bet, error) {
	if reader.scanner.Scan() {
		readBet := reader.scanner.Text()
		return reader.newBetFromText(readBet)
	}
	if err := reader.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (reader *BetsIOHandler) WriteBet(bet *Bet) error {
	line := fmt.Sprintf("%s%s%s%s%d%s%s%s%d\n", bet.first_name, CSV_SEPARATOR, bet.last_name, CSV_SEPARATOR, bet.document, CSV_SEPARATOR, bet.birthdate, CSV_SEPARATOR, bet.number)
	_, err := reader.output_file.WriteString(line)
	return err
}

func (reader *BetsIOHandler) Close() error {
	errInput := reader.input_file.Close()
	errOutput := reader.output_file.Close()

	if errInput != nil {
		return errInput
	}
	return errOutput
}

func (reader BetsIOHandler) newBetFromText(readBet string) (*Bet, error) {
	line := strings.TrimSpace(readBet)
	if line == "" {
		return nil, io.EOF
	}

	parts := strings.Split(line, CSV_SEPARATOR)
	if len(parts) != BET_FILE_FIELDS_AMOUNT {
		return nil, errors.New(MSG_ERROR_INVALID_LINE)
	}

	firstName := strings.TrimSpace(parts[0])
	lastName := strings.TrimSpace(parts[1])

	document, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, err
	}

	birthdate := strings.TrimSpace(parts[3])

	number, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return nil, err
	}

	return &Bet{
		agency_id:  reader.agency_id,
		first_name: firstName,
		last_name:  lastName,
		document:   document,
		birthdate:  birthdate,
		number:     number,
	}, nil
}
