STRATEGY ?= tournament
POOL ?= conservative
GENERATIONS ?= 0
TARGET ?= pi
WORKERS ?= 0

TARGET_GENETIC_SERIES = genetic_series
TARGET_EVAL = eval

.PHONY: build test clean run tools release
default: release

build:
	go build -o $(TARGET_GENETIC_SERIES) .
	go build -o $(TARGET_EVAL) ./cmd/eval/

test: build
	go test ./...

clean:
	rm -f $(TARGET_EVAL)
	rm -f $(TARGET_GENETIC_SERIES)
	rm -f *.tex *.pdf *.aux *.log

run: build
	GOMAXPROCS=$(WORKERS) ./genetic_series -target $(TARGET) -strategy $(STRATEGY) -pool $(POOL) -population 1000 -generations $(GENERATIONS) -maxterms 512 -seed 0 -workers $(WORKERS)

release: clean build test

tools:
	@if [ "$$(uname)" = "Darwin" ]; then \
		if ! command -v pdflatex >/dev/null 2>&1; then \
			echo "Installing pdflatex via MacTeX..."; \
			brew install --cask mactex-no-gui; \
		else \
			echo "pdflatex already installed"; \
		fi \
	elif [ "$$(uname)" = "Linux" ]; then \
		if ! command -v pdflatex >/dev/null 2>&1; then \
			if command -v apt-get >/dev/null 2>&1; then \
				echo "Installing pdflatex via texlive (apt-get)..."; \
				sudo apt-get update && sudo apt-get install -y texlive-latex-base texlive-latex-extra; \
			elif command -v yum >/dev/null 2>&1; then \
				echo "Installing pdflatex via texlive (yum)..."; \
				sudo yum install -y texlive texlive-latex; \
			else \
				echo "Error: No supported package manager found (apt-get or yum)"; \
				echo "Please manually install pdflatex for your distribution"; \
				exit 1; \
			fi \
		else \
			echo "pdflatex already installed"; \
		fi \
	else \
		echo "Unsupported OS: $$(uname)"; \
		exit 1; \
	fi
