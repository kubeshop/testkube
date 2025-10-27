*** Settings ***
Resource        ../resources/settings.robot
Resource        ../resources/keywords/homepage_keywords.robot

Suite Setup     New Browser    headless=True
Suite Teardown  Close Browser

*** Test Cases ***
Homepage Title Test
    New Page    ${BASE_URL}
    Wait For Elements State    h1    visible    timeout=30s
    Get Text    h1    ==    Testkube test page - Lipsum

Homepage Title Test - keywords
    Open Homepage
    Get Text    h1    ==    Testkube test page - Lipsum

Homepage test 3
    Sleep    5s
    Should Be Equal    1    1

Homepage test 4
    Sleep    2s
    Should Be Equal    1    1

Homepage test 5
    Sleep    5s
    Should Be Equal    1    1