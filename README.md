# API Design 3 - Ville Portuaire Intelligente

Voici l'API de design 3 pour la ville portuaire intelligente. Elle permet aux sous-systèmes grues et véhicules de communiquer ensemble : chaque équipe de grue publie le nombre de jetons sur sa balance, et chaque équipe de véhicule publie son chemin planifié et lit l'état des grues et des autres véhicules.

## Démarrage rapide

1. S'assurer d'avoir Docker (avec Docker Compose).
2. Depuis ce répertoire :

```sh
cp .env.example .env
docker compose up --build
```

3. Le serveur est accessible sur [http://localhost:5000](http://localhost:5000). La page d'accueil affiche les véhicules et les grues qui ont publié leurs données.

Pour l'arrêter, faire `Ctrl+C`, puis `docker compose down`. Les données sont conservées dans un volume Docker; `docker compose down -v` les supprime aussi.

## Exemples

Pour votre utilisation de l'API, vous pouvez vous inspirer des scripts dans le répertoire [./examples](./examples/). Il y a un script d'exemple pour chaque sous-système. Ils n'ont besoin que de Python 3 et du package `requests` (`pip install requests`).

```sh
python3 examples/python/crane.py     # une grue : publie un nombre de jetons à 1 Hz
python3 examples/python/vehicle.py   # un véhicule : lit les grues et les véhicules, publie son chemin à 1 Hz
```

Pour viser un autre serveur que `http://localhost:5000`, définir la variable d'environnement `BASE_URL` :

```sh
BASE_URL=http://192.168.1.50:5000 python3 examples/python/vehicle.py
```

Le répertoire [./tests](./tests/) contient aussi des scripts `curl` pour essayer les routes à la main. Ils utilisent la même variable `BASE_URL`.

```sh
./examples/bash/post_crane.bash 1 3     # la grue ZC1 publie 3 jetons
./examples/bash/post_vehicle.bash 7     # le véhicule ZD7 publie le chemin ["ZD7", "A1", "ZC1"]
./examples/bash/get_cranes.bash         # toutes les grues (ou ./tests/get_cranes.bash 1 pour une seule)
./examples/bash/get_vehicles.bash       # tous les véhicules (ou ./tests/get_vehicles.bash 7)
```

## Cycle de vie d'une équipe

La base de données démarre vide : aucune grue, aucun véhicule. **Chaque équipe doit publier (POST) dès le début, même si elle n'a rien à rapporter** :

- une grue dont la balance est vide publie `{"nb_tokens": 0}`;
- un véhicule qui n'a pas encore de chemin publie `{"path": []}`.

Le premier POST d'une équipe la crée. Ensuite, elle apparaît dans les GET (`/cranes`, `/cranes/:id`, `/vehicles`, `/vehicles/:id`) et sur la page d'accueil avec la dernière valeur publiée. Une équipe qui n'a jamais publié reçoit **404** sur son propre GET et est absente des listes et de la page d'accueil.

## Routes de l'API

Toutes les réponses JSON ont l'en-tête `Content-Type: application/json`. Toutes les erreurs ont la forme `{"error": "<message>"}`.

Erreurs communes à toutes les routes :

| Statut | Message | Cas |
|---|---|---|
| **404** | `not found` | Route inconnue ou identifiant invalide) |
| **405** | `method not allowed` | Mauvaise méthode HTTP pour cette route |
| **413** | `request body too large` | Corps de requête de plus de 64 Kio |
| **400** | `invalid request body` | Erreur dans le corp de la requête |
| **500** | `internal error` | Erreur interne du serveur |

### Identifiants de zone (`:id`)

- Les identifiants valides vont de 1 à 99.
- Un identifiant valide qui n'a jamais été publié donne un **404** sur le GET.

### GET `/cranes`

Route pour obtenir l'information sur toutes les grues qui ont publié au moins une fois. Le champ `id` correspond à l'id de la zone de chargement (i.e. ZC\<id\>). Le champ `nb_tokens` correspond au nombre de marchandises actuellement sur la balance. La liste est triée par `id` croissant.

- Réponse :
  - **200** OK
    ```json
    {
      "cranes": [
        { "id": 1, "nb_tokens": 0 },
        { "id": 2, "nb_tokens": 4 }
      ]
    }
    ```

### GET `/cranes/:id`

Route pour obtenir l'information sur une grue spécifique.

- Réponse :
  - **200** OK
    ```json
    { "id": 3, "nb_tokens": 0 }
    ```

### POST `/cranes/:id`

Route utilisée pour mettre à jour le nombre de marchandises sur la balance. Le premier POST crée la grue.

- Corps :
  ```json
  { "nb_tokens": 3 }
  ```
- Réponse :
  - **200** OK, avec la grue : `{"id": 3, "nb_tokens": 3}`

### GET `/vehicles`

Route pour obtenir l'information sur tous les véhicules qui ont publié au moins une fois. Le champ `id` correspond à l'id de la zone de dépôt (i.e. ZD\<id\>) associée au véhicule. Le champ `path` correspond au chemin actuel suivi par le véhicule. La liste est triée par `id` croissant.

- Réponse :
  - **200** OK
    ```json
    {
      "vehicles": [
        { "id": 7, "path": ["ZD7", "A1", "ZC1"] },
        { "id": 8, "path": [] }
      ]
    }
    ```

### GET `/vehicles/:id`

Route pour obtenir l'information sur un véhicule spécifique.

- Réponse :
  - **200** OK
    ```json
    { "id": 7, "path": ["ZD7", "A1", "ZC1"] }
    ```

### POST `/vehicles/:id`

Route utilisée pour mettre à jour le chemin planifié par les équipes du sous-système véhicule. Le premier POST crée le véhicule. Le chemin doit commencer par le dernier panneau détecté et se terminer par une zone de chargement `ZC<id>` ou une zone de dépôt `ZD<id>`.

- Corps :
  ```json
  { "path": ["ZD7", "A1", "ZC1"] }
  ```
- Réponse :
  - **200** OK, avec le véhicule : `{"id": 7, "path": ["ZD7", "A1", "ZC1"]}`

### GET `/`

Page d'accueil. Elle affiche le chemin de chaque véhicule et le nombre de jetons de chaque grue qui ont publié. La page ne se rafraîchit pas toute seule.