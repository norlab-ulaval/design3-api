from time import sleep
import requests
import numpy as np

TEAM_ID = 1
URL = f"http://localhost:5000/cranes/{TEAM_ID}"

if __name__ == "__main__":
    # 1 Hz refresh loop
    while True:
        # Compute the number of tokens on the scale
        nb_tokens_on_scale = np.random.randint(0, 10)

        # Notify the server of the number of tokens on the scale
        requests.post(URL, json={"nb_tokens": nb_tokens_on_scale})

        print(f"Number of tokens on the scale: {nb_tokens_on_scale}")

        sleep(1)
