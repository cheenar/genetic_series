STRATEGY ?= tournament
POOL ?= conservative
GENERATIONS ?= 0
TARGET ?= pi
WORKERS ?= 0

.PHONY: build test clean run

build:
	go build -o genetic_series .

test:
	go test ./...

clean:
	rm -f *.tex *.pdf *.aux *.log

run: build
	GOMAXPROCS=$(WORKERS) ./genetic_series -target $(TARGET) -strategy $(STRATEGY) -pool $(POOL) -population 1000 -generations $(GENERATIONS) -maxterms 512 -seed 0 -workers $(WORKERS)
