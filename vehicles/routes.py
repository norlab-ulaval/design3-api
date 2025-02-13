from flask import Blueprint, request, jsonify
from constants import NB_VEHICLES, VEHICULE_ID_OFFSET
from models import Vehicle

vehicles_bp = Blueprint("vehicles", __name__, url_prefix="/vehicles")

vehicles: list[Vehicle] = [Vehicle(x + VEHICULE_ID_OFFSET, []) for x in range(NB_VEHICLES)]


@vehicles_bp.route("/")
def get_vehicles():
    response = {"vehicles": [vehicle for vehicle in vehicles]}

    return jsonify(response)


@vehicles_bp.route("/<int:vehicle_id>")
def get_vehicle(vehicle_id: int):
    if is_vehicle_id_invalid(vehicle_id):
        return jsonify({"error": "vehicle_id is invalid"}), 400

    vehicle = vehicles[vehicle_id - VEHICULE_ID_OFFSET]

    return jsonify(vehicle)


@vehicles_bp.route("/<int:vehicle_id>", methods=["POST"])
def set_vehicle(vehicle_id: int):
    if not request.is_json:
        return jsonify({"error": "Content type is not json"}), 400

    if is_vehicle_id_invalid(vehicle_id):
        return jsonify({"error": "vehicle_id is invalid"}), 400

    content = request.get_json()

    try:
        path = content["path"]
        # Check if path is a list of string
        if not isinstance(path, list) or not all(isinstance(x, str) for x in path):
            raise ValueError

        path = [str(x) for x in content["path"]]
    except:
        return jsonify({"error": "path must be a list of string"}), 400

    vehicle = vehicles[vehicle_id - VEHICULE_ID_OFFSET]
    vehicle.path = path

    return jsonify(vehicle)


def is_vehicle_id_invalid(team_id: int):
    return team_id < VEHICULE_ID_OFFSET or team_id > NB_VEHICLES + VEHICULE_ID_OFFSET - 1
