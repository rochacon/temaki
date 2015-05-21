FROM golang:1.4
RUN go get -v golang.org/x/tools/cmd/cover

ENV PACKAGE_NAME github.com/rochacon/temaki
WORKDIR /go/src/$PACKAGE_NAME
ENTRYPOINT ["/go/bin/temaki"]

COPY . /go/src/$PACKAGE_NAME
RUN go get -v $PACKAGE_NAME/... && go install -v $PACKAGE_NAME 
