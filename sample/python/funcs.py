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