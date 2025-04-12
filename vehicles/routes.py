from flask import Blueprint, request, jsonify
from models import Vehicle
from storage import Storage

vehicles_bp = Blueprint("vehicles", __name__, url_prefix="/vehicles")


@vehicles_bp.route("/")
def get_vehicles():
    response = {"vehicles": [vehicle for vehicle in Storage().vehicles]}

    return jsonify(response)


@vehicles_bp.route("/<int:unloading_zone_id>")
def get_vehicle(unloading_zone_id: int):
    if is_unloading_zone_id_invalid(unloading_zone_id):
        return jsonify({"error": "unloading_zone_id is invalid"}), 400

    vehicle = get_vehicle_by_unloading_zone_id(unloading_zone_id)
    if vehicle is None:
        return jsonify({"error": "vehicle not found"}), 404

    return jsonify(vehicle)


@vehicles_bp.route("/<int:unloading_zone_id>", methods=["POST"])
def set_vehicle(unloading_zone_id: int):
    if not request.is_json:
        return jsonify({"error": "Content type is not json"}), 400

    if is_unloading_zone_id_invalid(unloading_zone_id):
        return jsonify({"error": "unloading_zone_id is invalid"}), 400

    vehicle = get_vehicle_by_unloading_zone_id(unloading_zone_id)
    if vehicle is None:
        vehicle = Vehicle(unloading_zone_id, [])
        Storage().vehicles.append(vehicle)

    content = request.get_json()

    try:
        path = content["path"]
        # Check if path is a list of string
        if not isinstance(path, list) or not all(isinstance(x, str) for x in path):
            raise ValueError

        path = [str(x) for x in content["path"]]
    except:
        return jsonify({"error": "path must be a list of string"}), 400

    vehicle.path = path

    return jsonify(vehicle)


def get_vehicle_by_unloading_zone_id(id: int):
    return next((vehicle for vehicle in Storage().vehicles if vehicle.id == id), None)


def is_unloading_zone_id_invalid(unloading_zone_id: int):
    return unloading_zone_id <= 0 or unloading_zone_id >= 100
