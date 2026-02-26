# Copyright 2024 Dan Thagard
#
# Licensed under the MIT license (the "License"); you may not
# use this file except in compliance with the License.
#
# You may obtain a copy of the License at the LICENSE file in
# the root directory of this source tree.

FROM golang:1.23-alpine AS builder

RUN apk add --update --no-cache make

WORKDIR /go/src/tforganize

COPY . .
RUN make all

################

FROM alpine:3.19.1

RUN apk add --no-cache git

COPY --from=builder /go/src/tforganize/bin/tforganize /usr/local/bin/

ENTRYPOINT ["tforganize"]