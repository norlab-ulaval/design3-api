from flask import Flask, jsonify, render_template, request
from storage import Storage
from vehicles import vehicles_bp
from cranes import cranes_bp
from scores import scores_bp
from dotenv import load_dotenv
import os

scoreboard_password = os.environ.get("SCOREBOARD_PASSWORD")


def create_app():
    app = Flask(__name__)
    app.register_blueprint(vehicles_bp)
    app.register_blueprint(cranes_bp)
    app.register_blueprint(scores_bp)

    load_dotenv()

    @app.route("/")
    def root():
        return render_template(
            "index.html",
            vehicles=Storage().vehicles,
            cranes=filter(lambda c: c.nb_tokens > 0, Storage().cranes),
        )

    @app.route("/reset", methods=["POST"])
    def reset():
        if not is_authorized():
            return jsonify({"error": "Unauthorized"}), 401
        Storage().reset()
        return jsonify({"message": "API reset was successful."})

    @app.route("/input_scores")
    def input_scores():
        return render_template("input_scores.html")

    @app.route("/is_authorized")
    def is_authorized_route():
        if is_authorized():
            return jsonify({"status": "Ok"}), 200
        else:
            return jsonify({"status": "Unauthorized"}), 401

    return app


def is_authorized():
    if scoreboard_password is None:
        return False

    password = request.headers.get("Authorization")
    return password == scoreboard_password


if __name__ == "__main__":
    app = create_app()

    app.logger.warning("Starting the app in development mode.")
    app.run(host="0.0.0.0", port=5000, debug=True)
