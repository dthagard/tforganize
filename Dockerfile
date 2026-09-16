# Copyright 2024 Dan Thagard
#
# Licensed under the MIT license (the "License"); you may not
# use this file except in compliance with the License.
#
# You may obtain a copy of the License at the LICENSE file in
# the root directory of this source tree.

FROM golang:1.27.1-alpine3.24 AS builder

ARG VERSION=dev
RUN apk add --update --no-cache make git

WORKDIR /go/src/tforganize

COPY . .
RUN VERSION=${VERSION} make all

################

FROM alpine:3.24.1

RUN apk add --no-cache git

COPY --from=builder /go/src/tforganize/bin/tforganize /usr/local/bin/

ENTRYPOINT ["tforganize"]