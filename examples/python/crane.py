import os
import random
from time import sleep
import requests

LOADING_ZONE_ID = 1  # ZC1
BASE_URL = os.environ.get("BASE_URL", "http://localhost:5000")
URL = f"{BASE_URL}/cranes/{LOADING_ZONE_ID}"

if __name__ == "__main__":
    # 1 Hz refresh loop
    while True:
        # Compute the number of tokens on the scale
        nb_tokens_on_scale = random.randint(0, 9)

        # Notify the server of the number of tokens on the scale
        requests.post(URL, json={"nb_tokens": nb_tokens_on_scale})

        print(f"Number of tokens on the scale: {nb_tokens_on_scale}")

        sleep(1)
