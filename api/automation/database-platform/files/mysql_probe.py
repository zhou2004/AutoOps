"""Role-sensitive HAProxy health check. No AutoOps API dependency."""
import http.server
import json
import sys
import pymysql

config = json.load(open(sys.argv[1], encoding="utf-8"))

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        healthy = False
        try:
            with pymysql.connect(**{k: config[k] for k in ("host", "port", "user", "password")}, connect_timeout=2, read_timeout=2, write_timeout=2, cursorclass=pymysql.cursors.DictCursor) as conn:
                with conn.cursor() as cursor:
                    cursor.execute("SELECT @@global.read_only AS ro, @@global.super_read_only AS sro")
                    state = cursor.fetchone()
                    if self.path == "/primary":
                        cursor.execute("SHOW SLAVE STATUS")
                        healthy = not state["ro"] and not state["sro"] and cursor.fetchone() is None
                    elif self.path == "/replica":
                        cursor.execute("SHOW SLAVE STATUS")
                        replica = cursor.fetchone() or {}
                        lag = replica.get("Seconds_Behind_Master")
                        healthy = state["ro"] and state["sro"] and replica.get("Slave_IO_Running") == "Yes" and replica.get("Slave_SQL_Running") == "Yes" and lag is not None and lag <= config["lag"]
        except Exception:
            pass
        self.send_response(200 if healthy else 503)
        self.end_headers()
        self.wfile.write(b"healthy\n" if healthy else b"unavailable\n")

    def log_message(self, *_):
        pass

http.server.ThreadingHTTPServer((config["host"], config["listen"]), Handler).serve_forever()
