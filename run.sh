rm -rf exported/ && \
  go vet && \
  go build && \
  ./export-vbulletin
