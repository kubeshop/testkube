FROM scratch
COPY kubectl-testkube /kubectl-testkube
USER 1001
EXPOSE 8088
CMD ["/kubectl-testkube"]
