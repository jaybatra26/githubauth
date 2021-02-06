FROM golang:1.14.3-alpine AS build
WORKDIR /src
COPY . .
RUN export CGO_ENABLED=0  && go build  -o /out/githubauth  ./cmd/githubauth

FROM scratch AS bin
COPY .env .env
COPY --from=build /out/githubauth /
ENTRYPOINT ["./githubauth"]