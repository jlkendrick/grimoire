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