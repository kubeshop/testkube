from pages.lipsum_page import LipsumPage
import time

def test_example_1_1(driver):
    page = LipsumPage(driver)
    page.open()
    assert page.get_title() == 'Testkube test page - Lorem Ipsum'

def test_example_1_2(driver):
    time.sleep(5)