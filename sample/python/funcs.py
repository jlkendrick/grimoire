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
	return f"Output21: {input1}", f"Output22: {input2}"


if __name__ == "__main__":
	hello_world()