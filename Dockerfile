##############################
###### STAGE: BUILD     ######
##############################
FROM golang:1.14-alpine AS build-env

WORKDIR /src

COPY ./ ./

RUN go mod download && go mod vendor
RUN GOARCH=amd64 GOOS=linux CGO_ENABLED=0  go build -trimpath -o /contentfulproxy

##############################
###### STAGE: PACKAGE   ######
##############################
FROM alpine:3.11

ENV CONTENTFULPROXY_SERVER_ADDR=0.0.0.0:80
ENV LOG_JSON=1

RUN apk add --update --no-cache ca-certificates curl bash && rm -rf /var/cache/apk/*

COPY --from=build-env /contentfulproxy /usr/sbin/contentfulproxy

ENTRYPOINT ["/usr/sbin/contentfulproxy"]

CMD ["-webserver-address=$CONTENTFULPROXY_SERVER_ADDR"]

EXPOSE 80
# Zap
EXPOSE 9100
# Prometheus
EXPOSE 9200
# Viper
EXPOSE 9300
