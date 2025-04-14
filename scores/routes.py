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


@scores_bp.route("/increment/<int:team_id>", methods=["POST"])
def increment_team_score(team_id: int):
    if not is_authorized():
        return jsonify({"error": "Unauthorized"}), 401

    if team_id > 14 or team_id < 0:
        return jsonify({"error": "team_id is invalid"}), 400

    Storage().scoreboard.increment_team_score(team_id)

    return jsonify(Storage().scoreboard)


@scores_bp.route("/decrement/<int:team_id>", methods=["POST"])
def decrement_team_score(team_id: int):
    if not is_authorized():
        return jsonify({"error": "Unauthorized"}), 401

    if team_id > 14 or team_id < 0:
        return jsonify({"error": "team_id is invalid"}), 400

    if Storage().scoreboard.get_team_score(team_id) == 0:
        return jsonify({"error": "team score is already 0"}), 400
    Storage().scoreboard.decrement_team_score(team_id)

    return jsonify(Storage().scoreboard)


def is_authorized():
    if scoreboard_password is None:
        return False

    password = request.headers.get("Authorization")
    return password == scoreboard_password
