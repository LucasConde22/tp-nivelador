from typing import Literal
import safe_socket
from lottery import Bet

MSG_TYPE_BET = 0
MSG_TYPE_MULTI_BETS = 1
MSG_TYPE_REQUEST_WINNERS = 2
MSG_TYPE_WINNER = 3
MSG_TYPE_ACK = 4

DELIMITER = "|"
ENCODING = "utf-8"

HEADER_PAYLOAD_LEN_SIZE = 4
HEADER_TYPE_SIZE = 1
HEADER_NO_BETS_SIZE = 4
HEADER_SIZE = HEADER_PAYLOAD_LEN_SIZE + HEADER_TYPE_SIZE
BYTE_ORDER: Literal["big", "little"] = "big"

BET_PAYLOAD_FIELDS_AMOUNT = 6
BET_FIELDS_AMOUNT = 5

class BetsProtocol:
    def __init__(self, socket) -> None:
        self.socket = socket

    def receive_message(self):
        """
        Receives a message from the client.
        """
        payload_len, msg_type = self._receive_header()
        if msg_type is None or payload_len is None:
            return None, None

        if msg_type == MSG_TYPE_MULTI_BETS:
            bets = self._receive_multi_bets(payload_len)
            if bets is None:
                return None, None
            return MSG_TYPE_MULTI_BETS, bets

        if msg_type == MSG_TYPE_BET: # Used for exercise 5
            bet = self._receive_bet(payload_len)
            if bet is None:
                return None, None
            return MSG_TYPE_BET, [bet]

        if msg_type == MSG_TYPE_REQUEST_WINNERS:
            return MSG_TYPE_REQUEST_WINNERS, None

        payload = self._receive_payload(payload_len)
        return msg_type, payload

    def send_winner(self, bet: Bet) -> int | None:
        """
        Sends a winner message to the client.
        """
        msg = self._serialize_bet(bet)
        safe_socket.send_all(self.socket, msg)

    def send_ack(self) -> int | None:
        """
        Sends an acknowledgment message to the client.
        """
        msg = self._build_ack_msg()
        safe_socket.send_all(self.socket, msg)

    def _receive_header(self) -> tuple[int, int] | tuple[None, None]:
        """
        Receives the header of a message from the client.
        """
        header = safe_socket.recv_all(self.socket, HEADER_SIZE)
        if not header or len(header) < HEADER_SIZE:
            return None, None

        payload_len = int.from_bytes(header[:HEADER_PAYLOAD_LEN_SIZE], BYTE_ORDER)
        msg_type = header[HEADER_PAYLOAD_LEN_SIZE]
        return payload_len, msg_type

    def _receive_payload(self, payload_len: int) -> bytes | None:
        """
        Receives the payload of a message from the client.
        """
        if payload_len == 0:
            return b""
        payload = safe_socket.recv_all(self.socket, payload_len)
        if len(payload) < payload_len:
            return None
        return payload

    def _receive_bet(self, payload_len: int) -> Bet | None:
        """
        Receives a single bet from the client.
        """
        payload = self._receive_payload(payload_len)
        if payload is None:
            return None
        return self._deserialize_bet(payload.decode(ENCODING))

    def _receive_multi_bets(self, payload_len: int) -> list[Bet] | None:
        """
        Receives multiple bets from the client.
        """
        num_bets_bytes = safe_socket.recv_all(self.socket, HEADER_NO_BETS_SIZE)
        if not num_bets_bytes or len(num_bets_bytes) < HEADER_NO_BETS_SIZE:
            return None
        number_of_bets = int.from_bytes(num_bets_bytes, BYTE_ORDER)

        payload = self._receive_payload(payload_len)
        if payload is None:
            return None
        bets = self._deserialize_bets(payload.decode(ENCODING), number_of_bets)
        if bets is None or len(bets) != number_of_bets:
            return None
        return bets

    def _build_ack_msg(self) -> bytes:
        """
        Builds an ack message to be sent to the client.
        """
        return (0).to_bytes(HEADER_PAYLOAD_LEN_SIZE, BYTE_ORDER) + bytes([MSG_TYPE_ACK])

    def _deserialize_bet(self, payload: str) -> Bet | None:
        """
        Deserializes a bet from a string.
        """
        try:
            parts = payload.split(DELIMITER)
            if len(parts) != BET_PAYLOAD_FIELDS_AMOUNT:
                return None
            return Bet(
                agency_id=int(parts[0]),
                first_name=parts[1],
                last_name=parts[2],
                document=int(parts[3]),
                birthdate=parts[4],
                number=int(parts[5]),
            )
        except (ValueError, IndexError):
            return None

    def _deserialize_bets(self, payload: str, number_of_bets: int) -> list[Bet] | None:
        """
        Deserializes multiple bets from a string.
        """
        try:
            parts = payload.split(DELIMITER)
            expected_parts = 1 + number_of_bets * BET_FIELDS_AMOUNT
            if len(parts) != expected_parts:
                return None

            agency_id = int(parts[0])
            bets = []

            for i in range(number_of_bets):
                offset = 1 + i * BET_FIELDS_AMOUNT
                bet = Bet(
                    agency_id=agency_id,
                    first_name=parts[offset],
                    last_name=parts[offset + 1],
                    document=int(parts[offset + 2]),
                    birthdate=parts[offset + 3],
                    number=int(parts[offset + 4]),
                )
                bets.append(bet)

            return bets
        except (ValueError, IndexError):
            return None

    def _serialize_bet(self, bet: Bet) -> bytes:
        """
        Serializes a bet into bytes.
        """
        payload = f"{bet.agency_id}{DELIMITER}{bet.first_name}{DELIMITER}{bet.last_name}{DELIMITER}{bet.document}{DELIMITER}{bet.birthdate}{DELIMITER}{bet.number}".encode(ENCODING)
        header = len(payload).to_bytes(HEADER_PAYLOAD_LEN_SIZE, BYTE_ORDER) + bytes([MSG_TYPE_WINNER])
        return header + payload
