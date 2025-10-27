*** Settings ***
Resource        ../resources/settings.robot

Suite Setup     New Browser    headless=True
Suite Teardown  Close Browser

*** Test Cases ***

Test Case 6
    Sleep    2s
    Should Be Equal    1    1

Test Case 7
    Sleep    10s
    Should Be Equal    1    1

Test Case 8
    Sleep    2s
    Should Be Equal    1    1

Test Case 9
    Sleep    3s
    Should Be Equal    1    1

Test Case 10
    Sleep    4s
    Should Be Equal    1    1