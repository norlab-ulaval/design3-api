from time import sleep
import requests

LOADING_ZONE_ID = 7  # ZD7
BASE_PATH = f"http://localhost:5000"

if __name__ == "__main__":
    # 1 Hz refresh loop
    while True:
        # Get info from all vehicles
        vehicles_response = requests.get(f"{BASE_PATH}/vehicles")
        print("============VEHICLES INFO============")
        print(vehicles_response.json())

        # Get info from all cranes
        cranes_response = requests.get(f"{BASE_PATH}/cranes")
        print("============CRANES INFO============")
        print(cranes_response.json())

        # Choose path
        path = ["ZD7", "A1", "ZC1"]

        # Notify the server of the path
        requests.post(f"{BASE_PATH}/vehicles/{LOADING_ZONE_ID}", json={"path": path})

        sleep(1)
