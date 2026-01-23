import os
import json
import logging
from flask import jsonify

TOKEN = os.environ["HEARTBEAT_TOKEN"]

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(levelname)s %(message)s",
)

def heartbeat(request):
    if request.method != "POST":
        return ("Method Not Allowed", 405)

    if request.headers.get("X-Heartbeat-Token") != TOKEN:
        return ("Unauthorized", 401)

    data = request.get_json(silent=True) or {}

    service = data.get("service")
    if not service:
        return (jsonify({"error": "service is required"}), 400)

    env = data.get("env", "unknown")
    info = data.get("info")

    logging.info(json.dumps({
        "type": "heartbeat",
        "service": str(service),
        "env": str(env),
        "info": info,
    }, ensure_ascii=False))

    return (jsonify({"ok": True}), 200)
