import io
import os
import sys
import threading
import time
from pathlib import Path

import matplotlib
from flask import Blueprint, Response, jsonify, render_template, request

matplotlib.use("Agg")

competition_bp = Blueprint("competition", __name__, url_prefix="/competition")

scoreboard_password = os.environ.get("SCOREBOARD_PASSWORD")

_interface_instance = None
_cached_png = None
_is_viz_locked = False

interface_dir = Path(__file__).resolve().parent / "design3-competition-interface"
if str(interface_dir) not in sys.path:
    sys.path.insert(0, str(interface_dir))
from interface import CompetitionInterface


def _get_interface_instance():
    global _interface_instance

    if _interface_instance is None:
        _interface_instance = CompetitionInterface(scoreboard_password=scoreboard_password)
    return _interface_instance


@competition_bp.route("/", methods=["GET"])
def competition_view():
    return render_template("competition.html")

@competition_bp.route("/lock_viz", methods=["GET"])
def lock_viz():
    if not is_authorized():
        return jsonify({"error": "Unauthorized"}), 401

    global _is_viz_locked
    _is_viz_locked = True

    return jsonify({"status": "locked"})

@competition_bp.route("/unlock_viz", methods=["GET"])
def unlock_viz():
    if not is_authorized():
        return jsonify({"error": "Unauthorized"}), 401

    global _is_viz_locked
    _is_viz_locked = False

    return jsonify({"status": "unlocked"})

@competition_bp.route("/image", methods=["GET"])
def get_competition_frame():
    global _cached_png, _cached_at

    if _is_viz_locked:
        return _cached_png

    # If we have the image in cached
    now = time.monotonic()
    if _cached_png is not None and (now - _cached_at) <= 3.0:
        return Response(_cached_png, mimetype="image/png")

    # If not, we regenerate it and cache it back
    try:
        interface = _get_interface_instance()
        fig = interface.draw()
    except Exception as exc:
        return jsonify({"error": str(exc)}), 500

    buffer = io.BytesIO()

    try:
        fig.savefig(buffer, format="png", dpi=150, bbox_inches="tight", pad_inches=0)
    finally:
        import matplotlib.pyplot as plt
        plt.close(fig)

    buffer.seek(0)
    _cached_png = buffer.getvalue()
    _cached_at = time.monotonic()

    return Response(_cached_png, mimetype="image/png")

def is_authorized():
    if scoreboard_password is None:
        return False

    password = request.headers.get("Authorization")
    return password == scoreboard_password
