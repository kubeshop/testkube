*** Settings ***
Resource        ../resources/settings.robot

Suite Setup     New Browser    headless=True
Suite Teardown  Close Browser

*** Test Cases ***

Failing Test 1 - Wrong Title
    [Tags]    negative
    New Page    ${BASE_URL}
    Wait For Elements State    h1    visible    timeout=30s
    Get Text    h1    ==    This text does not exist

Failing Test 2 - Element Missing
    [Tags]    negative
    New Page    ${BASE_URL}
    Wait For Elements State    #missing-element    visible    timeout=5s

Failing Test 3 - Wrong Assertion
    [Tags]    negative
    Should Be Equal    1    2
