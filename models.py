from dataclasses import dataclass


@dataclass
class Crane:
    id: int
    nb_tokens: int


@dataclass
class Vehicle:
    id: int
    path: list[str]


@dataclass
class Score:
    id: int
    points: int


@dataclass
class Scoreboard:
    vehicle_scores: list[Score]
    crane_scores: list[Score]

    def get_team_score(self, team_id: int) -> int:
        if team_id < 7:
            return self.crane_scores[team_id - 1].points
        else:
            return self.vehicle_scores[team_id - 7].points

    def increment_team_score(self, team_id: int):
        if team_id < 7:
            self.crane_scores[team_id - 1].points += 1
        else:
            self.vehicle_scores[team_id - 7].points += 1

    def decrement_team_score(self, team_id: int):
        if team_id < 7:
            self.crane_scores[team_id - 1].points -= 1
        else:
            self.vehicle_scores[team_id - 7].points -= 1

    def set_vehicle_score(self, vehicle_id: int, points: int):
        for vehicle_score in self.vehicle_scores:
            if vehicle_score.id == vehicle_id:
                vehicle_score.points = points

    def set_crane_score(self, crane_id: int, points: int):
        for crane_score in self.crane_scores:
            if crane_score.id == crane_id:
                crane_score.points = points
