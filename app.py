from flask import Flask, jsonify
from vehicles import vehicles_bp
from cranes import cranes_bp
from scores import scores_bp
from dotenv import load_dotenv


def create_app():
    app = Flask(__name__)
    app.register_blueprint(vehicles_bp)
    app.register_blueprint(cranes_bp)
    app.register_blueprint(scores_bp)

    load_dotenv()

    @app.route("/")
    def root():
        return jsonify({"message": "Welcome to the design 3 API."})

    return app


if __name__ == "__main__":
    app = create_app()

    app.logger.warning("Starting the app in development mode.")
    app.run(host="0.0.0.0", port=5000, debug=True)
