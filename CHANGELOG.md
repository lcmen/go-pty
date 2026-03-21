# Changelog

All notable changes to test improvements will be documented in this file.

## Unreleased

Add `-e` flag to load environment variables from a `.env` file with `${VAR}` expansion.
Add `-s` flag to run a subset of processes from the Procfile (e.g. `go-pty -s web,worker`).

## 0.1.1

Improvements for errors handling.

## 0.1.0

Public release.

## 0.0.4

Fix race conditions, cleanups and process monitoring.

## 0.0.3

Shutdown multiplexer when one of the processes crashes.

## 0.0.2

First release.
