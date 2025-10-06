import time
import random

def test_EEE():
    time.sleep(3)
    if random.randint(1, 4) == 4:
        assert 0 == 1
    else:
        assert 1 == 1

def test_FFF():
    time.sleep(5)
    assert 1 == 1

def test_GGG():
    time.sleep(2)
    assert 1 == 2

def test_HHH():
    time.sleep(1)
    if random.randint(1, 10) == 10:
        assert 0 == 1
    else:
        assert 1 == 1
