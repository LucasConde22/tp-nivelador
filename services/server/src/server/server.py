import socket
import logger
from lottery import Lottery
from .bets_protocol import BetsProtocol, MSG_TYPE_BET, MSG_TYPE_REQUEST_WINNERS

_STORAGE_PATH = "/output/bets.csv"


class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str = _STORAGE_PATH) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.storage_path = storage_path
        self.lottery = Lottery(self.storage_path)

    def _handle_client(self, client_socket):
        action = "handle-client"
        message_amount = 0
        protocol = BetsProtocol(client_socket)
        agency_id = None
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                msg_type, data = protocol.receive_message()
                if msg_type is None:
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    return

                message_amount += 1

                if msg_type == MSG_TYPE_BET:
                    agency_id = data.agency_id
                    self.lottery.store_bets([data])
                elif msg_type == MSG_TYPE_REQUEST_WINNERS:
                    for bet in self.lottery.load_bets():
                        if self.lottery.has_won(bet) and (agency_id is None or bet.agency_id == agency_id):
                            protocol.send_winner(bet)
                    logger.info(
                        action,
                        logger.LogResult.success,
                        "messages-amount",
                        message_amount,
                    )
                    return
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "messages-amount", message_amount
            )
            raise e
        finally:
            client_socket.close()

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_socket)
