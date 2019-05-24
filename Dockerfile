FROM alpine:latest

RUN apk add --no-cache ca-certificates
WORKDIR /code
USER 1001
COPY bin/linux/kops-autoscaler-openstack .
ENTRYPOINT ["/code/kops-autoscaler-openstack"]
