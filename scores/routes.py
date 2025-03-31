import os
from flask import Blueprint, jsonify, request
from storage import Storage

scores_bp = Blueprint("scores", __name__, url_prefix="/scores")

scoreboard_password = os.environ.get("SCOREBOARD_PASSWORD")


@scores_bp.route("/")
def get_scores():
    if not is_authorized():
        return jsonify({"error": "Unauthorized"}), 401

    return jsonify(Storage().scoreboard)


@scores_bp.route("/", methods=["POST"])
def set_scores():
    if not request.is_json:
        return jsonify({"error": "Content type is not json"}), 400

    if not is_authorized():
        return jsonify({"error": "Unauthorized"}), 401

    content = request.get_json()
    try:
        for vehicle_score in content["vehicle_scores"]:
            vehicle_id = vehicle_score["id"]
            points = vehicle_score["points"]

            Storage().scoreboard.set_vehicle_score(vehicle_id, points)

        for crane_score in content["crane_scores"]:
            crane_id = crane_score["id"]
            points = crane_score["points"]

            Storage().scoreboard.set_crane_score(crane_id, points)
    except:
        return jsonify({"error": "Invalid request"}), 400

    return jsonify(Storage().scoreboard)


def is_authorized():
    if scoreboard_password is None:
        return False

    password = request.headers.get("Authorization")
    return password == scoreboard_password
