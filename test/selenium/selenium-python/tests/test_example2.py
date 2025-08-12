from pages.lipsum_page import LipsumPage
import time

def test_example_2_1(driver):
    page = LipsumPage(driver)
    page.open()
    time.sleep(10)  # just to make the test longer
    assert page.get_title() == page.EXPECTED_TITLE

def test_example_2_2(driver):
    time.sleep(7)

def test_example_2_3(driver):
    time.sleep(10)