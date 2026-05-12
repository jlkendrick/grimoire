import sys
import requests
import cowsay

def tell_random_joke(message: str):
    # Fetch a random joke to prove 'requests' works
    response = requests.get("https://official-joke-api.appspot.com/random_joke")
    joke = response.json()
    full_message = f"{message}\n\n{joke['setup']} ... {joke['punchline']}"
    
    # Print it using 'cowsay' to prove the second dependency works
    cowsay.cow(full_message)


def fetch_joke():
    r = requests.get("https://official-joke-api.appspot.com/random_joke", timeout=5)
    r.raise_for_status()
    j = r.json()
    return j["setup"], j["punchline"]


def cow_the_joke(setup: str, punchline: str):
    msg = f"{setup}\n\n   ... {punchline}"
    return cowsay.get_output_string("cow", msg)