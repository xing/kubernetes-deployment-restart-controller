FROM golang:1.20

WORKDIR /src/

COPY go.* /src/
RUN go mod download

COPY . /src/

COPY ./testdata/.kube/config /root/.kube/config

# We need to disable cgo support, otherwise images built on scratch will fail with this error message:
# standard_init_linux.go:195: exec user process caused "no such file or directory"
ENV CGO_ENABLED=0
RUN make clean \
  && make test \
  && make

FROM scratch
COPY --from=0 /src/kubernetes-deployment-restart-controller /usr/bin/kubernetes-deployment-restart-controller
ENTRYPOINT ["/usr/bin/kubernetes-deployment-restart-controller"]
