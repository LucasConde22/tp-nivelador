package client

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
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
		logger.Error("open-output-file", logger.Fail, "err", err)
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
	if bet == nil {
		return nil
	}
	line := fmt.Sprintf("%s,%s,%d,%s,%d\n", bet.first_name, bet.last_name, bet.document, bet.birthdate, bet.number)
	_, err := reader.output_file.WriteString(line)
	return err
}

func (reader *BetsIOHandler) Close() error {
	var errInput, errOutput error
	if reader.input_file != nil {
		errInput = reader.input_file.Close()
	}
	if reader.output_file != nil {
		errOutput = reader.output_file.Close()
	}
	if errInput != nil {
		return errInput
	}
	return errOutput
}

func (reader BetsIOHandler) newBetFromText(readBet string) (*Bet, error) {
	line := strings.TrimSpace(readBet)
	if line == "" {
		return nil, io.EOF // TODO: Handlear bien
	}

	parts := strings.Split(line, ",")
	if len(parts) != 5 {
		return nil, io.EOF // TODO: Handlear bien
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
