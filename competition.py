import io
import importlib.util
import os
import sys
from pathlib import Path

import matplotlib
from flask import Blueprint, Response, jsonify

matplotlib.use("Agg")

competition_bp = Blueprint("competition", __name__, url_prefix="/competition")

_interface_instance = None

scoreboard_password = os.environ.get("SCOREBOARD_PASSWORD")

def _load_interface_class():
    interface_path = Path(__file__).resolve().parent / "design3-competition-interface" / "interface.py"
    interface_dir = interface_path.parent
    if str(interface_dir) not in sys.path:
        sys.path.insert(0, str(interface_dir))
    spec = importlib.util.spec_from_file_location("competition_interface", interface_path)
    if spec is None or spec.loader is None:
        raise RuntimeError("Unable to load competition interface module.")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module.CompetitionInterface


def _get_interface_instance():
    global _interface_instance

    if _interface_instance is None:
        interface_class = _load_interface_class()
        _interface_instance = interface_class(scoreboard_password=scoreboard_password)
    return _interface_instance


@competition_bp.route("/", methods=["GET"])
def get_competition_frame():
    try:
        interface = _get_interface_instance()
        fig = interface.draw()
    except Exception as exc:
        return jsonify({"error": str(exc)}), 500

    buffer = io.BytesIO()
    fig.savefig(buffer, format="png", dpi=150, bbox_inches="tight", pad_inches=0)
    buffer.seek(0)
    return Response(buffer.getvalue(), mimetype="image/png")
