FROM golang:1.25 AS builder

WORKDIR /go/src/github.com/fi-ts/gardener-extension-monitoring-fits
COPY . .
RUN make install \
 && strip /go/bin/gardener-extension-monitoring-fits

FROM alpine:3.22
WORKDIR /
COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-monitoring-fits /gardener-extension-monitoring-fits
CMD ["/gardener-extension-monitoring-fits"]
