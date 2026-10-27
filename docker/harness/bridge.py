"""AutoOps HTTP adapter for the official DeepSeek Harness Python SDK (not a Harness HTTP API)."""
import asyncio
import hmac
import json
import logging
import os
import queue
import tempfile
import threading
from pathlib import Path

from deepseek_harness import DeepSeekHarness
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import StreamingResponse

app = FastAPI(docs_url=None, redoc_url=None, openapi_url=None)
slots = asyncio.Semaphore(4)
PLUGIN = Path(__file__).with_name("autoops-tools.mjs").resolve()


def patch_file(home: Path) -> Path:
    # Pin row IDs to the sdk-minimal composition shipped in 0.2.1rc2.
    patch = [{"id": name, "disabled": True} for name in
             ("persistent-bash", "persistent-pwsh", "tool-fs", "tool-working-directory")]
    patch.append({"insert": [{"id": "autoops-tools", "name": str(PLUGIN)}]})
    target = home / "autoops.patch.json"
    target.write_text(json.dumps(patch), encoding="utf-8")
    return target


@app.get("/health")
async def health():
    return {"status": "ok", "sdk": "0.2.1rc2"}


@app.post("/run")
async def run(request: Request):
    secret = os.environ.get("AIOPS_HARNESS_SECRET", "")
    if len(secret) < 32 or not hmac.compare_digest(request.headers.get("authorization", ""), "Bearer " + secret):
        raise HTTPException(401, "Unauthorized")
    raw = bytearray()
    async for chunk in request.stream():
        raw.extend(chunk)
        if len(raw) > 1048576:
            raise HTTPException(413, "Request too large")
    try:
        body = json.loads(raw)
        timeout = max(10, min(600, int(body["timeout"])))
        if not isinstance(body["messages"], list) or len(body["messages"]) > 42:
            raise ValueError()
        if not all(isinstance(m.get("content"), str) and m.get("role") in ("system", "user", "assistant") for m in body["messages"]):
            raise ValueError()
        if not isinstance(body["model"], str) or not body["model"]:
            raise ValueError()
        if not isinstance(body["callbackToken"], str) or len(body["callbackToken"]) != 64:
            raise ValueError()
        if not body["baseUrl"].startswith(("https://", "http://")):
            raise ValueError()
        callback = os.environ["AUTOOPS_CALLBACK_URL"].rstrip("/") + "/api/v1/aiops/harness/tools"
        if body["callbackUrl"] != callback:
            raise ValueError()
    except (ValueError, KeyError, TypeError, AttributeError):
        raise HTTPException(400, "Invalid bridge request") from None

    async def stream():
        async with slots:
            events = queue.Queue(maxsize=256)
            stopped = threading.Event()
            holder = []

            def emit(value):
                while not stopped.is_set():
                    try:
                        events.put(value, timeout=0.1)
                        return
                    except queue.Full:
                        continue

            def worker():
                try:
                    with tempfile.TemporaryDirectory(prefix="autoops-harness-") as directory:
                        home = Path(directory)
                        patch = patch_file(home)
                        system = "\n".join(m["content"] for m in body["messages"] if m["role"] == "system")
                        system += "\nSelected connection IDs: " + body.get("targets", "[]")
                        system += "\nGranted action names (server validates expiry): " + body.get("actions", "")
                        prompt = json.dumps([m for m in body["messages"] if m["role"] != "system"], ensure_ascii=False)
                        prompt = "Continue this conversation. Earlier messages are context; respond to the final user message.\n" + prompt
                        harness = DeepSeekHarness(
                            dsh_home=str(home), cwd=str(home), profile="sdk-minimal",
                            patches=(str(patch),), provider="deepseek-official", model=body["model"],
                            base_url=body["baseUrl"], api_key=body.get("apiKey", ""),
                            request_timeout_seconds=timeout, initialize_timeout_seconds=30,
                            env={"DSH_SYSTEM_PROMPT": system, "AUTOOPS_CALLBACK_URL": callback,
                                 "AUTOOPS_CALLBACK_TOKEN": body["callbackToken"]},
                        )
                        holder.append(harness)
                        try:
                            if stopped.is_set():
                                return
                            emit({"type": "step", "text": "Harness 正在分析会话与授权工具"})

                            def notification(event):
                                payload = event.payload.get("event", {})
                                kind = payload.get("type", "")
                                # Only operational progress is relayed; private reasoning is not exposed.
                                if "tool" in kind:
                                    emit({"type": "tool", "text": "Harness 正在调用授权工具", "step": kind})

                            result = harness.run(prompt, on_notification=notification)
                            if result.finish_reason != "completed":
                                raise RuntimeError("Harness turn did not complete")
                            emit({"type": "delta", "text": result.final_response})
                            emit({"type": "done"})
                        finally:
                            harness.close()
                except Exception as error:
                    # SDK exceptions can contain provider payloads; log the type only.
                    logging.error("Harness turn failed: %s", type(error).__name__)
                    emit({"type": "error", "text": "Harness execution failed"})
                finally:
                    emit(None)

            thread = threading.Thread(target=worker, daemon=True)
            thread.start()
            try:
                while True:
                    try:
                        event = events.get_nowait()
                    except queue.Empty:
                        if await request.is_disconnected():
                            break
                        await asyncio.sleep(0.1)
                        continue
                    if event is None:
                        break
                    yield json.dumps(event, ensure_ascii=False) + "\n"
            finally:
                stopped.set()
                if holder:
                    await asyncio.to_thread(holder[0].close)
                await asyncio.to_thread(thread.join, 5)

    return StreamingResponse(stream(), media_type="application/x-ndjson", headers={"Cache-Control": "no-store", "X-Accel-Buffering": "no"})
