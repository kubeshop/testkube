import time

def test_AAA():
    assert 1 == 2

def test_BBB():
    time.sleep(3)
    assert 1 == 1

def test_CCC():
    assert 1 == 2

def test_DDD():
    time.sleep(2)
    assert 1 == 1
