package client

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

type BetsReader struct {
	agency_id int
	file      *os.File
	scanner   *bufio.Scanner
}

func newBetsReader(agency_id int, BetsFile string) (*BetsReader, error) {
	betsFile, err := os.Open(BetsFile)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(betsFile)

	return &BetsReader{
		agency_id,
		betsFile,
		scanner,
	}, nil
}

func (reader BetsReader) NextBet() (*Bet, error) {
	if reader.scanner.Scan() {
		readBet := reader.scanner.Text()
		return reader.newBetFromText(readBet)
	}
	if err := reader.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (reader BetsReader) Close() error {
	return reader.file.Close()
}

func (reader BetsReader) newBetFromText(readBet string) (*Bet, error) {
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
