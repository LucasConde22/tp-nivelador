import socket
import logger
from lottery import Lottery
from .bets_protocol import BetsProtocol, MSG_TYPE_BET, MSG_TYPE_REQUEST_WINNERS

STORAGE_PATH = "./bets.csv"
ACTION_HANDLE_CLIENT = "handle-client"
ACTION_ACCEPT_CONNECTION = "accept-connection"
LOG_FIELD_MESSAGES_AMOUNT = "messages-amount"

class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str = STORAGE_PATH) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.storage_path = storage_path
        self.lottery = Lottery(self.storage_path)

    def _handle_client(self, client_socket):
        message_amount = 0
        try:
            logger.info(ACTION_HANDLE_CLIENT, logger.LogResult.in_progress)
            message_amount = self.process_bets(client_socket, message_amount)
            logger.info(
                ACTION_HANDLE_CLIENT,
                logger.LogResult.success,
                LOG_FIELD_MESSAGES_AMOUNT,
                message_amount,
            )
        except Exception as e:
            logger.error(
                ACTION_HANDLE_CLIENT,
                logger.LogResult.fail,
                LOG_FIELD_MESSAGES_AMOUNT,
                message_amount,
            )
            raise e
        finally:
            client_socket.close()

    def process_bets(self, client_socket, message_amount):
        protocol = BetsProtocol(client_socket)
        agency_id = None

        while True:
            msg_type, data = protocol.receive_message()
            if msg_type is None:
                break

            message_amount += 1

            if msg_type == MSG_TYPE_BET:
                agency_id = data.agency_id
                self.lottery.store_bets([data])
            elif msg_type == MSG_TYPE_REQUEST_WINNERS:
                for bet in self.lottery.load_bets():
                    if self.lottery.has_won(bet) and (agency_id is None or bet.agency_id == agency_id):
                        protocol.send_winner(bet)
                break

        return message_amount

    def run(self):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(ACTION_ACCEPT_CONNECTION, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(ACTION_ACCEPT_CONNECTION, logger.LogResult.fail)
                    raise e
                logger.info(ACTION_ACCEPT_CONNECTION, logger.LogResult.success)

                self._handle_client(client_socket)