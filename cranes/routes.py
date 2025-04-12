from flask import Blueprint, request, jsonify
from models import Crane
from storage import Storage

cranes_bp = Blueprint("cranes", __name__, url_prefix="/cranes")


@cranes_bp.route("/")
def get_cranes():
    response = {"cranes": [crane for crane in Storage().cranes]}

    return jsonify(response)


@cranes_bp.route("/<int:loading_zone_id>")
def get_crane(loading_zone_id: int):
    if is_loading_zone_id_invalid(loading_zone_id):
        return jsonify({"error": "loading_zone_id is invalid"}), 400

    crane = get_crane_by_loading_zone_id(loading_zone_id)
    if crane is None:
        return jsonify({"error": "crane not found"}), 404

    return jsonify(crane)


@cranes_bp.route("/<int:loading_zone_id>", methods=["POST"])
def set_cranes(loading_zone_id: int):
    if not request.is_json:
        return jsonify({"error": "Content type is not json"}), 400

    if is_loading_zone_id_invalid(loading_zone_id):
        return jsonify({"error": "loading_zone_id is invalid"}), 400

    crane = get_crane_by_loading_zone_id(loading_zone_id)
    if crane is None:
        crane = Crane(loading_zone_id, 0)
        Storage().cranes.append(crane)

    content = request.get_json()

    nb_tokens = 0
    try:
        nb_tokens = int(content["nb_tokens"])
        if nb_tokens < 0:
            raise Exception()
    except:
        return jsonify({"error": "nb_tokens is invalid"}), 400

    crane.nb_tokens = nb_tokens

    return jsonify(crane)


def get_crane_by_loading_zone_id(loading_zone_id: int) -> Crane | None:
    return next((crane for crane in Storage().cranes if crane.id == loading_zone_id), None)


def is_loading_zone_id_invalid(loading_zone_id: int) -> bool:
    return loading_zone_id <= 0 or loading_zone_id >= 100
