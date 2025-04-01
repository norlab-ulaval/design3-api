# API Design 3 - Ville Portuaire Intelligente

Voici l'implémentation de l'API de design 3 pour la ville portuaire intelligente. Il s'agit d'une API en Python sous Flask qui permet aux sous-systèmes grues et véhicules de communiquer ensemble.

## Installation

### Natif

1. S'assurer d'avoir python 3.12 (Peut-être fonctionnel avec d'autres versions, mais pas testé)

```sh
pip install -r requirements.txt
python app.py
```

2. Le serveur devrait être démarré en mode développement sur [http://localhost:5000](http://localhost:5000)

### Avec Docker

1. S'assurer d'avoir Docker

```sh
docker build . --target dev -t design3-api
docker run -p 5000:5000 design3-api
```

2. Le serveur devrait être démarré en mode développement sur [http://localhost:5000](http://localhost:5000)

## Exemples

Pour votre utilisation de l'API, vous pouvez vous inspirer des scripts dans le répertoire [./examples](./examples/). Il y a un script d'exemple pour chaque sous-système.

## Déploiement

1. Le serveur est déployé et accessible à l'adresse suivante: [http://172.105.22.140/](http://172.105.22.140/). Voici les commandes utilisés pour le déployer en mode production:

```
./build.bash
./launch.bash
```

## Routes de l'API

### GET `/`

Page pour valider que l'API est bien démarrée

- Response:
  - **200** OK
    ```json
    {
      "message": "Welcome to the design 3 API."
    }
    ```

### GET `/cranes`

Route pour obtenir l'information sur tous les sous-systèmes grues. Le champ `id` correspond à l'id de la zone de chargement (i.e. ZC\<id\>). Le champ `nb_tokens` correspond au nombre de marchandise actuellement sur la balance.

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

Route pour obtenir l'information sur une équipe du sous-système grue spécifique.

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

Route utilisé pour mettre à jour le nombre de marchandise sur la balance pour les équipes du sous-système grue.

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

Route pour obtenir l'information sur tous les sous-systèmes véhicules. Le champ `id` correspond au numéro d'une équipe et correspond à l'id de la zone de dépôt (i.e. ZD\<id\>). Le champ `path` correspond au chemin actuel suivi par le véhicule.

- Response:
  - **200** OK
    ```json
    {
      "vehicles": [
        {
          "path": ["ZD7", "A1", "ZC1"],
          "id": 7
        },
        {
          "path": ["ZD8", "A2", "ZC2"],
          "id": 8
        }
      ]
    }
    ```

### GET `/vehicles/:id`

Route pour obtenir l'information sur une équipe du sous-système véhicule spécifique.

- Response:

  - **200** OK

    ```json
    {
      "path": ["ZD7", "A1", "ZC1"],
      "id": 7
    }
    ```

  - **400** BAD_REQUEST
    ```json
    {
      "error": "vehicle_id is invalid"
    }
    ```

### POST `/vehicles/:id`

Route utilisé pour mettre à jour le chemin planifié par les équipes du sous-système véhicule. Le chemin doit commencer par le dernier panneau détecté et se terminer par une zone de chargement `ZC<id>` ou une zone de dépôt `ZD<id>`.

- Body:

  ```json
  {
    "path": ["ZD7", "A1", "ZC1"]
  }
  ```

- Response:
  - **200** OK
  - **400** BAD_REQUEST\
     String: (invalid vehicle_id, invalid JSON, invalid path)
