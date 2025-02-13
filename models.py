from dataclasses import dataclass


@dataclass
class Crane:
    id: int
    nb_tokens: int


@dataclass
class Vehicle:
    id: int
    path: list[str]
