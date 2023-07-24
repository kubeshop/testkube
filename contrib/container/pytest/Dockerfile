FROM python:3.8.17-alpine3.18

RUN set -ex \
    && python3 -m pip install --upgrade --no-cache-dir pip \
    && python3 -m pip install \
        --no-cache-dir \
        --upgrade pipenv \
        --upgrade requests \
        --upgrade pytest
