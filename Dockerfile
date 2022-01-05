FROM alpine:latest

RUN apk update && \
    apk upgrade && \
    apk --no-cache add ca-certificates && \
    rm -rf /var/cache/apk/*
WORKDIR /code
USER 1001
COPY bin/linux/kops-autoscaler-openstack .
ENTRYPOINT ["/code/kops-autoscaler-openstack"]
