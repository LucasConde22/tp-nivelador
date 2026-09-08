import socket
from threading import Condition, Lock, Thread
import logger
from lottery import Lottery
from .bets_protocol import BetsProtocol, MSG_TYPE_BET, MSG_TYPE_REQUEST_WINNERS, MSG_TYPE_MULTI_BETS

THREADS_TIMEOUT_TIME = 3

STORAGE_PATH = "./bets.csv"
ACTION_HANDLE_CLIENT = "handle-client"
ACTION_ACCEPT_CONNECTION = "accept-connection"
LOG_FIELD_MESSAGES_AMOUNT = "messages-amount"
MSG_INVALID_BATCH = 'invalid or incomplete batch'

class Server:
    def __init__(self, server_host: str, server_port: int, agency_quorum_min: int, storage_path: str = STORAGE_PATH) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.lottery = Lottery(storage_path)
        self.agency_quorum_min = agency_quorum_min

        self.quorum_condition = Condition()
        self.lottery_lock = Lock()
        self.agencies_ready = 0
        self.quorum_reached = False

        self.running = False
        self.server_socket = None
        self.client_sockets = set()
        self.client_threads = []
        self.clients_lock = Lock()
        self.shutdown_lock = Lock()

    def _handle_client(self, client_socket):
        """
        Handles the communication with a connected client, processing bets and requests for winners.
        """
        message_amount = 0
        try:
            logger.info(ACTION_HANDLE_CLIENT, logger.LogResult.in_progress)
            message_amount = self.process_bets(client_socket, message_amount)
            if self.running:
                logger.info(
                    ACTION_HANDLE_CLIENT,
                    logger.LogResult.success,
                    LOG_FIELD_MESSAGES_AMOUNT,
                    message_amount,
                )
        except Exception as e:
            if self.running:
                logger.error(
                    ACTION_HANDLE_CLIENT,
                    logger.LogResult.fail,
                    LOG_FIELD_MESSAGES_AMOUNT,
                    message_amount,
                    "err",
                    e,
                )
        finally:
            with self.clients_lock:
                self.client_sockets.discard(client_socket)
            try:
                client_socket.close()
            except OSError:
                pass

    def process_bets(self, client_socket, message_amount):
        """
        Processes incoming bets and requests for winners from a connected client, storing bets and
        sending winners.
        """
        protocol = BetsProtocol(client_socket)
        agency_id = None

        while True:
            msg_type, data = protocol.receive_message()
            if msg_type is None:
                break

            message_amount += 1

            if msg_type == MSG_TYPE_BET or msg_type == MSG_TYPE_MULTI_BETS:
                if not data or not isinstance(data, list):
                    logger.error(ACTION_HANDLE_CLIENT, logger.LogResult.fail, "reason", MSG_INVALID_BATCH)
                    break
                agency_id = data[0].agency_id
                self._store_bets(data)
                protocol.send_ack() # Let's the client know that all bets were processed

            elif msg_type == MSG_TYPE_REQUEST_WINNERS:
                self._wait_for_quorum()
                if not self.running:
                    break
                self._send_winners(agency_id, protocol)
                break

        return message_amount

    def _store_bets(self, bets):
        """
        Stores the provided bets in the lottery.
        """
        with self.lottery_lock:
            self.lottery.store_bets(bets)

    def _send_winners(self, agency_id, protocol):
        """
        Sends the winners for the specified agency to the client.
        """
        winners = []
        with self.lottery_lock:
            for bet in self.lottery.load_bets():
                if self.lottery.has_won(bet) and bet.agency_id == agency_id:
                    winners.append(bet)
        for bet in winners:
            protocol.send_winner(bet)

    def _wait_for_quorum(self):
        """
        Waits for the quorum of agencies to be reached.
        """
        with self.quorum_condition:
            self.agencies_ready += 1
            if self.agencies_ready >= self.agency_quorum_min:
                self.quorum_reached = True
                self.quorum_condition.notify_all()

            while not self.quorum_reached and self.running:
                self.quorum_condition.wait()

    def run(self):
        self.running = True
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            self.server_socket = server_socket
            while self.running:
                try:
                    logger.info(ACTION_ACCEPT_CONNECTION, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    if not self.running:
                        break
                    logger.error(ACTION_ACCEPT_CONNECTION, logger.LogResult.fail)
                    raise e
                logger.info(ACTION_ACCEPT_CONNECTION, logger.LogResult.success)
                hilo = Thread(target=self._handle_client, args=(client_socket,))
                with self.clients_lock:
                    self.client_sockets.add(client_socket)
                    self.client_threads.append(hilo)
                hilo.start()

        self.stop()

    def stop(self):
        with self.shutdown_lock:
            if not self.running:
                return
            self.running = False

            with self.quorum_condition:
                self.quorum_condition.notify_all()

            if self.server_socket:
                try:
                    self.server_socket.close()
                except Exception:
                    pass

            with self.clients_lock:
                for client_socket in list(self.client_sockets):
                    try:
                        client_socket.shutdown(socket.SHUT_RDWR)
                    except Exception:
                        pass
                    try:
                        client_socket.close()
                    except Exception:
                        pass
                self.client_sockets.clear()

            with self.clients_lock:
                threads = list(self.client_threads)
            for thread in threads:
                thread.join(timeout=THREADS_TIMEOUT_TIME)