# DIII 2025 -- Networking

This is the documentation for the API used to communicate between the vehicles and the cranes.

## Install

### Native

1. Make sure to have python 3.12 (Might work on other versions)

```sh
pip install -r requirements.txt
python app.py
```

### With docker

```sh
docker build . --target dev -t design3-api
docker run -p 5000:5000 design3-api
```

## API

### GET `/`

Validate that the API is running.

- Response:
  - **200** OK
    ```json
    {
      "message": "Welcome to the design 3 API."
    }
    ```

### GET `/cranes`

Get the number of tokens on each scale. The id corresponds to the team id (AKA ZC\<id\>)

- Response:
  - **200** OK
    ```json
    {
      "cranes": [
        {
          "id": 1,
          "nb_tokens": 0
        },
        {
          "id": 2,
          "nb_tokens": 0
        }
      ]
    }
    ```

### GET `/cranes/:id`

Get the number of tokens on a scale.

- Response:

  - **200** OK

    ```json
    {
      "id": 3,
      "nb_tokens": 0
    }
    ```

  - **400** BAD_REQUEST
    ```json
    {
      "error": "crane_id is invalid"
    }
    ```

### POST `/cranes/:id`

Update the number of merchandise on a scale.

- Body:

  ```json
  {
    "nb_tokens": 3
  }
  ```

- Response:
  - **200** OK
  - **400** BAD_REQUEST\
     String: (invalid crane_id, invalid JSON, invalid nb_tokens)

### GET `/vehicles`

Get the objective of every vehicles. The id corresponds to the team id (AKA ZD\<id\>)

- Response:
  - **200** OK
    ```json
    {
      "vehicles": [
        {
          "going_to": 0,
          "id": 1
        },
        {
          "going_to": 0,
          "id": 2
        }
      ]
    }
    ```

### GET `/vehicles/:id`

Get the objective of a vehicle.

- Response:

  - **200** OK

    ```json
    {
      "going_to": 0,
      "id": 3
    }
    ```

  - **400** BAD_REQUEST
    ```json
    {
      "error": "vehicle_id is invalid"
    }
    ```

### POST `/vehicles/:id`

Update the objective of a vehicle.

- Body:

  ```json
  {
    "going_to": 2
  }
  ```

- Response:
  - **200** OK
  - **400** BAD_REQUEST\
     String: (invalid vehicle_id, invalid JSON, invalid going_to)
