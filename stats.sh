#!/bin/sh

echo "Test $(go test -cover ./gopty/ | grep -oE 'coverage: [0-9.]+%')"

echo "\nLines of code (excluding comments and blanks):"
total=0
for f in gopty/*.go cmd/main.go; do
  n=$(grep -cvE '^\s*(//|$)' "$f")
  total=$((total + n))
  printf "%8d %s\n" "$n" "$f"
done
printf "%8d total\n" "$total"
