# Common makefile structure for buildable go projects that result in a docker container artifact
export SHELL:=/bin/bash
export SHELLOPTS:=$(if $(SHELLOPTS),$(SHELLOPTS):)pipefail:errexit

CWD=$(shell pwd)

.PHONY: buf.% lint generate

buf.lint:
	$(CWD)/buf.sh lint

buf.gen: buf.format
	rm -rf "$(CWD)/lib/schema/*"
	$(CWD)/buf.sh generate --template=go.buf.gen.yaml

buf.format:
	$(CWD)/buf.sh format ./proto/market-tracker -w

generate: buf.gen
lint: buf.lint
