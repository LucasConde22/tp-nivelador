import safe_socket
from lottery import Bet

MSG_TYPE_BET = 0
MSG_TYPE_REQUEST_WINNERS = 1
MSG_TYPE_WINNER = 2
DELIMITER = "|"

class BetsProtocol:
    def __init__(self, socket) -> None:
        self.socket = socket

    def receive_message(self):
        header = safe_socket.recv_all(self.socket, 5)
        if not header or len(header) < 5:
            return None, None

        payload_len = int.from_bytes(header[:4], "big")
        msg_type = header[4]

        payload = b""
        if payload_len > 0:
            payload = safe_socket.recv_all(self.socket, payload_len)
            if len(payload) < payload_len:
                return None, None

        if msg_type == MSG_TYPE_BET:
            bet = self._deserialize_bet(payload.decode("utf-8"))
            return MSG_TYPE_BET, bet
        elif msg_type == MSG_TYPE_REQUEST_WINNERS:
            return MSG_TYPE_REQUEST_WINNERS, None

        return msg_type, payload

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

    def _serialize_bet(self, bet: Bet) -> bytes:
        payload = f"{bet.agency_id}{DELIMITER}{bet.first_name}{DELIMITER}{bet.last_name}{DELIMITER}{bet.document}{DELIMITER}{bet.birthdate}{DELIMITER}{bet.number}".encode("utf-8")
        header = len(payload).to_bytes(4, "big") + bytes([MSG_TYPE_WINNER])
        return header + payload
