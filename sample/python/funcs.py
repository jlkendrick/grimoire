import time

def hello_world(n: int = 1):
	# print(f"Hell{"o" * n}, World!")

	return f"This is a return value: {n}"


def print_logs():
	time.sleep(1)
	print("This is a log")
	time.sleep(1)
	print("This is another log")
	time.sleep(1)
	print("This is a third log")

def test_pipe(input1: str, input2: str):
	return input1, input2

def test_pipey(input1: str, input2: str) -> str:
	return input1 + input2

def no_th_default(input1 = 1, input2 = 2):
	return input1 + input2

def mixed_defaults(name: str = "Alice", age: int = 30, score: float = 1.5, active: bool = True):
	return f"{name=} {age=} {score=} {active=}"


import random

_THEMES = {
	"noir": (
		["a rain-slick alley", "the back booth of an all-night diner", "a fog-bound pier", "the lobby of the Ambassador Hotel"],
		["uneasy", "resigned", "sharp as cheap gin", "watchful"],
	),
	"fantasy": (
		["the moss-walled archive of Vael", "a market under three moons", "the bone bridge at Lirien", "an inn at the edge of the world"],
		["expectant", "hush-bright", "wind-bitten", "ceremonial"],
	),
	"cosmic": (
		["the observation deck of the Kepler-22 relay", "a derelict listening post", "the crystal gardens of Io", "a chapel orbiting a dead star"],
		["dimly hopeful", "vacuum-quiet", "iridescent", "geometrically wrong"],
	),
}

def seed_world(seed: int, theme: str = "noir"):
	settings, moods = _THEMES.get(theme, _THEMES["noir"])
	rng = random.Random(seed)
	return rng.choice(settings), rng.choice(moods)

def compose_opening(setting: str, hero: str, villain: str, mcguffin: str, style: str = "terse"):
	if style == "florid":
		return f"In {setting}, {hero} weighed the matter of {mcguffin} — and across the room, {villain} was already counting the cost."
	if style == "telegram":
		return f"{hero.upper()} STOP {setting.upper()} STOP {villain.upper()} HAS {mcguffin.upper()} STOP ADVISE"
	# terse (default)
	return f"{hero} walked into {setting}. {villain} had {mcguffin}, and only one of them was leaving with it."


def map_list_output() -> dict:
	return {
		"a": {
			"b": {
				"c": 4,
			},
		},
	}

def func(n: int) -> int:
	return n


def classify_temperature(celsius: int):
	if celsius >= 30:
		return {"category": "hot", "freezing": False}
	if celsius <= 0:
		return {"category": "freezing", "freezing": True}
	return {"category": "mild", "freezing": False}

def warn_frostbite():
	return "wear gloves, watch your fingers"

def recommend_shorts():
	return "leave the jacket at home"