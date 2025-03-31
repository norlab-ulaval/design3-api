from constants import NB_VEHICLES, VEHICULE_ID_OFFSET, NB_CRANES
from models import Score, Scoreboard, Vehicle, Crane


class Storage:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(Storage, cls).__new__(cls)
            cls._instance.reset()
        return cls._instance

    def reset(self):
        self.vehicles: list[Vehicle] = [
            Vehicle(x + VEHICULE_ID_OFFSET, []) for x in range(NB_VEHICLES)
        ]
        self.scoreboard = Scoreboard(
            vehicle_scores=[
                Score(id=i + VEHICULE_ID_OFFSET, points=0) for i in range(NB_VEHICLES)
            ],
            crane_scores=[Score(id=i + 1, points=0) for i in range(NB_CRANES)],
        )
        self.cranes: list[Crane] = [Crane(x + 1, 0) for x in range(NB_CRANES)]
