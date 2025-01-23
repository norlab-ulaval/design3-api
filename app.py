from flask import Flask, jsonify
from vehicles import vehicles_bp
from cranes import cranes_bp

app = Flask(__name__)

app.register_blueprint(vehicles_bp)
app.register_blueprint(cranes_bp)

@app.route("/")
def root():
    return jsonify({"message": "Welcome to the design 3 API."})


if __name__ == '__main__':
    app.run(debug=True)