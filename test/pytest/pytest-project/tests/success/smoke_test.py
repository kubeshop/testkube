import time
import pytest

def test_1():
    assert 1 == 1

def test_2():
    time.sleep(3)
    assert 1 == 1

def test_3():
    assert 1 == 1

def test_4():
    time.sleep(2)
    assert 1 == 1

@pytest.mark.skip(reason="Work in progress")
def test_5():
    time.sleep(1)
    assert 1 == 1

@pytest.mark.skip(reason="Work in progress")
def test_6():
    time.sleep(5)
    assert 1 == 1
