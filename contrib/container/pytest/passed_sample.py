import pytest
import requests

from requests.exceptions import ConnectionError

def is_responsive():
    try:
        response = requests.get("https://testkube.io")
        if response.status_code == 200:
            return True
    except ConnectionError:
        return False

def test_status():
    assert is_responsive()
