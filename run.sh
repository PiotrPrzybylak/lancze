#!/usr/bin/env bash

cd server/cmd
go build
cd ../..
export DATABASE_URL=postgres://dbuser:dbpassword@localhost:5432/lancze?sslmode=disable
server/cmd/cmd