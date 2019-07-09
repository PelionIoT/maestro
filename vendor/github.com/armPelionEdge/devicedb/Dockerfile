FROM golang
ADD . /go/src/github.com/armPelionEdge/devicedb
RUN cd src/github.com/armPelionEdge/devicedb && go install
ENTRYPOINT [ "/bin/sh", "/go/src/github.com/armPelionEdge/devicedb/startup-kube.sh" ]
EXPOSE 8080
EXPOSE 9090