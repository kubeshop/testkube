*** Keywords ***
Open Homepage
    New Page    ${BASE_URL}
    Wait For Elements State    h1    visible    timeout=30s
