FROM golang
ADD . /go/src/github.com/PelionIoT/devicedb
RUN cd src/github.com/PelionIoT/devicedb && go install
ENTRYPOINT [ "/bin/sh", "/go/src/github.com/PelionIoT/devicedb/startup-kube.sh" ]
EXPOSE 8080
EXPOSE 9090