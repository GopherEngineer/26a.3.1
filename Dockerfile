FROM golang:latest AS compiling
RUN mkdir -p /go/src/pipeline
WORKDIR /go/src/pipeline
ADD main.go .
ADD go.mod .
RUN go install .
 
FROM scratch
LABEL version="1.0.0"
LABEL maintainer="Gopher Engineer<gopher@skillfactory>"
WORKDIR /root/
COPY --from=compiling /go/bin/pipeline .
CMD [ "./pipeline" ]