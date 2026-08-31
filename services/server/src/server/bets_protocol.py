from typing import Literal
import safe_socket
from lottery import Bet

MSG_TYPE_BET = 0
MSG_TYPE_MULTI_BETS = 1
MSG_TYPE_REQUEST_WINNERS = 2
MSG_TYPE_WINNER = 3

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
        payload_len, msg_type = self._receive_header()
        if msg_type is None or payload_len is None:
            return None, None

        if msg_type == MSG_TYPE_MULTI_BETS:
            bets = self._receive_multi_bets(payload_len)
            if bets is None:
                return None, None
            return MSG_TYPE_MULTI_BETS, bets

        if msg_type == MSG_TYPE_BET:
            bet = self._receive_bet(payload_len)
            if bet is None:
                return None, None
            return MSG_TYPE_BET, [bet]

        if msg_type == MSG_TYPE_REQUEST_WINNERS:
            return MSG_TYPE_REQUEST_WINNERS, None

        payload = self._receive_payload(payload_len)
        return msg_type, payload

    def _receive_header(self) -> tuple[int, int] | tuple[None, None]:
        header = safe_socket.recv_all(self.socket, HEADER_SIZE)
        if not header or len(header) < HEADER_SIZE:
            return None, None

        payload_len = int.from_bytes(header[:HEADER_PAYLOAD_LEN_SIZE], BYTE_ORDER)
        msg_type = header[HEADER_PAYLOAD_LEN_SIZE]
        return payload_len, msg_type

    def _receive_payload(self, payload_len: int) -> bytes | None:
        if payload_len == 0:
            return b""
        payload = safe_socket.recv_all(self.socket, payload_len)
        if len(payload) < payload_len:
            return None
        return payload

    def _receive_bet(self, payload_len: int) -> Bet | None:
        payload = self._receive_payload(payload_len)
        if payload is None:
            return None
        return self._deserialize_bet(payload.decode(ENCODING))

    def _receive_multi_bets(self, payload_len: int) -> list[Bet] | None:
        num_bets_bytes = safe_socket.recv_all(self.socket, HEADER_NO_BETS_SIZE)
        if not num_bets_bytes or len(num_bets_bytes) < HEADER_NO_BETS_SIZE:
            return None
        number_of_bets = int.from_bytes(num_bets_bytes, BYTE_ORDER)

        payload = self._receive_payload(payload_len)
        if payload is None:
            return None
        return self._deserialize_bets(payload.decode(ENCODING), number_of_bets)

    def send_winner(self, bet: Bet):
        msg = self._serialize_bet(bet)
        safe_socket.send_all(self.socket, msg)

    def _deserialize_bet(self, payload: str) -> Bet:
        parts = payload.split(DELIMITER)
        return Bet(
            agency_id=int(parts[0]),
            first_name=parts[1],
            last_name=parts[2],
            document=int(parts[3]),
            birthdate=parts[4],
            number=int(parts[5]),
        )

    def _deserialize_bets(self, payload: str, number_of_bets: int) -> list[Bet]:
        parts = payload.split(DELIMITER)
        if not parts or len(parts) < 1:
            return []

        agency_id = int(parts[0])
        num_bets = number_of_bets
        bets = []

        for i in range(num_bets):
            offset = 1 + i * BET_FIELDS_AMOUNT
            if offset + BET_FIELDS_AMOUNT > len(parts):
                break
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

    def _serialize_bet(self, bet: Bet) -> bytes:
        payload = f"{bet.agency_id}{DELIMITER}{bet.first_name}{DELIMITER}{bet.last_name}{DELIMITER}{bet.document}{DELIMITER}{bet.birthdate}{DELIMITER}{bet.number}".encode(ENCODING)
        header = len(payload).to_bytes(HEADER_PAYLOAD_LEN_SIZE, BYTE_ORDER) + bytes([MSG_TYPE_WINNER])
        return header + payload
