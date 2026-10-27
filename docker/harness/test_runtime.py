"""Exercise the real pinned SDK/runtime against a local model and callback fixture."""
import json
import os
import socket
import tempfile
import threading
import time
import unittest
import urllib.request
from unittest.mock import patch
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from deepseek_harness import DeepSeekHarness
from bridge import app, patch_file
import uvicorn


class RuntimeTest(unittest.TestCase):
    def test_tool_round_trip_and_tool_allowlist(self):
        requests, callbacks = [], []

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                if self.path in ("/callback", "/api/v1/aiops/harness/tools"):
                    callbacks.append((self.headers.get("Authorization"), body))
                    data = json.dumps({"status": "succeeded", "result": "fixture-container running"}).encode()
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(data)))
                    self.end_headers()
                    self.wfile.write(data)
                    return
                requests.append(body)
                completed = any(block.get("type") == "tool_result" for message in body["messages"] for block in message.get("content", []) if isinstance(block,dict))
                events = [{"type":"message_start","message":{"id":"msg_fixture","model":"deepseek-v4-flash","usage":{"input_tokens":50,"output_tokens":0}}}]
                if completed:
                    events += [{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}, {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"The container is running."}}]
                else:
                    events += [{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_fixture_001","name":"docker_list","input":{}}},{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":'{"connectionId":7}'}}]
                events += [{"type":"content_block_stop","index":0},{"type":"message_delta","delta":{"stop_reason":"end_turn" if completed else "tool_use"},"usage":{"output_tokens":10}},{"type":"message_stop"}]
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.end_headers()
                for event in events:
                    self.wfile.write(("event: " + event["type"] + "\ndata: " + json.dumps(event) + "\n\n").encode())
                self.wfile.flush()

        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory() as directory:
                overlay = patch_file(Path(directory))
                url = f"http://127.0.0.1:{server.server_port}"
                with DeepSeekHarness(dsh_home=directory, cwd=directory, profile="sdk-minimal", patches=(str(overlay),), provider="deepseek-official", model="deepseek-v4-flash", base_url=url, api_key="fixture", request_timeout_seconds=45, env={"AUTOOPS_CALLBACK_URL": url + "/callback", "AUTOOPS_CALLBACK_TOKEN": "fixture-ticket", "DSH_SYSTEM_PROMPT": "Inspect connection 7 using docker_list."}) as harness:
                    result = harness.run("Inspect the container.")
                self.assertEqual(result.finish_reason, "completed", json.dumps(result.events[-5:]))
                self.assertIn("running", result.final_response)
            self.assertEqual(len(callbacks), 1)
            self.assertEqual(callbacks[0][0], "Bearer fixture-ticket")
            self.assertEqual(callbacks[0][1]["connectionId"], 7)
            names = {tool["name"] for tool in requests[0]["tools"]}
            self.assertEqual(names, {"docker_info", "docker_list", "docker_logs", "docker_start", "docker_stop", "docker_restart"})
            # Exercise the shipped HTTP adapter as well as the direct SDK API.
            sock = socket.socket()
            sock.bind(("127.0.0.1", 0))
            bridge_port = sock.getsockname()[1]
            bridge_server = uvicorn.Server(uvicorn.Config(app, log_level="critical", access_log=False))
            bridge_thread = threading.Thread(target=bridge_server.run, kwargs={"sockets": [sock]}, daemon=True)
            secret = "fixture-shared-secret-" + "x" * 32
            with patch.dict(os.environ, {"AIOPS_HARNESS_SECRET": secret, "AUTOOPS_CALLBACK_URL": url}):
                bridge_thread.start()
                try:
                    deadline = time.monotonic() + 5
                    while not bridge_server.started and time.monotonic() < deadline:
                        time.sleep(0.05)
                    self.assertTrue(bridge_server.started)
                    data = {"model": "deepseek-v4-flash", "baseUrl": url, "apiKey": "fixture", "timeout": 30,
                            "messages": [{"role": "user", "content": "Inspect connection 7"}],
                            "targets": "[7]", "actions": "", "callbackUrl": url + "/api/v1/aiops/harness/tools", "callbackToken": "a" * 64}
                    request = urllib.request.Request(f"http://127.0.0.1:{bridge_port}/run", data=json.dumps(data).encode(), headers={"Authorization": "Bearer " + secret, "Content-Type": "application/json"})
                    with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(request, timeout=40) as response:
                        events = [json.loads(line) for line in response]
                    self.assertEqual(events[-1]["type"], "done", events)
                    self.assertTrue(any(e.get("text") == "The container is running." for e in events))
                    self.assertEqual(len(callbacks), 2)
                finally:
                    bridge_server.should_exit = True
                    bridge_thread.join(5)
                    sock.close()
        finally:
            server.shutdown()
            server.server_close()
            thread.join()


if __name__ == "__main__":
    unittest.main()
