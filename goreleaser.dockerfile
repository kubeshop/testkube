FROM scratch
COPY testkube-api /testkube-api
USER 1001
EXPOSE 8088
CMD ["/testkube-api"]
