FROM --platform=linux/amd64 docker.io/library/alpine/git:v2.54.0
COPY --from=docker.io/library/golang:1.27-alpine3.24 /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:$PATH
RUN go install golang.org/x/tools/gopls@v0.24.0
