from flask import Blueprint, request, jsonify
from constants import NB_VEHICLES
from models import Vehicle

vehicles_bp = Blueprint('vehicles', __name__, url_prefix='/vehicles')

vehicles: list[Vehicle] = [Vehicle(x+1, "") for x in range(NB_VEHICLES)]

@vehicles_bp.route("/")
def get_vehicles():
    response = {
        "vehicles": [vehicle for vehicle in vehicles]
    }

    return jsonify(response)


@vehicles_bp.route("/<int:vehicle_id>")
def get_vehicle(vehicle_id: int):
    if is_vehicle_id_invalid(vehicle_id):
        return jsonify({"error": "vehicle_id is invalid"}), 400

    vehicle = vehicles[vehicle_id-1]

    return jsonify(vehicle)


@vehicles_bp.route("/<int:vehicle_id>", methods=["POST"])
def set_vehicle(vehicle_id: int):
    if not request.is_json:
        return jsonify({"error": "Content type is not json"}), 400

    if is_vehicle_id_invalid(vehicle_id):
        return jsonify({"error": "vehicle_id is invalid"}), 400

    content = request.get_json()

    going_to = ""
    try:
        going_to = str(content['going_to']).upper()
    except:
        return jsonify({"error": "going_to must be a string"}), 400

    if len(going_to) <= 2:
        return jsonify({"error": "going_to must be longer than 2 characters (ZCxx or ZDxx)"}), 400

    if not going_to.startswith("ZD") and not going_to.startswith("ZC"):
        return jsonify({"error": "going_to must start with ZD or ZC"}), 400
    
    # TODO: Validate number after ZD or ZC

    vehicle = vehicles[vehicle_id-1]
    vehicle.going_to = going_to

    return jsonify(vehicle)


def is_vehicle_id_invalid(team_id: int):
    return team_id <= 0 or team_id > NB_VEHICLES