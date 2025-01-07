#!/bin/bash

set -xe

go build cmd/server/server.go
go build cmd/client/client.go
