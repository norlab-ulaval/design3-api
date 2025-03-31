from flask import Blueprint, request, jsonify
from constants import NB_CRANES
from storage import Storage

cranes_bp = Blueprint("cranes", __name__, url_prefix="/cranes")


@cranes_bp.route("/")
def get_cranes():
    response = {"cranes": [crane for crane in Storage().cranes]}

    return jsonify(response)


@cranes_bp.route("/<int:crane_id>")
def get_crane(crane_id: int):
    if is_crane_id_invalid(crane_id):
        return jsonify({"error": "crane_id is invalid"}), 400

    crane = Storage().cranes[crane_id - 1]

    return jsonify(crane)


@cranes_bp.route("/<int:crane_id>", methods=["POST"])
def set_cranes(crane_id: int):
    if not request.is_json:
        return jsonify({"error": "Content type is not json"}), 400

    if is_crane_id_invalid(crane_id):
        return jsonify({"error": "crane_id is invalid"}), 400

    content = request.get_json()

    nb_tokens = 0
    try:
        nb_tokens = int(content["nb_tokens"])
        if nb_tokens < 0:
            raise Exception()
    except:
        return jsonify({"error": "nb_tokens is invalid"}), 400

    crane = Storage().cranes[crane_id - 1]
    crane.nb_tokens = nb_tokens

    return jsonify(crane)


def is_crane_id_invalid(team_id: int):
    return team_id <= 0 or team_id > NB_CRANES
