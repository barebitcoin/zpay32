test: 
    go list -f '{{{{.Dir}}/...' -m | xargs go test

tidy: 
    go list -f '{{{{.Dir}}' -m | xargs -L1 go mod tidy -C